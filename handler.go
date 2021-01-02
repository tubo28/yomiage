package main

import (
	"fmt"
	"log"
	"time"

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
		s.ChannelMessageSend(m.ChannelID, "cannot summon me without joining any voice channel")
		return
	}

	startWorker(guildID)
	time.Sleep(200 * time.Millisecond) // waiting for bot to join voice channel
	joinVC(s, m.GuildID, userVs.ChannelID, m.ChannelID)
}

func byeHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	stopWorker(m.GuildID)
	leaveVC(s, m.GuildID, m.ChannelID)
}

func nonCommandHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	t := task{
		guildID: m.GuildID,
		text:    m.Content,
	}
	addTask(m.GuildID, t)
}

func joinVC(s *discordgo.Session, guildID, vcID, tcID string) {
	if conn, ok := dg.VoiceConnections[guildID]; ok {
		// todo: force move here?
		if conn.ChannelID == vcID {
			log.Printf("bot is already joining target voice channel %s guild %s", conn.ChannelID, conn.GuildID)
			s.ChannelMessageSend(tcID, "already here")
		} else {
			log.Printf("bot is already joining other voice channel %s guild %s", conn.ChannelID, conn.GuildID)
			s.ChannelMessageSend(tcID, "bot is already working on other channel")
		}
		return
	}

	if _, err := s.ChannelVoiceJoin(guildID, vcID, false, true); err != nil {
		log.Printf("failed to join channel %s on guild %s", vcID, guildID)
		return
	}

	// todo: install errcheck
	if _, err := s.ChannelMessageSend(tcID, "I read text here"); err != nil {
		log.Printf("failed to send message to channel %s", tcID)
	}

}

func leaveVC(s *discordgo.Session, guildID, tcID string) {
	conn, ok := dg.VoiceConnections[guildID]
	if !ok {
		log.Print("not joining your voice channel")
		s.ChannelMessageSend(tcID, "not joining your voice channel")
		return
	}
	if err := conn.Disconnect(); err != nil {
		log.Printf("error disconnect from voice channel %s: %s", conn.ChannelID, err.Error())
		return
	}
	s.ChannelMessageSend(tcID, "Bye")
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
