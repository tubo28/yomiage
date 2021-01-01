package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	tts "cloud.google.com/go/texttospeech/apiv1"
	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/ogg"
	tts_pb "google.golang.org/genproto/googleapis/cloud/texttospeech/v1"
)

const (
	discordToken string = "Nzk0NDkzODg2ODI2ODA3MzI2.X-7oFw.Z2HjIydKzkCJ9B6PpsC-xxj-1RE"
)

var (
	client *tts.Client
	dg     *discordgo.Session
)

func main() {
	if discordToken == "" {
		log.Fatal("no discord token is given")
	}

	var err error
	client, err = tts.NewClient(context.TODO())
	if err != nil {
		log.Fatal("failed to create tts client: ", err.Error())
	}

	// Create a new Discord session using the provided bot token.
	dg, err = discordgo.New("Bot " + discordToken)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		return
	}

	// Register messageCreate as a callback for the messageCreate events.
	dg.AddHandler(messageCreate)

	// Register guildCreate as a callback for the guildCreate events.
	dg.AddHandler(guildCreate)

	// We need information about guilds (which includes their channels),
	// messages and voice states.
	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates)

	// Open the websocket and begin listening.
	if err := dg.Open(); err != nil {
		fmt.Println("Error opening Discord session: ", err)
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Airhorn is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}
	botID := m.Author.ID
	guildID := m.GuildID

	if strings.HasPrefix(m.Content, "!hi") {
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
	} else if strings.HasPrefix(m.Content, "!bye") {
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
	} else {
		if strings.HasPrefix(m.Content, "!") {
			return
		}
		conn, ok := dg.VoiceConnections[m.GuildID]
		if !ok {
			return
		}
		oggBuf, err := ttsOGGGoogle(m.Content, "ja-JP")
		if err != nil {
			log.Printf("failed to create tts audio: %s", err.Error())
			return
		}
		if err := playOGG(conn, oggBuf); err != nil {
			log.Printf("failed to play ogg: %s", err.Error())
		}
	}
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

// lang: https://cloud.google.com/text-to-speech/docs/voices
// todo: randomize voice
func ttsOGGGoogle(s, lang string) ([][]byte, error) {
	req := tts_pb.SynthesizeSpeechRequest{
		// Set the text input to be synthesized.
		Input: &tts_pb.SynthesisInput{
			InputSource: &tts_pb.SynthesisInput_Text{Text: s},
		},
		// Build the voice request, select the language code ("en-US") and the SSML
		// voice gender ("neutral").
		Voice: &tts_pb.VoiceSelectionParams{
			LanguageCode: lang,
			SsmlGender:   tts_pb.SsmlVoiceGender_NEUTRAL,
		},
		// Select the type of audio file you want returned.
		AudioConfig: &tts_pb.AudioConfig{
			AudioEncoding: tts_pb.AudioEncoding_OGG_OPUS,
		},
	}

	resp, err := client.SynthesizeSpeech(context.TODO(), &req)
	if err != nil {
		return nil, fmt.Errorf("failed to create ogg, received error from google api: %w", err)
	}

	log.Printf("ogg data created %d bytes", len(resp.AudioContent))
	return makeOGGBuffer(resp.AudioContent)
}

func makeOGGBuffer(in []byte) (output [][]byte, err error) {
	// Setup our ogg and packet decoders
	od := ogg.NewDecoder(bytes.NewReader(in))
	pd := ogg.NewPacketDecoder(od)

	// Run through the packet decoder appending the bytes to our output [][]byte
	for {
		packet, _, err := pd.Decode()
		if err != nil {
			if err != io.EOF {
				return nil, fmt.Errorf("error decode on PacketDecoder: %w", err)
			}
			return output, nil
		}
		output = append(output, packet)
	}
}

// playOGG plays the current buffer to the provided channel.
func playOGG(conn *discordgo.VoiceConnection, oggBuf [][]byte) (err error) {
	// Sleep for a specified amount of time before playing the sound
	time.Sleep(250 * time.Millisecond)

	// Start speaking.
	conn.Speaking(true)

	// Send the buffer data.
	for _, buff := range oggBuf {
		conn.OpusSend <- buff
	}

	// Stop speaking
	conn.Speaking(false)

	// Sleep for a specificed amount of time before ending.
	time.Sleep(250 * time.Millisecond)

	return nil
}

// This function will be called (due to AddHandler above) every time a new
// guild is joined.
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
