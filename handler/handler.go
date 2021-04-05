package handler

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/tubo28/yomiage/db"
	"github.com/tubo28/yomiage/discord"
	"github.com/tubo28/yomiage/worker"
	"mvdan.cc/xurls"
)

var (
	defaultTTSLang string = os.Getenv("DEFAULT_TTS_LANG")
)

func init() {
	if defaultTTSLang == "" {
		defaultTTSLang = "en-US"
	}
}

// Init adds handlers to discord
func Init() {
	discord.AddHandler(messageCreate)
	go CleanerWorkerEndless()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("recovered: ", err)
		}
	}()

	if strings.HasPrefix(m.Content, "!hi") {
		if m.Author.ID == s.State.User.ID {
			return
		}
		hiHandler(s, m)
		return
	}

	if strings.HasPrefix(m.Content, "!bye") {
		if m.Author.ID == s.State.User.ID {
			return
		}
		byeHandler(s, m)
		return
	}

	// if content starts with mention string to bot,
	prefixPatStr := fmt.Sprintf(`^\s*<@!?%s>`, s.State.User.ID) // '<@1234> ...' or '<@!1234> ...'
	prefixPat, err := regexp.Compile(prefixPatStr)
	if err != nil {
		log.Print("failed to compile mention prefix regexp: ", prefixPatStr)
	}
	if prefixPat != nil && prefixPat.MatchString(m.Content) {
		noMentionContent := strings.TrimSpace(prefixPat.ReplaceAllString(m.Content, ""))
		if noMentionContent == "" || noMentionContent == "help" {
			if m.Author.ID == s.State.User.ID {
				return
			}
			helpHandler(s, m)
			return
		}

		args := strings.Fields(noMentionContent)
		head, args := args[0], args[1:]
		if head == "lang" {
			if m.Author.ID == s.State.User.ID {
				return
			}
			langHandler(s, m, args)
			return
		}
		if head == "rand" {
			if m.Author.ID == s.State.User.ID {
				return
			}
			randHandler(s, m, args)
			return
		}
		return
	}

	if !strings.HasPrefix(m.Content, "!") {
		nonCommandHandler(s, m)
	}
}

func langHandler(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) == 0 {
		// get language
		lang, err := db.GetUserLanguage(m.Author.ID)
		if err != nil {
			log.Print("error get user ", m.Author.ID, "'s language: ", err.Error())
			return
		}
		// User %s's language is %s.
		msg := fmt.Sprintf("%s の読み上げ言語は %s です。", nick(s, m.GuildID, m.Author), lang)
		if _, err := s.ChannelMessageSend(m.ChannelID, msg); err != nil {
		}
	} else {
		// set language
		lang := args[0]
		if err := db.UpsertUserLanguage(m.Author.ID, lang); err != nil {
			log.Print("error update user ", m.Author.ID, "'s language to ", lang, ": ", err.Error())
			return
		}
		// User %s's language is updated to %s
		msg := fmt.Sprintf("%s の読み上げ言語を %s に変更しました。", nick(s, m.GuildID, m.Author), lang)
		if _, err := s.ChannelMessageSend(m.ChannelID, msg); err != nil {
		}
	}
}

func randHandler(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// generate random token and set for user
	u, _ := uuid.NewUUID()
	vt := u.String()
	err := db.UpsertUserVoiceToken(m.Author.ID, vt)
	if err != nil {
		log.Print("error update user voice token ", m.Author.ID, "'s token: ", err.Error())
		return
	}

	// update voice token
	if _, err := s.ChannelMessageSend(m.ChannelID, nick(s, m.GuildID, m.Author)+" の声を変更しました。"); err != nil {
	}

	// play sample voice
	var lang string
	lang, err = db.GetUserLanguage(m.Author.ID)
	if err != nil {
		log.Print("error get user "+m.Author.ID+"'s langage: ", err)
	}
	if lang == "" {
		lang = defaultTTSLang
	}

	ci, ok := consumers.Load(m.GuildID)
	if ok {
		c := ci.(*ttsConsumerBinding)
		// Sample: hello
		text := "サンプル: イカよろしく～"
		c.consumer.Add(worker.Task{
			ID: fmt.Sprintf("Read %s in guild %s", text, m.GuildID),
			Do: func() error {
				err := discord.Play(text, lang, vt, m.GuildID)
				time.Sleep(100 * time.Millisecond)
				return err
			},
		})
	}
}

