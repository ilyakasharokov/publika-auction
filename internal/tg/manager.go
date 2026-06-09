package tg

import (
	"context"
	"sync"

	"github.com/rs/zerolog/log"

	"publika-auction/internal/hub"
	"publika-auction/internal/tgqueue"
)

type NotifierSetter interface {
	SetNotifier(n tgqueue.NotifierIface)
}

type Status struct {
	Connected bool
	Username  string
	Token     string
	Endpoint  string
	ErrMsg    string
}

type Manager struct {
	mu     sync.RWMutex
	bot    *Bot
	cancel context.CancelFunc

	hub       *hub.Hub
	bidSvc    NotifierSetter
	clientSvc NotifierSetter

	status Status
}

func NewManager(h *hub.Hub, bidSvc, clientSvc NotifierSetter) *Manager {
	return &Manager{
		hub:       h,
		bidSvc:    bidSvc,
		clientSvc: clientSvc,
		status:    Status{Endpoint: "https://api.telegram.org/bot%s/%s"},
	}
}

func (m *Manager) Connect(token, endpoint string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
		m.bot = nil
	}

	if endpoint == "" {
		endpoint = "https://api.telegram.org/bot%s/%s"
	}

	bot, err := New(Config{Token: token, Endpoint: endpoint}, m.hub)
	if err != nil {
		m.status = Status{
			Connected: false,
			Token:     token,
			Endpoint:  endpoint,
			ErrMsg:    err.Error(),
		}
		log.Err(err).Msg("bot manager: connect failed")
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.bot = bot
	m.cancel = cancel
	m.status = Status{
		Connected: true,
		Username:  bot.Username(),
		Token:     token,
		Endpoint:  endpoint,
	}

	m.bidSvc.SetNotifier(bot.Queue())
	m.clientSvc.SetNotifier(bot.Queue())

	go bot.Start(ctx)
	log.Info().Str("username", bot.Username()).Msg("bot manager: connected")
	return nil
}

func (m *Manager) Disconnect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	prev := m.status
	m.bot = nil
	m.status = Status{
		Connected: false,
		Token:     prev.Token,
		Endpoint:  prev.Endpoint,
	}
	log.Info().Msg("bot manager: disconnected")
}

// Queue returns the active TG send queue, or nil if not connected.
func (m *Manager) Queue() *tgqueue.Queue {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.bot == nil {
		return nil
	}
	return m.bot.Queue()
}

func (m *Manager) GetStatus() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}
