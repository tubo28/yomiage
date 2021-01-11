package worker

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/tubo28/yomiage/discord"
)

// TTSTask is tts task executed async
type TTSTask struct {
	GuildID    string
	Text       string
	Lang       string
	VoiceToken string
}

type worker struct {
	guildID string
	done    chan bool
	queue   chan TTSTask
}

// one worker per guild
var ttsWorkers sync.Map

// StartWorker starts task consumer
func StartWorker(guildID string) {
	_, ok := ttsWorkers.Load(guildID)
	if ok {
		log.Print("worker is already running for guild ", guildID)
		return
	}
	w := &worker{
		guildID: guildID,
		done:    make(chan bool),
		queue:   make(chan TTSTask, 32),
	}
	ttsWorkers.Store(guildID, w)

	go func() {
	L:
		for {
			select {
			case <-w.done:
				log.Printf("stop worker. %d tasks remains", len(w.queue))
				break L
			case t, ok := <-w.queue:
				if !ok {
					log.Print("task queue is empty")
					break L
				}
				log.Print("process task: ", t.Text)
				if err := do(t); err != nil {
					log.Print("error processing task: ", err.Error())
				}
			}
		}
	}()
}

// AddTask adds task to queue
func AddTask(guildID string, t TTSTask) {
	iw, ok := ttsWorkers.Load(guildID)
	if !ok {
		log.Printf("no worker for guild %s, task %s is discarded", guildID, t.Text)
		return
	}
	w := iw.(*worker)
	select {
	case w.queue <- t:
		log.Print("task is added: ", t.Text)
	default:
		fmt.Println("discarding message due to chanel is full:", t.Text)
	}
}

// StopWorker stops worker for the guild specified by guildID
func StopWorker(guildID string) {
	iw, ok := ttsWorkers.Load(guildID)
	if !ok {
		log.Print("no worker for guild ", guildID)
		return
	}
	w := iw.(*worker)
	w.done <- true
	close(w.queue)
	for t := range w.queue {
		log.Print("task is discorded because worker is stopped: ", t.Text)
	}
	ttsWorkers.Delete(guildID)
}

func do(t TTSTask) error {
	time.Sleep(200 * time.Millisecond)
	return discord.Play(t.Text, t.Lang, t.VoiceToken, t.GuildID)
}

// CleanerWorker starts worker which clean up workers which is alone on voice channels
func CleanerWorker() {
	for {
		time.Sleep(10 * time.Second)
		gIDs := []string{}
		ttsWorkers.Range(func(k interface{}, v interface{}) bool {
			gID := k.(string)
			gIDs = append(gIDs, gID)
			return true
		})
		for _, gID := range gIDs {
			time.Sleep(1 * time.Second)
			if discord.Alone(gID) {
				log.Printf("bot is alone in voice channel on guild %s, leave", gID)
				StopWorker(gID)
				_ = discord.LeaveVC(gID)
			}
		}
	}
}
