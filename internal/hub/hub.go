package hub

import (
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"

	"publika-auction/internal/domain"
	bidsvc "publika-auction/internal/service/bid"
	clientsvc "publika-auction/internal/service/client"
)

type Hub struct {
	mu     sync.RWMutex
	chats  map[int64]*Chat

	auction *domain.Auction
	lots    []*domain.Lot

	bidSvc    *bidsvc.Service
	clientSvc *clientsvc.Service
	Out       chan tgbotapi.Chattable
}

func New(bidSvc *bidsvc.Service, clientSvc *clientsvc.Service) *Hub {
	return &Hub{
		chats:     make(map[int64]*Chat),
		bidSvc:    bidSvc,
		clientSvc: clientSvc,
		Out:       make(chan tgbotapi.Chattable, 256),
	}
}

func (h *Hub) SetActiveAuction(a *domain.Auction, lots []*domain.Lot) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.auction = a
	h.lots = lots
}

func (h *Hub) GetActiveAuction() *domain.Auction {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.auction
}

func (h *Hub) GetActiveLots() []*domain.Lot {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]*domain.Lot, len(h.lots))
	copy(out, h.lots)
	return out
}

func (h *Hub) GetLotByNum(num int) *domain.Lot {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, l := range h.lots {
		if l.Num == num {
			return l
		}
	}
	return nil
}

func (h *Hub) IsStarted() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.auction != nil && h.auction.Status == domain.AuctionActive
}

func (h *Hub) GetChat(id int64, tgUsername string, writer chan tgbotapi.Chattable) (*Chat, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	c, ok := h.chats[id]
	if !ok {
		c = &Chat{
			ID:         id,
			TGUserName: tgUsername,
			in:         make(chan tgbotapi.Update, 32),
			out:        writer,
			hub:        h,
			bidSvc:     h.bidSvc,
			clientSvc:  h.clientSvc,
		}
		h.chats[id] = c
		go c.Run(func() {
			h.mu.Lock()
			delete(h.chats, id)
			h.mu.Unlock()
			log.Info().Str("tgusername", tgUsername).Msg("chat removed")
		})
	}
	return c, ok
}

func (h *Hub) GetChatByID(id int64) *Chat {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.chats[id]
}

func (h *Hub) SendTo(id int64, text string) {
	h.mu.RLock()
	c := h.chats[id]
	h.mu.RUnlock()
	if c != nil {
		c.out <- tgbotapi.NewMessage(id, text)
		return
	}
	h.Out <- tgbotapi.NewMessage(id, text)
}

func (h *Hub) GetAllChats() []ChatInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var out []ChatInfo
	for _, c := range h.chats {
		out = append(out, ChatInfo{
			ID:         c.ID,
			TGUsername: c.TGUserName,
			Client:     c.client,
		})
	}
	return out
}

type ChatInfo struct {
	ID         int64
	TGUsername string
	Client     *domain.Client
}
