package tg

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"

	"publika-auction/internal/hub"
	"publika-auction/internal/tgqueue"
)

type Config struct {
	Token    string
	Endpoint string
}

type Bot struct {
	bot   *tgbotapi.BotAPI
	hub   *hub.Hub
	queue *tgqueue.Queue
}

func New(cfg Config, h *hub.Hub) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPIWithAPIEndpoint(cfg.Token, cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	bot.Debug = false
	log.Info().Str("username", bot.Self.UserName).Msg("telegram bot authorized")

	q := tgqueue.New(bot, 1000, 3)
	return &Bot{bot: bot, hub: h, queue: q}, nil
}

func (b *Bot) Queue() *tgqueue.Queue {
	return b.queue
}

func (b *Bot) Username() string {
	return b.bot.Self.UserName
}

func (b *Bot) Start(ctx context.Context) {
	b.queue.Start(ctx)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.bot.GetUpdatesChan(u)

	writer := make(chan tgbotapi.Chattable, 256)
	go b.sender(ctx, writer)

	for {
		select {
		case update := <-updates:
			var chatID int64
			username := ""
			if update.Message != nil {
				chatID = update.Message.Chat.ID
				username = update.Message.From.UserName
			} else if update.CallbackQuery != nil {
				chatID = update.CallbackQuery.From.ID
				username = update.CallbackQuery.From.UserName
			}
			if chatID != 0 {
				chat, _ := b.hub.GetChat(chatID, username, writer)
				go chat.SendTo(update)
			}
		case msg := <-b.hub.Out:
			if _, err := b.bot.Send(msg); err != nil {
				log.Err(err).Msg("hub.Out send error")
			}
		case <-ctx.Done():
			log.Info().Msg("tg bot stopped")
			return
		}
	}
}

func (b *Bot) sender(ctx context.Context, writer chan tgbotapi.Chattable) {
	for {
		select {
		case msg := <-writer:
			if _, err := b.bot.Send(msg); err != nil {
				log.Err(err).Msg("writer send error")
			}
		case <-ctx.Done():
			return
		}
	}
}
