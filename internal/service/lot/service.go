package lotsvc

import (
	"context"
	"time"

	"github.com/google/uuid"

	"publika-auction/internal/domain"
	"publika-auction/internal/repo"
)

type Service struct {
	lotRepo repo.LotRepo
}

func New(lr repo.LotRepo) *Service {
	return &Service{lotRepo: lr}
}

type CreateRequest struct {
	AuctionID   string
	Num         int
	Title       string
	Description string
	PhotoURL    string
	StartPrice  int
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*domain.Lot, error) {
	lot := &domain.Lot{
		ID:          uuid.New().String(),
		AuctionID:   req.AuctionID,
		Num:         req.Num,
		Title:       req.Title,
		Description: req.Description,
		PhotoURL:    req.PhotoURL,
		StartPrice:  req.StartPrice,
		Status:      domain.LotActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.lotRepo.Create(ctx, lot); err != nil {
		return nil, err
	}
	return lot, nil
}

func (s *Service) ListByAuction(ctx context.Context, auctionID string) ([]*domain.Lot, error) {
	return s.lotRepo.ListByAuction(ctx, auctionID)
}

func (s *Service) GetByID(ctx context.Context, id string) (*domain.Lot, error) {
	return s.lotRepo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, lot *domain.Lot) error {
	return s.lotRepo.Update(ctx, lot)
}

func (s *Service) Pull(ctx context.Context, lotID string) error {
	lot, err := s.lotRepo.GetByID(ctx, lotID)
	if err != nil {
		return err
	}
	lot.Status = domain.LotPulled
	return s.lotRepo.Update(ctx, lot)
}
