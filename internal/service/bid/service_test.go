package bidsvc_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"publika-auction/internal/domain"
	"publika-auction/internal/lock"
	"publika-auction/internal/repo/cache"
	"publika-auction/internal/repo/mock"
	bidsvc "publika-auction/internal/service/bid"
)

// helpers ----------------------------------------------------------------

type noopNotifier struct{}

func (n *noopNotifier) Send(_ int64, _ string) {}

type noopEvents struct{}

func (e *noopEvents) Publish(_ bidsvc.Event) {}

func newTestService() (*bidsvc.Service, *mock.BidRepo, *cache.BidCache) {
	bidRepo := mock.NewBidRepo()
	lotRepo := mock.NewLotRepo()
	bidCache := cache.NewBidCache()
	lk := lock.NewMutexLock()
	svc := bidsvc.New(bidRepo, lotRepo, bidCache, lk, &noopNotifier{}, &noopEvents{})
	return svc, bidRepo, bidCache
}

func seedLot(svc *bidsvc.Service, _ *cache.BidCache, auctionID, slug string, bidStep int) *domain.Lot {
	lot := &domain.Lot{
		ID:         uuid.New().String(),
		AuctionID:  auctionID,
		Num:        1,
		Title:      "Test lot",
		StartPrice: 1000,
		Status:     domain.LotActive,
	}
	svc.HydrateLot(auctionID, slug, lot, bidStep)
	return lot
}

// tests ------------------------------------------------------------------

func TestPlaceBid_Success(t *testing.T) {
	svc, bidRepo, _ := newTestService()
	auctionID := uuid.New().String()
	lot := seedLot(svc, nil, auctionID, "test", 100)

	_, err := svc.PlaceBid(context.Background(), bidsvc.PlaceBidRequest{
		AuctionID:   auctionID,
		AuctionSlug: "test",
		LotID:       lot.ID,
		LotNum:      1,
		ClientID:    "client-1",
		Phone:       "+79001234567",
		TgUserID:    1,
		Amount:      1100,
	})
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	time.Sleep(10 * time.Millisecond) // async insert
	if bidRepo.Count() != 1 {
		t.Fatalf("expected 1 bid in repo, got %d", bidRepo.Count())
	}
}

func TestPlaceBid_TooLow(t *testing.T) {
	svc, _, _ := newTestService()
	auctionID := uuid.New().String()
	lot := seedLot(svc, nil, auctionID, "test", 100)

	var tooLow bidsvc.ErrBidTooLowDetail
	_, err := svc.PlaceBid(context.Background(), bidsvc.PlaceBidRequest{
		AuctionID: auctionID, AuctionSlug: "test",
		LotID: lot.ID, LotNum: 1,
		ClientID: "c1", Phone: "+7900", TgUserID: 1,
		Amount: 1050, // less than 1000 + 100
	})
	if !errors.As(err, &tooLow) {
		t.Fatalf("expected ErrBidTooLowDetail, got: %v", err)
	}
	if tooLow.Current != 1000 {
		t.Fatalf("expected current=1000, got %d", tooLow.Current)
	}
}

func TestPlaceBid_Sequential(t *testing.T) {
	svc, bidRepo, _ := newTestService()
	auctionID := uuid.New().String()
	lot := seedLot(svc, nil, auctionID, "test", 100)

	for i := 1; i <= 5; i++ {
		_, err := svc.PlaceBid(context.Background(), bidsvc.PlaceBidRequest{
			AuctionID: auctionID, AuctionSlug: "test",
			LotID: lot.ID, LotNum: 1,
			ClientID: fmt.Sprintf("client-%d", i),
			Phone:    fmt.Sprintf("+7900000000%d", i),
			TgUserID: int64(i),
			Amount:   1000 + i*100,
		})
		if err != nil {
			t.Fatalf("bid %d failed: %v", i, err)
		}
	}

	state, _ := svc.GetLotState(auctionID, lot.ID)
	if state.MaxAmount != 1500 {
		t.Fatalf("expected max=1500, got %d", state.MaxAmount)
	}
	if state.BidCount != 5 {
		t.Fatalf("expected 5 bids in cache, got %d", state.BidCount)
	}
	time.Sleep(20 * time.Millisecond)
	if bidRepo.Count() != 5 {
		t.Fatalf("expected 5 bids in repo, got %d", bidRepo.Count())
	}
}

