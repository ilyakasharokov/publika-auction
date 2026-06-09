package tgqueue

import (
	"context"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"

	"publika-auction/internal/metrics"
)

type NotifierIface interface {
	Send(tgID int64, text string)
}

type Msg struct {
	ChatID int64
	Text   string
	Markup interface{}
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
			q.sendWithRetry(ctx, m)
		case <-ctx.Done():
			return
		}
	}
}

func (q *Queue) sendWithRetry(ctx context.Context, m Msg) {
	var msg tgbotapi.MessageConfig
	if m.Markup != nil {
		msg = tgbotapi.NewMessage(m.ChatID, m.Text)
		msg.ReplyMarkup = m.Markup
	} else {
		msg = tgbotapi.NewMessage(m.ChatID, m.Text)
	}

	backoff := 1 * time.Second
	for attempt := 0; attempt < 4; attempt++ {
		_, err := q.bot.Send(msg)
		if err == nil {
			metrics.TGMessagesSentTotal.WithLabelValues("queue").Inc()
			return
		}
		// Telegram rate-limit: retry after suggested delay.
		if strings.Contains(err.Error(), "429") {
			retryAfter := backoff
			if te, ok := err.(*tgbotapi.Error); ok && te.RetryAfter > 0 {
				retryAfter = time.Duration(te.RetryAfter) * time.Second
			}
			log.Warn().Int64("chat_id", m.ChatID).Dur("retry_after", retryAfter).Msg("tg rate limited, retrying")
			select {
			case <-time.After(retryAfter):
			case <-ctx.Done():
				return
			}
			backoff *= 2
			continue
		}
		// Non-retryable error.
		log.Err(err).Int64("chat_id", m.ChatID).Msg("tg send error")
		return
	}
	log.Warn().Int64("chat_id", m.ChatID).Msg("tg send failed after retries, dropping")
	metrics.TGMessagesDroppedTotal.Inc()
}