func nick(s *discordgo.Session, guildID string, m *discordgo.User) string {
	if member, err := s.State.Member(guildID, m.ID); err == nil && member.Nick != "" {
		return member.Nick
	}
	return m.Username
}

type ttsConsumerBinding struct {
	guildID        string
	voiceChannelID string // VC to send voice
	textChannelID  string // TC to read
	consumer       *worker.Consumer
	Cancel         func() // func to stop consumer
}

// maps guildID to ttsConsumerBinding to read
// also works as flag whether the bot is working on a guild
var consumers sync.Map

func hiHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	authorID := m.Author.ID
	guildID := m.GuildID

	// Bot is not working on this guild?
	if ci, ok := consumers.Load(guildID); ok {
		c := ci.(*ttsConsumerBinding)
		log.Printf("bot is already joining voice channel %s of this guild %s", c.voiceChannelID, m.GuildID)
		ch, err := s.State.GuildChannel(guildID, m.ChannelID)
		if err != nil {
			log.Printf("error find guild %s channel %s", guildID, m.ChannelID)
			return
		}
		// Already working on other channel
		msg := fmt.Sprintf("すでにこのサーバーのボイスチャンネル %s を読み上げ中です。", ch.Name)
		if _, err := s.ChannelMessageSend(m.ChannelID, msg); err != nil {
			log.Print("error send message to channel ", m.ChannelID, " on guild ", guildID, ": ", err)
		}
		return
	}

	// The command author is joining in a voice channel?
	userVs, err := discord.VoiceState(s, authorID, guildID)
	if err != nil {
		log.Printf("failed to get VoiceState of guild %s: %s", guildID, err.Error())
		return
	}
	if userVs == nil {
		log.Printf("member %s is not joining any voice channel", authorID)
		// Cannot summon the bot without joining any voice channel
		if _, err := s.ChannelMessageSend(m.ChannelID, "ボイスチャンネルに参加せずに呼び出すことはできません。"); err != nil {
			log.Print("error send message to channel ", m.ChannelID, " on guild ", guildID, ": ", err)
		}
		return
	}

	// Ok, then start worker
	ctx, cancel := context.WithCancel(context.Background())
	wg := new(sync.WaitGroup) // now wg is not used. we can deleted it
	consumer := worker.NewConsumer(guildID)
	consumers.Store(guildID, &ttsConsumerBinding{
		guildID:        m.GuildID,
		voiceChannelID: userVs.ChannelID,
		textChannelID:  m.ChannelID,
		consumer:       consumer, // consumer should have cancel and wg should?
		Cancel:         cancel,
	})

	consumer.StartAsync(ctx, wg)

	time.Sleep(200 * time.Millisecond) // waiting for bot to join voice channel
	if msg := discord.JoinVC(s, m.GuildID, userVs.ChannelID); msg != "" {
		if _, err := s.ChannelMessageSend(m.ChannelID, msg); err != nil {
			log.Print("error send message to channel ", m.ChannelID, " on guild ", guildID, ": ", err)
		}
	}
}

func byeHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	userID := m.Author.ID
	botID := s.State.User.ID
	guildID := m.GuildID

	// Bot is working on this guild?
	ci, ok := consumers.Load(guildID)
	if !ok {
		log.Print("not working on this guild ", guildID)
		// Not working on this guild
		if _, err := s.ChannelMessageSend(m.ChannelID, "現在このサーバーでは読み上げていません。"); err != nil {
			log.Print("error send message to channel ", m.ChannelID, " on guild ", guildID, ": ", err)
		}
		return
	}

	c := ci.(*ttsConsumerBinding)

	// Text channel on which the commend was post is the working text channel of bot?
	if m.ChannelID != c.textChannelID {
		log.Printf("member %s is not joining voice channel bot is reading %s", userID, c.voiceChannelID)
		thisCh, err := s.State.GuildChannel(guildID, m.ChannelID)
		if err != nil {
			log.Printf("error find guild %s channel %s", guildID, m.ChannelID)
			return
		}
		wrkCh, err := s.State.GuildChannel(guildID, c.textChannelID)
		if err != nil {
			log.Printf("error find guild %s channel %s", guildID, m.ChannelID)
			return
		}
		msg := fmt.Sprintf("このテキストチャンネル %s は読み上げていません。%s を読み上げ中です。", thisCh.Name, wrkCh.Name)
		if _, err := s.ChannelMessageSend(m.ChannelID, msg); err != nil {
			log.Print("error send message to channel ", m.ChannelID, " on guild ", guildID, ": ", err)
		}
		return
	}

	// The command author is joining the working voice channel of bot?
	userVs, err := discord.VoiceState(s, userID, guildID)
	if err != nil {
		log.Printf("failed to get VoiceState of guild %s: %s", guildID, err.Error())
		return
	}
	if userVs == nil {
		log.Printf("member %s is not joining any voice channel", userID)
		// Cannot summon me without joining any voice channel
		if _, err := s.ChannelMessageSend(m.ChannelID, "ボイスチャンネルに参加せずに呼び出すことはできません。"); err != nil {
			log.Print("error send message to channel ", m.ChannelID, " on guild ", guildID, ": ", err)
		}
		return
	}
	if userVs.ChannelID != c.voiceChannelID {
		log.Printf("member %s is not joining voice channel bot is reading %s", botID, c.voiceChannelID)
		wrkCh, err := s.State.GuildChannel(guildID, c.textChannelID)
		if err != nil {
			log.Printf("error find guild %s channel %s", guildID, m.ChannelID)
			return
		}
		msg := fmt.Sprintf("読み上げ中のボイスチャンネル %s に参加せずに読み上げを止めることはできません。", wrkCh.Name)
		if _, err := s.ChannelMessageSend(m.ChannelID, msg); err != nil {
			log.Print("error send message to channel ", m.ChannelID, " on guild ", guildID, ": ", err)
		}
		return
	}

	// Ok, then stop worker
	consumers.Delete(m.GuildID)
	c.Cancel()
	if msg := discord.LeaveVC(m.GuildID); msg != "" {
		if _, err := s.ChannelMessageSend(m.ChannelID, msg); err != nil {
			log.Print("error send message to channel ", m.ChannelID, " on guild ", m.GuildID, ": ", err)
		}
	}
}

func helpHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Usage: ...
	msg := "使い方: https://github.com/tubo28/yomiage/blob/main/README.md"
	if _, err := s.ChannelMessageSend(m.ChannelID, msg); err != nil {
		log.Print("error send message to channel ", m.ChannelID, " on guild ", m.GuildID, ": ", err)
	}
}

const maxTTSLength = 50

func nonCommandHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	ci, ok := consumers.Load(m.GuildID)
	if !ok {
		log.Printf("not working on this guild %s. message is ignored", m.GuildID)
		return
	}
	c := ci.(*ttsConsumerBinding)
	if m.ChannelID != c.textChannelID {
		log.Printf("bot is working but not reading this text channel%s. message is ignored", m.ChannelID)
		return
	}

	lang, err := db.GetUserLanguage(m.Author.ID)
	if err != nil {
		log.Print("error get user "+m.Author.ID+"'s langage: ", err)
	}
	if lang == "" {
		lang = defaultTTSLang
	}

	vt, err := db.GetUserVoiceToken(m.Author.ID)
	if err != nil {
		log.Print("error get user "+m.Author.ID+"'s voice token: ", err)
	}
	if vt == "" {
		vt = m.Author.ID
	}

	// TODO: trim ogg files by time
	text := replaceMention(s, m)
	text = Sanitize(text, lang)
	if textR := []rune(text); len(textR) > maxTTSLength {
		text = string(textR[:maxTTSLength]) + " 以下略" // following is omitted
	}

	c.consumer.Add(worker.Task{
		ID: fmt.Sprintf("Read %s in guild %s", text, m.GuildID),
		Do: func() error {
			err := discord.Play(text, lang, vt, m.GuildID)
			time.Sleep(100 * time.Millisecond)
			return err
		},
	})
}

var (
	urlReg    = xurls.Relaxed
	ignoreReg = regexp.MustCompile("^[(（)].*[）)]$")
	kusaReg   = regexp.MustCompile("[wWｗＷ]+$")

	patternChannels = regexp.MustCompile("<#[^>]*>") // <#12345>
)

