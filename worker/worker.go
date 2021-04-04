package worker

import (
	"context"
	"log"
	"sync"
)

const taskQueueCapacity = 32

// Task executed async
type Task struct {
	ID string
	Do func() error
}

func NewTask(ID string, Do func() error) *Task {
	return &Task{ID: ID, Do: Do}
}

type Consumer struct {
	ID    string
	queue chan Task
}

func NewConsumer(ID string) *Consumer {
	return &Consumer{
		ID:    ID,
		queue: make(chan Task, taskQueueCapacity),
	}
}

func (c *Consumer) StartAsync(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
	L:
		for {
			select {
			case <-ctx.Done():
				break L
			default:
			}

			task, ok := <-c.queue
			if !ok {
				log.Printf("task queue of consumer %s is closed", c.ID)
				break L
			}
			log.Print("<-- task received: ", task.ID)
			if err := task.Do(); err != nil {
				log.Print("error occurs: ", err)
			}
			log.Print("task finished: ", task.ID)
		}

		log.Printf("stop consumer %s. %d tasks remains", c.ID, len(c.queue))
		close(c.queue)
		for task := range c.queue {
			log.Printf("task %s is discarded. consumer will be killed", task.ID)
		}

		log.Printf("consume %s is killed", c.ID)
		wg.Done()
	}()
}

func (c *Consumer) Add(t Task) {
	select {
	case c.queue <- t:
		log.Printf("--> added task %+v to consumer %s", t, c.ID)
	default:
		log.Printf("--x discarded task %+v of consumer %s. task queue is full", t, c.ID)
	}
}
