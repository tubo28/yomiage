package main

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

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
		return
	}

	if conn, ok := dg.VoiceConnections[m.GuildID]; ok {
		// todo: force move here?
		log.Printf("bot %s is already joining voice channel %s guild %s", botID, conn.ChannelID, conn.GuildID)
		if conn.ChannelID == userVs.ChannelID {
			s.ChannelMessageSend(m.ChannelID, "already here")
		} else {
			s.ChannelMessageSend(m.ChannelID, "bot is already working on other channel")
		}
		return
	}

	if _, err := s.ChannelVoiceJoin(guildID, userVs.ChannelID, false, true); err != nil {
		log.Printf("failed to join channel %s on guild %s", userVs.ChannelID, userVs.GuildID)
		return
	}

	// todo: install errcheck
	if _, err := s.ChannelMessageSend(m.ChannelID, "I read text here"); err != nil {
		log.Printf("failed to send message to channel %s", m.ChannelID)
	}
}

func byeHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	go func() {
		for len(taskQueue) > 0 {
			t := <-taskQueue
			log.Panicf("task %s is discarded. remain task %d", t.text, len(taskQueue))
		}
	}()
	conn, ok := dg.VoiceConnections[m.GuildID]
	if !ok {
		log.Print("not joining your voice channel")
		s.ChannelMessageSend(m.ChannelID, "not joining your voice channel")
		return
	}
	if err := conn.Disconnect(); err != nil {
		log.Printf("error disconnect from voice channel %s", conn.ChannelID)
		return
	}
	s.ChannelMessageSend(m.ChannelID, "Bye")
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
