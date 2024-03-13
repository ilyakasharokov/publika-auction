package hub

import (
	"github.com/go-redis/redis/v8"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"
	"publika-auction/internal/app/bids"
	clients_repo "publika-auction/internal/app/clients-repo"
	"publika-auction/internal/app/models"
	"time"
)

type Hub struct {
	Chats map[int64]*Chat

	redis  *redis.Client
	clRepo *clients_repo.ClientsRepository
	bds    *bids.BidsStorage
	Out    chan tgbotapi.Chattable
}

func New(rd *redis.Client, repo *clients_repo.ClientsRepository, bds *bids.BidsStorage) *Hub {
	return &Hub{
		Chats:  make(map[int64]*Chat),
		redis:  rd,
		clRepo: repo,
		bds:    bds,
		Out:    make(chan tgbotapi.Chattable),
	}
}

func (c *Chat) SendTo(message tgbotapi.Update) {
	c.in <- message
}

func (h *Hub) GetChat(id int64, tgUsername string, writer chan tgbotapi.Chattable) (chat *Chat, ok bool) {
	c, ok := h.Chats[id]
	if !ok {
		c = &Chat{
			ID:         id,
			TGUserName: tgUsername,
			client:     nil,
			in:         make(chan tgbotapi.Update, 10),
			out:        writer,
			redis:      h.redis,
			clRepo:     h.clRepo,
			bds:        h.bds,
		}
		h.SetChat(c)
		ok = true
	}
	return c, ok
}

type ChatInfo struct {
	ID         int64
	TGUsername string
	Client     *clients_repo.Client
	Bids       []models.Bid
	Sent       bool
}

func (h *Hub) GetChats() []ChatInfo {
	ci := make([]ChatInfo, 0)
	for _, c := range h.Chats {
		cn := ChatInfo{
			ID:         c.ID,
			TGUsername: c.TGUserName,
			Client:     c.client,
		}
		ci = append(ci, cn)
	}
	return ci
}

func (h *Hub) GetChatInfoById(id int64) ChatInfo {
	for _, c := range h.Chats {
		if c.ID == id {
			tgun := "unregistered"
			if c.client != nil {
				tgun = c.client.TgUsername
			}
			return ChatInfo{
				ID:         c.ID,
				TGUsername: tgun,
				Client:     c.client,
				Bids:       nil,
			}
		}
	}
	return ChatInfo{}
}

func (h *Hub) GetChatById(id int64) Chat {
	for _, c := range h.Chats {
		if c.ID == id {
			return *c
		}
	}
	return Chat{}
}
func (h *Hub) SendTo(id int64, tgname string, message string) {
	cl, ok := h.Chats[id]
	if ok {
		msg := tgbotapi.NewMessage(id, message)
		cl.out <- msg
		if cl.client != nil {
			cl.client.Messages = append(cl.client.Messages, clients_repo.Message{
				Author: "Мы",
				Text:   message,
				Date:   time.Now(),
			})
			h.clRepo.SetClient(cl.client.Phone, *cl.client)
		}
		log.Info().Str("tgname", tgname).Str("message", message).Msg("hub sendto")
	} else {
		h.Out <- tgbotapi.NewMessage(id, message)
	}
}

func (h *Hub) SendToAll(message string) {
	for _, cl := range h.clRepo.GetAllWithId() {
		msg := tgbotapi.NewMessage(cl.TgUserId, message)
		h.Out <- msg
	}
	log.Info().Str("message", message).Msg("hub sendtoall")
}

func (h *Hub) SetChat(c *Chat) {
	h.Chats[c.ID] = c
	go c.Run(func() {
		log.Info().Str("tgusername", c.TGUserName).Msg("return chat callback")
		delete(h.Chats, c.ID)
	})
}
