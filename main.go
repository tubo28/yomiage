package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var (
	discordToken string = os.Getenv("DISCORD_TOKEN")
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

	go cleanerWorker()

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Airhorn is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}
