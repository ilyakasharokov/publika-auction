package clientsvc

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"publika-auction/internal/domain"
	mongorepo "publika-auction/internal/repo/mongo"
	"publika-auction/internal/repo/cache"
	"publika-auction/internal/tgqueue"
)

type Service struct {
	repo    *mongorepo.ClientRepo
	cache   *cache.ClientCache
	notifier Notifier
}

type Notifier = tgqueue.NotifierIface

func New(repo *mongorepo.ClientRepo, cache *cache.ClientCache, notifier Notifier) *Service {
	return &Service{repo: repo, cache: cache, notifier: notifier}
}

func (s *Service) SetNotifier(n Notifier) {
	s.notifier = n
}

func (s *Service) LoadAll(ctx context.Context) error {
	clients, err := s.repo.List(ctx)
	if err != nil {
		return err
	}
	for _, c := range clients {
		s.cache.Set(c)
	}
	return nil
}

func (s *Service) GetByPhone(ctx context.Context, phone string) (*domain.Client, bool) {
	if c, ok := s.cache.GetByPhone(phone); ok {
		return c, true
	}
	c, err := s.repo.GetByPhone(ctx, phone)
	if err != nil {
		return nil, false
	}
	s.cache.Set(c)
	return c, true
}

func (s *Service) GetByTgID(ctx context.Context, tgID int64) (*domain.Client, bool) {
	if c, ok := s.cache.GetByTgID(tgID); ok {
		return c, true
	}
	c, err := s.repo.GetByTgID(ctx, tgID)
	if err != nil {
		return nil, false
	}
	s.cache.Set(c)
	return c, true
}

func (s *Service) RegisterOrUpdate(ctx context.Context, c *domain.Client) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
		c.CreatedAt = time.Now()
	}
	c.UpdatedAt = time.Now()
	s.cache.Set(c)
	go func() {
		if err := s.repo.Upsert(context.Background(), c); err != nil {
			log.Err(err).Str("phone", c.Phone).Msg("client upsert failed")
		}
	}()
	return nil
}

func (s *Service) Block(ctx context.Context, phone string) error {
	if c, ok := s.cache.GetByPhone(phone); ok {
		c.IsBlocked = true
		s.cache.Set(c)
	}
	return s.repo.Block(ctx, phone)
}

func (s *Service) SendMessage(ctx context.Context, tgID int64, text string) error {
	s.notifier.Send(tgID, text)
	msg := &domain.ChatMessage{
		TgUserID:  tgID,
		Author:    "admin",
		Text:      text,
		CreatedAt: time.Now(),
	}
	return s.repo.InsertMessage(ctx, msg)
}

func (s *Service) RecordMessage(ctx context.Context, tgID int64, author, text string) {
	msg := &domain.ChatMessage{
		TgUserID:  tgID,
		Author:    author,
		Text:      text,
		CreatedAt: time.Now(),
	}
	go func() {
		if err := s.repo.InsertMessage(context.Background(), msg); err != nil {
			log.Err(err).Int64("tg_id", tgID).Msg("record message failed")
		}
	}()
}

func (s *Service) GetMessages(ctx context.Context, tgID int64) ([]*domain.ChatMessage, error) {
	return s.repo.ListMessages(ctx, tgID)
}

func (s *Service) ListAll(ctx context.Context) ([]*domain.Client, error) {
	return s.repo.List(ctx)
}

func (s *Service) BroadcastToAll(ctx context.Context, text string) {
	all := s.cache.GetAllWithTgID()
	for _, c := range all {
		s.notifier.Send(c.TgUserID, text)
	}
}