// Port of ContentWithMoreMentionsReplaced in DiscordGo
// https://github.com/bwmarrin/discordgo/blob/cbfa831b6c2dc48e550f89821640c6c7c03e1aa8/message.go#L404
// Differences:
// - don't skip non-mentionable role
// - add emoji replacement
func replaceMention(s *discordgo.Session, m *discordgo.MessageCreate) (content string) {
	content = m.Content

	if !s.StateEnabled {
		content = m.ContentWithMentionsReplaced()
		return
	}

	channel, err := s.State.Channel(m.ChannelID)
	if err != nil {
		content = m.ContentWithMentionsReplaced()
		return
	}

	for _, user := range m.Mentions {
		nick := user.Username

		println(user.ID)
		println(nick)

		member, err := s.State.Member(channel.GuildID, user.ID)
		if err == nil && member.Nick != "" {
			nick = member.Nick
		}

		content = strings.NewReplacer(
			"<@"+user.ID+">", "@"+user.Username,
			"<@!"+user.ID+">", "@"+nick,
		).Replace(content)
	}

	for _, roleID := range m.MentionRoles {
		role, err := s.State.Role(channel.GuildID, roleID)
		// modify: don't skip non-mentionable role
		// continues also if !role.Mentionable in original DiscordGo
		// but omit it.
		if err != nil {
			continue
		}
		content = strings.Replace(content, "<@&"+role.ID+">", "@"+role.Name, -1)
	}

	content = patternChannels.ReplaceAllStringFunc(content, func(mention string) string {
		channel, err := s.State.Channel(mention[2 : len(mention)-1])
		if err != nil || channel.Type == discordgo.ChannelTypeGuildVoice {
			return mention
		}

		return "#" + channel.Name
	})

	// modify: add emoji replacement
	guild, err := s.State.Guild(m.GuildID)
	if err != nil {
		return
	}

	for _, e := range guild.Emojis {
		// skip if not contains suffix of emoji string for performance
		// :1234>
		if !strings.Contains(content, ":"+e.ID+">") {
			continue
		}

		ps := "<:[^>:]+:" + e.ID + ">" // <:emoji_name:1234>
		pattern, err := regexp.Compile(ps)
		if err != nil {
			log.Print("failed to compile emoji pattern: ", ps)
			continue
		}

		content = pattern.ReplaceAllString(content, ":"+e.Name+":")
	}

	return
}

// Sanitize modifies m.Content easier to read for bot in following steps:
// 1. trim spaces
// 2. replace continuous 'w's to kusa
// 3. replace URL to "URL"
// 4. replace continuous whitespaces to single one
func Sanitize(content, lang string) string {
	s := strings.TrimSpace(content)

	// 1
	if ignoreReg.MatchString(s) {
		return ""
	}

	// 2
	if (strings.HasPrefix(lang, "ja-") || lang == "ja") && kusaReg.MatchString(s) {
		s = kusaReg.ReplaceAllString(s, " くさ")
	}

	// 3
	s = urlReg.ReplaceAllString(s, " URL ")

	// 4
	s = strings.Join(strings.Fields(s), " ")

	return s
}

// CleanerWorker starts worker which clean up workers which is alone on voice channels
func CleanerWorker() {
	log.Print("start cleaner worker")
	for {
		time.Sleep(10 * time.Second)
		cs := []*ttsConsumerBinding{}
		consumers.Range(func(k interface{}, v interface{}) bool {
			cs = append(cs, v.(*ttsConsumerBinding))
			return true
		})
		for _, c := range cs {
			time.Sleep(1 * time.Second)
			if discord.Alone(c.guildID) {
				log.Printf("bot is alone in voice channel on guild %s, leave", c.guildID)
				consumers.Delete(c.guildID)
				c.Cancel()
				_ = discord.LeaveVC(c.guildID)
			}
		}
	}
}

// CleanerWorkerEndless calls CleanerWorker and call it again if it panics
func CleanerWorkerEndless() {
	for {
		func() {
			defer func() {
				if err := recover(); err != nil {
					log.Println("CleanerWorker paniced unexpectedly: ", err)
				}
			}()
			CleanerWorker()
		}()
	}
}
