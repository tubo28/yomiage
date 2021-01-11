package discord

import (
	"fmt"
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/tubo28/yomiage/tts"
)

var (
	discordToken string = os.Getenv("DISCORD_TOKEN")
	dg           *discordgo.Session
)

func init() {
	if discordToken == "" {
		log.Fatal("no discord token is given")
	}
}

func Init() {
	var err error
	dg, err = discordgo.New("Bot " + discordToken)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		return
	}

	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates)

	if err := dg.Open(); err != nil {
		fmt.Println("Error opening Discord session: ", err)
	}
}

// Close closes discordgo client
func Close() {
	if err := dg.Close(); err != nil {
		log.Print(err.Error())
	}
}

func AddHandler(f interface{}) {
	dg.AddHandler(f)
}

func JoinVC(s *discordgo.Session, guildID, vcID string) (msg string) {
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

func LeaveVC(guildID string) (msg string) {
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

// VoiceState return discordgo.VoiceState of the guild on the user
func VoiceState(s *discordgo.Session, userID, guildID string) (*discordgo.VoiceState, error) {
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

func voiceConnection(guildID string) (*discordgo.VoiceConnection, bool) {
	conn, ok := dg.VoiceConnections[guildID]
	return conn, ok
}

// Play plays tts sound on VC
func Play(text, lang, voiceToken, guildID string) error {
	conn, ok := voiceConnection(guildID)
	if !ok {
		return fmt.Errorf("voice channel on guild %s is deleted. maybe zombie worker", guildID)
	}

	oggBuf, err := tts.OGGGoogle(text, lang, voiceToken)
	if err != nil {
		log.Printf("failed to create tts audio: %s", err.Error())
		return nil
	}
	if err := conn.Speaking(true); err != nil {
	}
	for _, buff := range oggBuf {
		conn.OpusSend <- buff
	}
	if err := conn.Speaking(false); err != nil {
	}
	return nil
}

// Alone returns whether the bot is alone on VC
func Alone(guildID string) bool {
	conn, ok := voiceConnection(guildID)
	if !ok {
		return true
	}
	vcID := ""
	g, err := dg.State.Guild(conn.GuildID)
	if err != nil {
		log.Print("error failed to get state of guild by GuildID : ", err.Error())
		return true
	}
	for _, vs := range g.VoiceStates {
		if vs.UserID == conn.UserID {
			vcID = vs.ChannelID
			break
		}
	}
	if vcID == "" {
		log.Print("error voice chat joining bot not found")
		return true
	}
	count := 0
	for _, vs := range g.VoiceStates {
		if vs.ChannelID == vcID {
			count++
		}
	}
	return count <= 1
}
