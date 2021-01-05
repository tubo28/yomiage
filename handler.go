package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

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
		// set users lang
	} else if strings.HasPrefix(m.Content, "!rand") {
		if m.Author.ID == s.State.User.ID {
			return
		}
		// randomize voice
	} else if !strings.HasPrefix(m.Content, "!") {
		nonCommandHandler(s, m)
	}
}

func hiHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	botID := m.Author.ID
	guildID := m.GuildID

	userVs, err := voiceState(s, botID, guildID)
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

	startWorker(guildID)
	time.Sleep(200 * time.Millisecond) // waiting for bot to join voice channel
	if msg := joinVC(s, m.GuildID, userVs.ChannelID); msg != "" {
		if _, err := s.ChannelMessageSend(m.ChannelID, msg); err != nil {
			log.Print("error send message to channel ", m.ChannelID, " on guild ", guildID, ": ", err)
		}
	}
}

func byeHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	stopWorker(m.GuildID)
	if msg := leaveVC(m.GuildID); msg != "" {
		if _, err := s.ChannelMessageSend(m.ChannelID, msg); err != nil {
			log.Print("error send message to channel ", m.ChannelID, " on guild ", m.GuildID, ": ", err)
		}
	}
}

func nonCommandHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	t := task{
		guildID:    m.GuildID,
		text:       m.Content,
		lang:       defaultTTSLang, // todo: use saved user's param
		voiceToken: m.Author.ID,
	}
	addTask(m.GuildID, t)
}

func joinVC(s *discordgo.Session, guildID, vcID string) (msg string) {
	if conn, ok := dg.VoiceConnections[guildID]; ok {
		// todo: force move here?
		if conn.ChannelID == vcID {
			log.Printf("bot is already joining target voice channel %s guild %s", conn.ChannelID, conn.GuildID)
			return "Already here"
		}
		log.Printf("bot is already joining other voice channel %s guild %s", conn.ChannelID, conn.GuildID)
		return "Already working on other channel"
	}

	if _, err := s.ChannelVoiceJoin(guildID, vcID, false, true); err != nil {
		log.Printf("failed to join channel %s on guild %s", vcID, guildID)
		return
	}

	return "I read text here"
}

func leaveVC(guildID string) (msg string) {
	conn, ok := dg.VoiceConnections[guildID]
	if !ok {
		return "Not joining your voice channel"
	}

	if err := conn.Disconnect(); err != nil {
		log.Printf("error disconnect from voice channel %s: %s", conn.ChannelID, err.Error())
		return
	}

	return "Bye"
}

func voiceState(s *discordgo.Session, userID, guildID string) (*discordgo.VoiceState, error) {
	g, err := s.State.Guild(guildID)
	if err != nil {
		return nil, fmt.Errorf("error find guild that the message post to: %s", err)
	}
	for _, vs := range g.VoiceStates {
		if vs.UserID == userID {
			return vs, nil
		}
	}
	return nil, nil
}

func guildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
	if event.Guild.Unavailable {
		return
	}

	for _, channel := range event.Guild.Channels {
		if channel.ID == event.Guild.ID {
			_, _ = s.ChannelMessageSend(channel.ID, "Airhorn is ready! Type !airhorn while in a voice channel to play a sound.")
			return
		}
	}
}
