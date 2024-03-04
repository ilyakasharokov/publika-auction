package worker

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"time"
)

type Worker struct {
	pusher Pusher
	jobs   chan DoContestTask
}

type DoContestTask struct {
	Begin time.Time
	End   time.Time
	UUID  uuid.UUID
}

type Pusher interface {
	Send(e []map[string]interface{}) error
}

func New() *Worker {
	return &Worker{
		// pusher: pusher,
		jobs: make(chan DoContestTask, 10),
	}
}

func (w *Worker) Start(ctx context.Context, tickerTime time.Duration) {
	ticker := time.NewTicker(tickerTime)
	for {
		select {
		case <-ticker.C:

		case <-w.jobs:

		case <-ctx.Done():
			log.Info().Msg("worker | start context is done")
			return
		}
	}
}

func (w *Worker) Add(task DoContestTask) error {
	select {
	case w.jobs <- task:
	default:
		err := errors.New("queue is full")
		log.Err(err).Msg("worker add error")
		return err
	}
	return nil
}
