package main

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

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
