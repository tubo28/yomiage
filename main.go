package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	tts "cloud.google.com/go/texttospeech/apiv1"
	"github.com/bwmarrin/discordgo"
)

var (
	discordToken   string = os.Getenv("DISCORD_TOKEN")
	defaultTTSLang string = os.Getenv("TTS_LANG")
	dg             *discordgo.Session
)

func main() {
	if discordToken == "" {
		log.Fatal("no discord token is given")
	}
	if defaultTTSLang == "" {
		defaultTTSLang = "en-US"
	}

	var err error
	dg, err = discordgo.New("Bot " + discordToken)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		return
	}

	ttsClient, err = tts.NewClient(context.TODO())
	if err != nil {
		log.Fatal("failed to create tts client: ", err.Error())
	}

	dg.AddHandler(messageCreate)
	dg.AddHandler(guildCreate)

	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates)

	if err := dg.Open(); err != nil {
		fmt.Println("Error opening Discord session: ", err)
	}

	go cleanerWorker()

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("yomiage is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	stat := 0
	if err := db.Close(); err != nil {
		log.Print(err.Error())
		stat = 1
	}
	if err := dg.Close(); err != nil {
		log.Print(err.Error())
		stat = 1
	}
	os.Exit(stat)
}
