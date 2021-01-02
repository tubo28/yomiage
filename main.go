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

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Airhorn is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
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

type task struct {
	guildID string
	text    string
}

type worker struct {
	guildID string
	done    chan bool
	queue   chan task
}

// one worker per guild
var ttsWorkers = make(map[string]*worker)

func startWorker(guildID string) {
	w, ok := ttsWorkers[guildID]
	if ok {
		log.Print("worker is already running for guild ", guildID)
		return
	}
	w = &worker{
		guildID: guildID,
		done:    make(chan bool),
		queue:   make(chan task, 32),
	}
	ttsWorkers[guildID] = w

	go func() {
	L:
		for {
			select {
			case <-w.done:
				log.Print("worker is stopped")
				break L
			case t, ok := <-w.queue:
				if !ok {
					log.Print("task queue is empty")
					break L
				}
				log.Print("process task: ", t.text)
				if err := doTask(t); err != nil {
					log.Print("error processing task: ", err.Error())
				}
			}
		}
	}()
}

func addTask(guildID string, t task) {
	w, ok := ttsWorkers[guildID]
	if !ok {
		log.Printf("no worker for guild %s, task %s is discarded", guildID, t.text)
		return
	}
	select {
	case w.queue <- t:
		log.Print("task is added: ", t.text)
	default:
		fmt.Println("discarding message due to chanel is full:", t.text)
	}
}

func stopWorker(guildID string) {
	w, ok := ttsWorkers[guildID]
	if !ok {
		log.Print("no worker for guild ", guildID)
		return
	}
	w.done <- true
	close(w.queue)
	for t := range w.queue {
		log.Print("task is discorded because worker is stopped: ", t.text)
	}
	delete(ttsWorkers, guildID)
}

func doTask(t task) error {
	defer func() {
		time.Sleep(200 * time.Millisecond)
	}()
	conn, ok := dg.VoiceConnections[t.guildID]
	if !ok {
		return fmt.Errorf("voice channel on guild %s is deleted. maybe zombie worker", t.guildID)
	}

	oggBuf, err := ttsOGGGoogle(t.text, "ja-JP")
	if err != nil {
		log.Printf("failed to create tts audio: %s", err.Error())
		return nil
	}
	playOGG(conn, oggBuf)
	return nil
}

func playOGG(conn *discordgo.VoiceConnection, oggBuf [][]byte) {
	conn.Speaking(true)
	for _, buff := range oggBuf {
		conn.OpusSend <- buff
	}
	conn.Speaking(false)
}