// TestPlaceBid_Concurrent verifies that under concurrent load with identical
// bid amounts, exactly one bid succeeds per price level and the cache
// reflects a consistent final state with no lost updates.
func TestPlaceBid_Concurrent(t *testing.T) {
	svc, bidRepo, _ := newTestService()
	auctionID := uuid.New().String()
	lot := seedLot(svc, nil, auctionID, "test", 100)

	const goroutines = 50
	var (
		wg      sync.WaitGroup
		success atomic.Int64
		reject  atomic.Int64
	)

	// All goroutines try to place the same first bid (1100).
	// Only one should win; the rest must get ErrBidTooLow after re-check.
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			_, err := svc.PlaceBid(context.Background(), bidsvc.PlaceBidRequest{
				AuctionID: auctionID, AuctionSlug: "test",
				LotID: lot.ID, LotNum: 1,
				ClientID: fmt.Sprintf("client-%d", i),
				Phone:    fmt.Sprintf("+7900%06d", i),
				TgUserID: int64(i + 1),
				Amount:   1100,
			})
			if err == nil {
				success.Add(1)
			} else {
				reject.Add(1)
			}
		}()
	}
	wg.Wait()

	if success.Load() != 1 {
		t.Fatalf("expected exactly 1 successful bid, got %d (rejected: %d)", success.Load(), reject.Load())
	}

	state, _ := svc.GetLotState(auctionID, lot.ID)
	if state.MaxAmount != 1100 {
		t.Fatalf("expected max=1100, got %d", state.MaxAmount)
	}
	if state.BidCount != 1 {
		t.Fatalf("expected BidCount=1, got %d", state.BidCount)
	}

	time.Sleep(20 * time.Millisecond)
	if bidRepo.Count() != 1 {
		t.Fatalf("expected 1 bid in repo, got %d", bidRepo.Count())
	}
}

// TestPlaceBid_Escalating places 20 sequential bids each higher than the last.
// All must succeed and the cache must reflect the final price.
func TestPlaceBid_Escalating(t *testing.T) {
	svc, _, _ := newTestService()
	auctionID := uuid.New().String()
	lot := seedLot(svc, nil, auctionID, "test", 100)

	const n = 20
	for i := 0; i < n; i++ {
		_, err := svc.PlaceBid(context.Background(), bidsvc.PlaceBidRequest{
			AuctionID: auctionID, AuctionSlug: "test",
			LotID: lot.ID, LotNum: 1,
			ClientID: fmt.Sprintf("client-%d", i),
			Phone:    fmt.Sprintf("+7900%06d", i),
			TgUserID: int64(i + 1),
			Amount:   1000 + (i+1)*100,
		})
		if err != nil {
			t.Fatalf("bid %d failed: %v", i, err)
		}
	}

	state, _ := svc.GetLotState(auctionID, lot.ID)
	if state.MaxAmount != 1000+n*100 {
		t.Fatalf("expected max=%d, got %d", 1000+n*100, state.MaxAmount)
	}
	if state.BidCount != n {
		t.Fatalf("expected BidCount=%d, got %d", n, state.BidCount)
	}
}

func TestCancelBid(t *testing.T) {
	svc, _, _ := newTestService()
	auctionID := uuid.New().String()
	lot := seedLot(svc, nil, auctionID, "test", 100)

	bid, err := svc.PlaceBid(context.Background(), bidsvc.PlaceBidRequest{
		AuctionID: auctionID, AuctionSlug: "test",
		LotID: lot.ID, LotNum: 1,
		ClientID: "c1", Phone: "+7900", TgUserID: 1, Amount: 1100,
	})
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(20 * time.Millisecond) // let async repo insert land
	if err := svc.CancelBid(context.Background(), bid.ID); err != nil {
		t.Fatalf("cancel failed: %v", err)
	}

	state, _ := svc.GetLotState(auctionID, lot.ID)
	if state.MaxBidID == bid.ID {
		t.Fatal("cancelled bid is still the max bid in cache")
	}
}
