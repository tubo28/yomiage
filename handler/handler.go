package handler

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/tubo28/yomiage/db"
	"github.com/tubo28/yomiage/discord"
	"github.com/tubo28/yomiage/worker"
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
	discord.AddHandler(guildCreate)
	discord.AddHandler(messageCreate)
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if strings.HasPrefix(m.Content, "!hi") {
		if m.Author.ID == s.State.User.ID {
			return
		}
		hiHandler(s, m)
	} else if strings.HasPrefix(m.Content, "!bye") {
		if m.Author.ID == s.State.User.ID {
			return
		}
		byeHandler(s, m)
	} else if strings.HasPrefix(m.Content, "!lang") {
		if m.Author.ID == s.State.User.ID {
			return
		}
		langHandler(s, m)
	} else if strings.HasPrefix(m.Content, "!rand") {
		if m.Author.ID == s.State.User.ID {
			return
		}
		randHandler(s, m)
	} else if strings.HasPrefix(m.Content, "!help") {
		if m.Author.ID == s.State.User.ID {
			return
		}
		helpHandler(s, m)
	} else if !strings.HasPrefix(m.Content, "!") {
		nonCommandHandler(s, m)
	}
}

func langHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	arg := strings.Fields(m.Content)
	arg = arg[1:]
	if len(arg) == 0 {
		// get language
		lang, err := db.GetUserLanguage(m.Author.ID)
		if err != nil {
			log.Print("error get user ", m.Author.ID, "'s language: ", err.Error())
			return
		}
		msg := fmt.Sprintf("User %s's language is %s", m.Author.Username, lang)
		if _, err := s.ChannelMessageSend(m.ChannelID, msg); err != nil {
		}
	} else if arg[0] == "set" {
		// set language
		arg = arg[1:]
		lang := arg[0]
		if err := db.UpsertUserLanguage(m.Author.ID, lang); err != nil {
			log.Print("error update user ", m.Author.ID, "'s language to ", lang, ": ", err.Error())
			return
		}
		msg := fmt.Sprintf("User %s's language is updated to %s", m.Author.Username, lang)
		if _, err := s.ChannelMessageSend(m.ChannelID, msg); err != nil {
		}
	}
}

func randHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// generate random token and set for user
	u, _ := uuid.NewUUID()
	vt := u.String()
	err := db.UpsertUserVoiceToken(m.Author.ID, vt)
	if err != nil {
		log.Print("error update user voice token ", m.Author.ID, "'s token", err.Error())
		return
	}
	if _, err := s.ChannelMessageSend(m.ChannelID, "Randomized your voice."); err != nil {
	}
	lang, err := db.GetUserLanguage(m.Author.ID)
	if err != nil {
		// todo: default
	}
	worker.AddTask(m.GuildID, worker.TTSTask{
		GuildID:    m.GuildID,
		Text:       "this is sample, hello, hello!",
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
	msg := "Read this: https://github.com/tubo28/yomiage/blob/main/README.md"
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
		Text:       m.Content,
		Lang:       lang,
		VoiceToken: m.Author.ID,
	}
	worker.AddTask(m.GuildID, t)
}

func guildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
	if event.Guild.Unavailable {
		return
	}

	for _, channel := range event.Guild.Channels {
		if channel.ID == event.Guild.ID {
			_, _ = s.ChannelMessageSend(channel.ID, "Yomiage is ready! Type `!hi` to start reading text channel. Type `!help` to show help.")
			return
		}
	}
}
