package tgqueue

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"

	"publika-auction/internal/metrics"
)

type NotifierIface interface {
	Send(tgID int64, text string)
}

type Msg struct {
	ChatID  int64
	Text    string
	LotNum  int
	Markup  interface{}
}

type TGSender interface {
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
}

type Queue struct {
	ch      chan Msg
	workers int
	bot     TGSender
}

func New(bot TGSender, bufSize, workers int) *Queue {
	return &Queue{
		ch:      make(chan Msg, bufSize),
		workers: workers,
		bot:     bot,
	}
}

func (q *Queue) Enqueue(m Msg) {
	select {
	case q.ch <- m:
		metrics.TGQueueDepth.Inc()
	default:
		metrics.TGMessagesDroppedTotal.Inc()
		log.Warn().Int64("chat_id", m.ChatID).Msg("tg queue full, message dropped")
	}
}

func (q *Queue) Send(tgID int64, text string) {
	q.Enqueue(Msg{ChatID: tgID, Text: text})
}

func (q *Queue) Start(ctx context.Context) {
	for i := 0; i < q.workers; i++ {
		go q.worker(ctx)
	}
}

func (q *Queue) worker(ctx context.Context) {
	for {
		select {
		case m := <-q.ch:
			metrics.TGQueueDepth.Dec()
			var msg tgbotapi.MessageConfig
			if m.Markup != nil {
				msg = tgbotapi.NewMessage(m.ChatID, m.Text)
				msg.ReplyMarkup = m.Markup
			} else {
				msg = tgbotapi.NewMessage(m.ChatID, m.Text)
			}
			if _, err := q.bot.Send(msg); err != nil {
				log.Err(err).Int64("chat_id", m.ChatID).Msg("tg send error")
			} else {
				metrics.TGMessagesSentTotal.WithLabelValues("queue").Inc()
			}
		case <-ctx.Done():
			return
		}
	}
}
