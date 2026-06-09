package auctionsvc

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"publika-auction/internal/domain"
	"publika-auction/internal/metrics"
	"publika-auction/internal/repo"
	bidsvc "publika-auction/internal/service/bid"
)

type Service struct {
	auctionRepo repo.AuctionRepo
	lotRepo     repo.LotRepo
	bidSvc      *bidsvc.Service
	events      bidsvc.EventBroadcaster
}

func New(ar repo.AuctionRepo, lr repo.LotRepo, bs *bidsvc.Service, events bidsvc.EventBroadcaster) *Service {
	return &Service{auctionRepo: ar, lotRepo: lr, bidSvc: bs, events: events}
}

type CreateRequest struct {
	Slug        string
	Title       string
	Description string
	BidStep     int
	StartAt     time.Time
	EndAt       time.Time
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*domain.Auction, error) {
	if req.Slug == "" {
		return nil, errors.New("slug required")
	}
	a := &domain.Auction{
		ID:          uuid.New().String(),
		Slug:        req.Slug,
		Title:       req.Title,
		Description: req.Description,
		BidStep:     req.BidStep,
		Status:      domain.AuctionDraft,
		StartAt:     req.StartAt,
		EndAt:       req.EndAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.auctionRepo.Create(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *Service) Activate(ctx context.Context, id string) error {
	a, err := s.auctionRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if a.Status == domain.AuctionActive {
		return nil
	}
	// End any currently active auction before activating this one.
	list, err := s.auctionRepo.List(ctx)
	if err != nil {
		return err
	}
	for _, other := range list {
		if other.ID != id && other.Status == domain.AuctionActive {
			if err := s.auctionRepo.UpdateStatus(ctx, other.ID, domain.AuctionEnded); err != nil {
				log.Err(err).Str("auction_id", other.ID).Msg("activate: failed to end previous active auction")
			} else {
				log.Info().Str("ended", other.Slug).Str("activated", a.Slug).Msg("previous active auction ended automatically")
				metrics.AuctionsActive.Dec()
			}
			if s.events != nil {
				s.events.Publish(bidsvc.Event{Type: "auction_ended", AuctionID: other.ID})
			}
		}
	}
	if err := s.auctionRepo.UpdateStatus(ctx, id, domain.AuctionActive); err != nil {
		return err
	}
	lots, err := s.lotRepo.ListByAuction(ctx, id)
	if err != nil {
		log.Err(err).Str("auction_id", id).Msg("activate: list lots error")
	}
	for _, lot := range lots {
		s.bidSvc.HydrateLot(id, a.Slug, lot, a.BidStep)
	}
	metrics.AuctionsActive.Inc()
	metrics.LotsActive.Add(float64(len(lots)))
	if s.events != nil {
		s.events.Publish(bidsvc.Event{Type: "auction_activated", AuctionID: id})
	}
	return nil
}

func (s *Service) End(ctx context.Context, id string) error {
	if err := s.auctionRepo.UpdateStatus(ctx, id, domain.AuctionEnded); err != nil {
		return err
	}
	metrics.AuctionsActive.Dec()
	if s.events != nil {
		s.events.Publish(bidsvc.Event{Type: "auction_ended", AuctionID: id})
	}
	return nil
}

func (s *Service) List(ctx context.Context) ([]*domain.Auction, error) {
	return s.auctionRepo.List(ctx)
}

func (s *Service) GetBySlug(ctx context.Context, slug string) (*domain.Auction, error) {
	return s.auctionRepo.GetBySlug(ctx, slug)
}

func (s *Service) GetByID(ctx context.Context, id string) (*domain.Auction, error) {
	return s.auctionRepo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, a *domain.Auction) error {
	return s.auctionRepo.Update(ctx, a)
}

func (s *Service) GetActiveAuction(ctx context.Context) (*domain.Auction, error) {
	list, err := s.auctionRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, a := range list {
		if a.Status == domain.AuctionActive {
			return a, nil
		}
	}
	return nil, nil
}
