//go:build load

package test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"publika-auction/internal/domain"
	auctionsvc "publika-auction/internal/service/auction"
	bidsvc "publika-auction/internal/service/bid"
)

func bid(a *domain.Auction, lot *domain.Lot, clientID, phone string, tgID int64, amount int) bidsvc.PlaceBidRequest {
	return bidsvc.PlaceBidRequest{
		AuctionID: a.ID, AuctionSlug: a.Slug,
		LotID: lot.ID, LotNum: lot.Num,
		ClientID: clientID, Phone: phone, TgUserID: tgID,
		Amount:   amount,
	}
}

// TestLoad_BidsPerSecond measures sustained bid throughput over 10 seconds
// across N lots in parallel, using real Redis distributed lock.
func TestLoad_BidsPerSecond(t *testing.T) {
	d, cleanup := setup(t)
	defer cleanup()
	bs, as := d.bs, d.as
	ctx := context.Background()

	const (
		numLots    = 10
		goroutines = 50
		duration   = 10 * time.Second
	)

	a, _ := as.Create(ctx, auctionsvc.CreateRequest{
		Slug: "load-" + uuid.New().String()[:8], Title: "Load test",
		BidStep: 1, StartAt: time.Now(), EndAt: time.Now().Add(time.Hour),
	})
	as.Activate(ctx, a.ID)
	a.Status = domain.AuctionActive

	lots := make([]*domain.Lot, numLots)
	for i := range lots {
		lot := &domain.Lot{
			ID: uuid.New().String(), AuctionID: a.ID, Num: i + 1,
			Title: fmt.Sprintf("Lot %d", i+1), StartPrice: 1000,
			Status: domain.LotActive, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		d.lotRepo.Create(ctx, lot)
		bs.HydrateLot(a.ID, a.Slug, lot, 1)
		lots[i] = lot
	}

	var totalOK, totalErr atomic.Int64
	var wg sync.WaitGroup
	deadline := time.Now().Add(duration)

	for li, lot := range lots {
		for g := 0; g < goroutines; g++ {
			wg.Add(1)
			li, lot, g := li, lot, g
			go func() {
				defer wg.Done()
				var amount atomic.Int64
				amount.Store(int64(1000 + g*100000))
				for time.Now().Before(deadline) {
					amt := int(amount.Add(int64(goroutines)))
					_, err := bs.PlaceBid(ctx, bid(a, lot,
						fmt.Sprintf("c%d-%d", li, g),
						fmt.Sprintf("+7%03d%06d", li, g),
						int64(li*1000+g+1), amt))
					if err == nil {
						totalOK.Add(1)
					} else {
						totalErr.Add(1)
					}
				}
			}()
		}
	}

	wg.Wait()

	ok, errs := totalOK.Load(), totalErr.Load()
	t.Logf("Duration:    %s", duration)
	t.Logf("Lots:        %d × %d goroutines", numLots, goroutines)
	t.Logf("Accepted:    %d (%.0f/s)", ok, float64(ok)/duration.Seconds())
	t.Logf("Rejected:    %d (lock contention / too-low)", errs)

	if ok == 0 {
		t.Fatal("zero successful bids")
	}
}

// TestLoad_SingleLotContention — 100 goroutines fighting for one lot.
// Worst-case Redis lock contention.
func TestLoad_SingleLotContention(t *testing.T) {
	d, cleanup := setup(t)
	defer cleanup()
	bs, as := d.bs, d.as
	ctx := context.Background()

	const (
		goroutines = 100
		duration   = 5 * time.Second
	)

	a, _ := as.Create(ctx, auctionsvc.CreateRequest{
		Slug: "contention-" + uuid.New().String()[:8], Title: "Contention test",
		BidStep: 1, StartAt: time.Now(), EndAt: time.Now().Add(time.Hour),
	})
	as.Activate(ctx, a.ID)
	a.Status = domain.AuctionActive

	lot := &domain.Lot{
		ID: uuid.New().String(), AuctionID: a.ID, Num: 1,
		Title: "Hot lot", StartPrice: 1000,
		Status: domain.LotActive, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	d.lotRepo.Create(ctx, lot)
	bs.HydrateLot(a.ID, a.Slug, lot, 1)

	var totalOK, totalErr atomic.Int64
	var globalAmount atomic.Int64
	globalAmount.Store(1000)
	var wg sync.WaitGroup
	deadline := time.Now().Add(duration)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		g := g
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				amt := int(globalAmount.Add(1))
				_, err := bs.PlaceBid(ctx, bid(a, lot,
					fmt.Sprintf("c%d", g),
					fmt.Sprintf("+7%010d", g),
					int64(g+1), amt))
				if err == nil {
					totalOK.Add(1)
				} else {
					totalErr.Add(1)
				}
			}
		}()
	}

	wg.Wait()

	ok := totalOK.Load()
	t.Logf("Single lot, %d goroutines, %s", goroutines, duration)
	t.Logf("Accepted:  %d (%.0f/s)", ok, float64(ok)/duration.Seconds())
	t.Logf("Rejected:  %d", totalErr.Load())
}
