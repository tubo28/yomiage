package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/mattn/go-sqlite3"
	"github.com/tubo28/yomiage/db"
	"github.com/tubo28/yomiage/discord"
	"github.com/tubo28/yomiage/handler"
	"github.com/tubo28/yomiage/tts"
)

func main() {
	tts.Init()
	defer tts.Close()

	db.Init()
	defer db.Close()

	discord.Init()
	defer discord.Close()

	handler.Init()

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("yomiage is now running. press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	fmt.Println("Exit.")

	os.Exit(0)
}
