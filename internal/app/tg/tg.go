package tg

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"
	"publika-auction/internal/app/bids"
	"publika-auction/internal/app/hub"
)

type TGBot struct {
	bot    *tgbotapi.BotAPI
	hb     *hub.Hub
	bdsOut chan bids.Msg
}

type Config struct {
	Token    string
	Endpoint string
}

func New(c Config, hb *hub.Hub, bdsOut chan bids.Msg) (b *TGBot, err error) {
	// tg bot
	bot, err := tgbotapi.NewBotAPIWithAPIEndpoint(c.Token, c.Endpoint)
	if err != nil {
		log.Err(err).Msg("bot api create error error")
		return nil, err
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)
	return &TGBot{bot: bot, hb: hb, bdsOut: bdsOut}, nil
}

func (bt *TGBot) Start(ctx context.Context) {

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bt.bot.GetUpdatesChan(u)
	/*
		tempCmnds := []tgbotapi.BotCommand{
			{
				Command:     "/help",
				Description: "Помощь",
			},
		}
		cmnds := tgbotapi.NewSetMyCommands(tempCmnds...)
		bt.bot.Send(cmnds)
	*/

	writer := make(chan tgbotapi.Chattable)

	for {
		select {
		case update := <-updates:
			var chatID int64
			username := ""
			if update.Message != nil { // If we got a message
				chatID = update.Message.Chat.ID
				username = update.Message.From.UserName
				log.Info().Int64("chatid", update.Message.Chat.ID).Str("username", update.Message.From.UserName).Str("message", update.Message.Text).Msg("tg income message t1")
			} else if update.CallbackQuery != nil {
				log.Info().Interface("data", update.CallbackQuery.Data).Msg("tg income message t2")
				chatID = update.CallbackQuery.From.ID
				username = update.CallbackQuery.From.UserName
			}
			if chatID != 0 {
				chat, _ := bt.hb.GetChat(chatID, username, writer)
				go chat.SendTo(update)
			} else {
				log.Info().Interface("update", update).Msg("unknown type")
			}
		case toSend := <-writer:
			//

			m, _ := bt.bot.Send(toSend)
			log.Info().Interface("toSend", toSend).Interface("message", m).Msg("tg send message")
		case toSend := <-bt.bdsOut:
			//
			msg := tgbotapi.NewMessage(toSend.ChatId, toSend.Message)
			m, _ := bt.bot.Send(msg)
			log.Info().Interface("toSend", toSend).Interface("message", m).Msg("tg send message")
		case <-ctx.Done():
			log.Info().Msg("bot stopped (context is done")
			return
		}
	}
}
