package hub

import (
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"

	"publika-auction/internal/domain"
	bidsvc "publika-auction/internal/service/bid"
	clientsvc "publika-auction/internal/service/client"
)

// Broadcaster is anything that can send a TG message to a specific chat ID.
type Broadcaster interface {
	Send(tgID int64, text string)
}

type Hub struct {
	mu    sync.RWMutex
	chats map[int64]*Chat

	auction *domain.Auction
	lots    []*domain.Lot

	bidSvc      *bidsvc.Service
	clientSvc   *clientsvc.Service
	broadcaster Broadcaster
	Out         chan tgbotapi.Chattable
}

func New(bidSvc *bidsvc.Service, clientSvc *clientsvc.Service) *Hub {
	return &Hub{
		chats:     make(map[int64]*Chat),
		bidSvc:    bidSvc,
		clientSvc: clientSvc,
		Out:       make(chan tgbotapi.Chattable, 1024),
	}
}

// SetBroadcaster wires the TG queue so hub.SendToAll uses it instead of hub.Out.
func (h *Hub) SetBroadcaster(b Broadcaster) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.broadcaster = b
}

func (h *Hub) SetActiveAuction(a *domain.Auction, lots []*domain.Lot) {
	h.mu.Lock()
	h.auction = a
	h.lots = lots
	var notify []*Chat
	if a != nil && a.Status == domain.AuctionActive {
		for _, c := range h.chats {
			notify = append(notify, c)
		}
	}
	h.mu.Unlock()

	for _, c := range notify {
		c.sendLotsKeyboard()
	}
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
	msg := tgbotapi.NewMessage(id, text)
	if c != nil {
		select {
		case c.out <- msg:
		default:
			log.Warn().Int64("id", id).Msg("chat out channel full, message dropped")
		}
		return
	}
	select {
	case h.Out <- msg:
	default:
		log.Warn().Int64("id", id).Msg("hub.Out full, message dropped")
	}
}

// SendToAll sends text to every client with a known TG ID, routing through
// the TG queue (rate-limited) rather than hub.Out (unbounded blocking).
func (h *Hub) SendToAll(text string) {
	h.mu.RLock()
	b := h.broadcaster
	clients := h.clientSvc.CachedAllWithTgID()
	h.mu.RUnlock()

	for _, cl := range clients {
		if b != nil {
			b.Send(cl.TgUserID, text)
		} else {
			select {
			case h.Out <- tgbotapi.NewMessage(cl.TgUserID, text):
			default:
				log.Warn().Int64("id", cl.TgUserID).Msg("hub.Out full on broadcast, dropped")
			}
		}
	}
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
