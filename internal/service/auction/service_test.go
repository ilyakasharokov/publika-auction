package auctionsvc_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"publika-auction/internal/domain"
	"publika-auction/internal/lock"
	"publika-auction/internal/repo/cache"
	"publika-auction/internal/repo/mock"
	auctionsvc "publika-auction/internal/service/auction"
	bidsvc "publika-auction/internal/service/bid"
)

type noopEvents struct{}

func (e *noopEvents) Publish(_ bidsvc.Event) {}

func newTestAuctionService() (*auctionsvc.Service, *mock.AuctionRepo) {
	auctionRepo := mock.NewAuctionRepo()
	lotRepo := mock.NewLotRepo()
	bidRepo := mock.NewBidRepo()
	bidCache := cache.NewBidCache()
	lk := lock.NewMutexLock()
	bs := bidsvc.New(bidRepo, lotRepo, bidCache, lk, nil, &noopEvents{})
	svc := auctionsvc.New(auctionRepo, lotRepo, bs, &noopEvents{})
	return svc, auctionRepo
}

func makeAuction(slug string) *domain.Auction {
	return &domain.Auction{
		ID:        uuid.New().String(),
		Slug:      slug,
		Title:     slug,
		Status:    domain.AuctionDraft,
		BidStep:   100,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestActivate_SingleAuction(t *testing.T) {
	svc, repo := newTestAuctionService()
	ctx := context.Background()

	a := makeAuction("spring")
	repo.Create(ctx, a)

	if err := svc.Activate(ctx, a.ID); err != nil {
		t.Fatalf("activate failed: %v", err)
	}

	got, _ := repo.GetByID(ctx, a.ID)
	if got.Status != domain.AuctionActive {
		t.Fatalf("expected active, got %s", got.Status)
	}
}

func TestActivate_EndsOtherActive(t *testing.T) {
	svc, repo := newTestAuctionService()
	ctx := context.Background()

	a1 := makeAuction("spring")
	a2 := makeAuction("summer")
	repo.Create(ctx, a1)
	repo.Create(ctx, a2)

	// Activate first auction.
	if err := svc.Activate(ctx, a1.ID); err != nil {
		t.Fatal(err)
	}

	// Activate second — first must be ended automatically.
	if err := svc.Activate(ctx, a2.ID); err != nil {
		t.Fatal(err)
	}

	got1, _ := repo.GetByID(ctx, a1.ID)
	got2, _ := repo.GetByID(ctx, a2.ID)

	if got1.Status != domain.AuctionEnded {
		t.Errorf("auction 1: expected ended, got %s", got1.Status)
	}
	if got2.Status != domain.AuctionActive {
		t.Errorf("auction 2: expected active, got %s", got2.Status)
	}
}

func TestActivate_Idempotent(t *testing.T) {
	svc, repo := newTestAuctionService()
	ctx := context.Background()

	a := makeAuction("spring")
	repo.Create(ctx, a)

	svc.Activate(ctx, a.ID)
	if err := svc.Activate(ctx, a.ID); err != nil {
		t.Fatalf("second activate should be idempotent, got: %v", err)
	}
}

func TestGetActiveAuction_ReturnsNilWhenNone(t *testing.T) {
	svc, repo := newTestAuctionService()
	ctx := context.Background()

	a := makeAuction("spring")
	repo.Create(ctx, a)
	// no activation

	got, err := svc.GetActiveAuction(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}
