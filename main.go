package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	discordToken string = "Nzk0NDkzODg2ODI2ODA3MzI2.X-7oFw.Z2HjIydKzkCJ9B6PpsC-xxj-1RE"
	dg           *discordgo.Session
)

func init() {
	if discordToken == "" {
		log.Fatal("no discord token is given")
	}

	var err error
	dg, err = discordgo.New("Bot " + discordToken)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		return
	}
}

func main() {
	dg.AddHandler(messageCreate)
	dg.AddHandler(guildCreate)

	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates)

	if err := dg.Open(); err != nil {
		fmt.Println("Error opening Discord session: ", err)
	}

	go ttsTaskConsumer()

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Airhorn is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	if strings.HasPrefix(m.Content, "!hi") {
		hiHandler(s, m)
	} else if strings.HasPrefix(m.Content, "!bye") {
		byeHandler(s, m)
	} else if strings.HasPrefix(m.Content, "!lang") {
		// set users lang
	} else if strings.HasPrefix(m.Content, "!rand") {
		// randomize voice
	} else if !strings.HasPrefix(m.Content, "!") {
		nonCommandHandler(s, m)
	}
}

type task struct {
	guildID string
	text    string
}

const queueCap = 32

var taskQueue = make(chan task, queueCap)

func nonCommandHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	t := task{
		guildID: m.GuildID,
		text:    m.Content,
	}
	select {
	case taskQueue <- t:
	default:
		fmt.Println("discarding message due to chanel is full:", m.Content)
	}
}

func ttsTaskConsumer() {
	log.Print("start ttsTaskConsumer")
	for {
		time.Sleep(200 * time.Millisecond)
		if t, ok := <-taskQueue; ok {
			consumeOne(t)
		}
		println("ttsTaskConsumer: remain task", len(taskQueue))
	}
}

func consumeOne(t task) {
	conn, ok := dg.VoiceConnections[t.guildID]
	if !ok {
		log.Printf("voice channel on guild %s is deleted", t.guildID)
		return
	}

	oggBuf, err := ttsOGGGoogle(t.text, "ja-JP")
	if err != nil {
		log.Printf("failed to create tts audio: %s", err.Error())
		return
	}
	playOGG(conn, oggBuf)
}

func playOGG(conn *discordgo.VoiceConnection, oggBuf [][]byte) {
	conn.Speaking(true)
	for _, buff := range oggBuf {
		conn.OpusSend <- buff
	}
	conn.Speaking(false)
}
