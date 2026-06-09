//go:build integration

package test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"

	"publika-auction/internal/domain"
	"publika-auction/internal/lock"
	"publika-auction/internal/repo/cache"
	mongorepo "publika-auction/internal/repo/mongo"
	auctionsvc "publika-auction/internal/service/auction"
	bidsvc "publika-auction/internal/service/bid"
)

type noopNotifier struct{}

func (n *noopNotifier) Send(_ int64, _ string) {}

type noopEvents struct{}

func (e *noopEvents) Publish(_ bidsvc.Event) {}

type testDeps struct {
	bs      *bidsvc.Service
	as      *auctionsvc.Service
	lotRepo *mongorepo.LotRepo
}

func setup(t *testing.T) (testDeps, func()) {
	t.Helper()
	ctx := context.Background()

	db, err := mongorepo.Connect(ctx, "mongodb://localhost:27017", "auction_test")
	if err != nil {
		t.Skipf("mongodb not available: %v", err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not available: %v", err)
	}

	bidRepo := mongorepo.NewBidRepo(db)
	lotRepo := mongorepo.NewLotRepo(db)
	auctionRepo := mongorepo.NewAuctionRepo(db)
	bidCache := cache.NewBidCache()
	redisLock := lock.New(rdb)

	bs := bidsvc.New(bidRepo, lotRepo, bidCache, redisLock, &noopNotifier{}, &noopEvents{})
	as := auctionsvc.New(auctionRepo, lotRepo, bs, &noopEvents{})

	cleanup := func() {
		db.Collection("auctions").Drop(ctx)
		db.Collection("lots").Drop(ctx)
		db.Collection("bids").Drop(ctx)
	}
	return testDeps{bs: bs, as: as, lotRepo: lotRepo}, cleanup
}

// TestIntegration_FullFlow creates an auction, adds a lot, activates, bids,
// then sells — verifying the full happy path against real infrastructure.
func TestIntegration_FullFlow(t *testing.T) {
	d, cleanup := setup(t)
	defer cleanup()
	bs, as := d.bs, d.as
	ctx := context.Background()

	// Create & activate auction.
	a, err := as.Create(ctx, auctionsvc.CreateRequest{
		Slug: "integration-" + uuid.New().String()[:8], Title: "Integration Auction",
		BidStep: 100, StartAt: time.Now(), EndAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := as.Activate(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	a.Status = domain.AuctionActive

	// Insert lot into MongoDB and hydrate cache.
	lot := &domain.Lot{
		ID: uuid.New().String(), AuctionID: a.ID, Num: 1,
		Title: "Test lot", StartPrice: 1000, Status: domain.LotActive,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := d.lotRepo.Create(ctx, lot); err != nil {
		t.Fatal(err)
	}
	bs.HydrateLot(a.ID, a.Slug, lot, a.BidStep)

	// Place 3 escalating bids.
	for i := 1; i <= 3; i++ {
		_, err := bs.PlaceBid(ctx, bidsvc.PlaceBidRequest{
			AuctionID: a.ID, AuctionSlug: a.Slug,
			LotID: lot.ID, LotNum: 1,
			ClientID: fmt.Sprintf("client-%d", i),
			Phone:    fmt.Sprintf("+7900%06d", i),
			TgUserID: int64(i),
			Amount:   1000 + i*100,
		})
		if err != nil {
			t.Fatalf("bid %d: %v", i, err)
		}
	}

	state, ok := bs.GetLotState(a.ID, lot.ID)
	if !ok {
		t.Fatal("lot state not found in cache")
	}
	if state.MaxAmount != 1300 {
		t.Fatalf("expected max=1300, got %d", state.MaxAmount)
	}

	// Sell the lot.
	time.Sleep(50 * time.Millisecond) // let async inserts land
	bids, err := bs.GetLotBids(ctx, lot.ID)
	if err != nil || len(bids) == 0 {
		t.Fatalf("no bids in DB: %v", err)
	}

	// Find the winning bid.
	var winner *domain.Bid
	for _, b := range bids {
		if b.Amount == 1300 {
			winner = b
		}
	}
	if winner == nil {
		t.Fatal("winning bid not found in DB")
	}
	if err := bs.SellLot(ctx, lot.ID, winner.ID); err != nil {
		t.Fatalf("sell lot: %v", err)
	}
}

// TestIntegration_ConcurrentBids fires 100 goroutines at the same lot
// and verifies only one bid wins per price level (no data races,
// no cache inconsistency) using real Redis distributed lock.
func TestIntegration_ConcurrentBids(t *testing.T) {
	d, cleanup := setup(t)
	defer cleanup()
	bs, as := d.bs, d.as
	ctx := context.Background()

	a, _ := as.Create(ctx, auctionsvc.CreateRequest{
		Slug: "concurrent-" + uuid.New().String()[:8], Title: "Concurrent test",
		BidStep: 100, StartAt: time.Now(), EndAt: time.Now().Add(time.Hour),
	})
	as.Activate(ctx, a.ID)
	a.Status = domain.AuctionActive

	lot := &domain.Lot{
		ID: uuid.New().String(), AuctionID: a.ID, Num: 1,
		Title: "Concurrent lot", StartPrice: 1000, Status: domain.LotActive,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	bs.HydrateLot(a.ID, a.Slug, lot, a.BidStep)

	const n = 100
	var success, reject atomic.Int64
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			_, err := bs.PlaceBid(ctx, bidsvc.PlaceBidRequest{
				AuctionID: a.ID, AuctionSlug: a.Slug,
				LotID: lot.ID, LotNum: 1,
				ClientID: fmt.Sprintf("c%d", i),
				Phone:    fmt.Sprintf("+7%010d", i),
				TgUserID: int64(i + 1),
				Amount:   1100, // all bid the same amount
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
		t.Fatalf("expected exactly 1 success, got %d (rejected: %d)", success.Load(), reject.Load())
	}

	state, _ := bs.GetLotState(a.ID, lot.ID)
	if state.BidCount != 1 {
		t.Fatalf("expected BidCount=1, got %d", state.BidCount)
	}

	time.Sleep(100 * time.Millisecond)
	bids, _ := bs.GetLotBids(ctx, lot.ID)
	if len(bids) != 1 {
		t.Fatalf("expected 1 bid in DB, got %d", len(bids))
	}
}
