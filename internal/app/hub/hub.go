package hub

import (
	"github.com/go-redis/redis/v8"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"
	"publika-auction/internal/app/bids"
	clients_repo "publika-auction/internal/app/clients-repo"
	"publika-auction/internal/app/models"
)

type Hub struct {
	Chats map[int64]*Chat

	redis  *redis.Client
	clRepo *clients_repo.ClientsRepository
	bds    *bids.BidsStorage
}

func New(rd *redis.Client, repo *clients_repo.ClientsRepository, bds *bids.BidsStorage) *Hub {
	return &Hub{
		Chats:  make(map[int64]*Chat),
		redis:  rd,
		clRepo: repo,
		bds:    bds,
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

func (h *Hub) GetChatById(id int64) ChatInfo {
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

func (h *Hub) SendTo(id int64, tgname string, message string) {
	cl, ok := h.Chats[id]
	if ok {
		msg := tgbotapi.NewMessage(id, message)
		cl.out <- msg
		log.Info().Str("tgname", tgname).Str("message", message).Msg("hub sendto")
	}
}

func (h *Hub) SendToAll(message string) {
	for _, c := range h.Chats {
		msg := tgbotapi.NewMessage(c.ID, message)
		c.out <- msg
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