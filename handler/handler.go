package handler

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
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
		var head string
		if head == "lang" {
			if m.Author.ID == s.State.User.ID {
				return
			}
			langHandler(s, m, args[1:])
			return
		}
		if head == "rand" {
			if m.Author.ID == s.State.User.ID {
				return
			}
			randHandler(s, m, args[1:])
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
		msg := fmt.Sprintf("%s の読み上げ言語は %s です。", m.Author.Username, lang)
		if _, err := s.ChannelMessageSend(m.ChannelID, msg); err != nil {
		}
	} else if args[0] == "set" {
		// set language
		args = args[1:]
		lang := args[0]
		if err := db.UpsertUserLanguage(m.Author.ID, lang); err != nil {
			log.Print("error update user ", m.Author.ID, "'s language to ", lang, ": ", err.Error())
			return
		}
		// User %s's language is updated to %s
		msg := fmt.Sprintf("%s の読み上げ言語を %s に変更しました。", m.Author.Username, lang)
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
	if _, err := s.ChannelMessageSend(m.ChannelID, "声を変更しました。"); err != nil {
	}
	var lang string
	lang, err = db.GetUserLanguage(m.Author.ID)
	if err != nil {
		lang = defaultTTSLang
	}
	worker.AddTask(m.GuildID, worker.TTSTask{
		GuildID:    m.GuildID,
		Text:       "サンプル、イカよろしく～", // This is test
		Lang:       lang,
		VoiceToken: vt,
	})
}

func hiHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	botID := m.Author.ID
	guildID := m.GuildID

	userVs, err := discord.VoiceState(s, botID, guildID)
	if err != nil {
		log.Printf("failed to get VoiceState of guild %s: %s", guildID, err.Error())
		return
	}
	if userVs == nil {
		log.Printf("member %s is not joining any voice channel", botID)
		if _, err := s.ChannelMessageSend(m.ChannelID, "Cannot summon me without joining any voice channel"); err != nil {
			log.Print("error send message to channel ", m.ChannelID, " on guild ", guildID, ": ", err)
		}
		return
	}

	worker.StartWorker(guildID)
	time.Sleep(200 * time.Millisecond) // waiting for bot to join voice channel
	if msg := discord.JoinVC(s, m.GuildID, userVs.ChannelID); msg != "" {
		if _, err := s.ChannelMessageSend(m.ChannelID, msg); err != nil {
			log.Print("error send message to channel ", m.ChannelID, " on guild ", guildID, ": ", err)
		}
	}
}

func byeHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
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

func nonCommandHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	lang, err := db.GetUserLanguage(m.Author.ID)
	if err != nil {
		log.Print("error get user "+m.Author.ID+"'s langage: ", err)
	}
	if lang == "" {
		lang = defaultTTSLang
	}

	t := worker.TTSTask{
		GuildID:    m.GuildID,
		Text:       Sanitize(m.Content, lang, m.Mentions),
		Lang:       lang,
		VoiceToken: m.Author.ID,
	}
	worker.AddTask(m.GuildID, t)
}

var (
	urlReg    = xurls.Relaxed
	ignoreReg = regexp.MustCompile("^[(（)].*[）)]$")
	kusaReg   = regexp.MustCompile("[wWｗＷ]+$")
)

// Sanitize modifies m.Content easier to read for bot in following steps:
// 1. trim spaces
// 2. replace mention string to user name
// 3. replace continuous 'w's to kusa
// 4. replace URL to "URL"
// 5. replace continuous whitespaces to single one
func Sanitize(content, lang string, mentions []*discordgo.User) string {
	s := strings.TrimSpace(content)

	// 1
	if ignoreReg.MatchString(s) {
		return ""
	}

	// 2
	for _, user := range mentions {
		mentionPatStr := fmt.Sprintf(`<@!?%s>`, user.ID)
		mentionPat, err := regexp.Compile(mentionPatStr)
		if err != nil {
			log.Printf("failed to compile mention pattern %s to regexp: %s", mentionPat, err)
			continue
		}
		s = mentionPat.ReplaceAllString(s, user.Username)
	}

	// 3
	if (strings.HasPrefix(lang, "ja-") || lang == "ja") && kusaReg.MatchString(s) {
		s = kusaReg.ReplaceAllString(s, " くさ")
	}

	// 4
	s = urlReg.ReplaceAllString(s, " URL ")

	// 5
	s = strings.Join(strings.Fields(s), " ")

	return s
}
