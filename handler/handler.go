package handler

import (
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
		// todo
		// if head == "rand" {
		// 	if m.Author.ID == s.State.User.ID {
		// 		return
		// 	}
		// 	randHandler(s, m, args)
		// 	return
		// }
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
	// Randomized your voice.
	if _, err := s.ChannelMessageSend(m.ChannelID, nick(s, m.GuildID, m.Author)+" の声を変更しました。"); err != nil {
	}
	var lang string
	lang, err = db.GetUserLanguage(m.Author.ID)
	if err != nil {
		log.Print("error get user "+m.Author.ID+"'s langage: ", err)
	}
	if lang == "" {
		lang = defaultTTSLang
	}
	worker.AddTask(m.GuildID, worker.TTSTask{
		GuildID:    m.GuildID,
		Text:       "サンプル、イカよろしく～", // This is test
		Lang:       lang,
		VoiceToken: vt,
	})
}

func nick(s *discordgo.Session, guildID string, m *discordgo.User) string {
	if member, err := s.State.Member(guildID, m.ID); err == nil && member.Nick != "" {
		return member.Nick
	}
	return m.Username
}

type channelBinding struct {
	voiceChannelID string
	textChannelID  string
}

// maps guildID to channelBinding to read
// also works as flag whether the bot is working on a guild
var bindings sync.Map

func hiHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	authorID := m.Author.ID
	guildID := m.GuildID

	// Bot is not working on this guild?
	if cid, ok := bindings.Load(guildID); ok {
		log.Printf("bot is already joining voice channel %s this guild %s", cid, m.GuildID)
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
	bindings.Store(guildID, channelBinding{
		voiceChannelID: userVs.ChannelID,
		textChannelID:  m.ChannelID,
	})

	worker.StartWorker(guildID)
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
	bi, ok := bindings.Load(guildID)
	if !ok {
		log.Print("not working on this guild ", guildID)
		// Not working on this guild
		if _, err := s.ChannelMessageSend(m.ChannelID, "現在このサーバーでは読み上げていません。"); err != nil {
			log.Print("error send message to channel ", m.ChannelID, " on guild ", guildID, ": ", err)
		}
		return
	}

	b := bi.(channelBinding)

	// Text channel on which the commend was post is the working text channel of bot?
	if m.ChannelID != b.textChannelID {
		log.Printf("member %s is not joining voice channel bot is reading %s", userID, b.voiceChannelID)
		thisCh, err := s.State.GuildChannel(guildID, m.ChannelID)
		if err != nil {
			log.Printf("error find guild %s channel %s", guildID, m.ChannelID)
			return
		}
		wrkCh, err := s.State.GuildChannel(guildID, b.textChannelID)
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
	if userVs.ChannelID != b.voiceChannelID {
		log.Printf("member %s is not joining voice channel bot is reading %s", botID, b.voiceChannelID)
		wrkCh, err := s.State.GuildChannel(guildID, b.textChannelID)
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
	bindings.Delete(m.GuildID)
	worker.StopWorker(m.GuildID)
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
	bi, ok := bindings.Load(m.GuildID)
	if !ok {
		log.Printf("not working on this guild %s. message is ignored", m.GuildID)
		return
	}
	b := bi.(channelBinding)
	if m.ChannelID != b.textChannelID {
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

	// TODO: trim ogg files by time
	text := replaceMention(s, m)
	text = Sanitize(text, lang)
	if textR := []rune(text); len(textR) > maxTTSLength {
		text = string(textR[:maxTTSLength]) + " 以下略" // following is omitted
	}

	t := worker.TTSTask{
		GuildID:    m.GuildID,
		Text:       text,
		Lang:       lang,
		VoiceToken: m.Author.ID, // todo
	}
	worker.AddTask(m.GuildID, t)
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
