package bidsvc_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"

	bidsvc "publika-auction/internal/service/bid"
)

// BenchmarkPlaceBid_Sequential — single goroutine, each bid higher than last.
// Measures lock acquire + cache update + async insert throughput.
func BenchmarkPlaceBid_Sequential(b *testing.B) {
	svc, _, _ := newTestService()
	auctionID := uuid.New().String()
	lot := seedLot(svc, nil, auctionID, "bench", 1)

	var amount int64 = 1000
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a := atomic.AddInt64(&amount, 1)
		svc.PlaceBid(context.Background(), bidsvc.PlaceBidRequest{
			AuctionID: auctionID, AuctionSlug: "bench",
			LotID: lot.ID, LotNum: 1,
			ClientID: fmt.Sprintf("c%d", i),
			Phone:    fmt.Sprintf("+7%010d", i),
			TgUserID: int64(i + 1),
			Amount:   int(a),
		})
	}
}

// BenchmarkPlaceBid_Parallel — concurrent goroutines bidding on separate lots.
// Measures per-lot lock contention at full GOMAXPROCS parallelism.
func BenchmarkPlaceBid_Parallel(b *testing.B) {
	svc, _, _ := newTestService()
	auctionID := uuid.New().String()

	// Pre-create one lot per goroutine to avoid cross-lot contention.
	lots := make([]string, 128)
	for i := range lots {
		l := seedLot(svc, nil, auctionID, "bench", 1)
		lots[i] = l.ID
	}

	var counter atomic.Int64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		idx := int(counter.Add(1)) % len(lots)
		var amount int64 = 1000
		for pb.Next() {
			a := atomic.AddInt64(&amount, 1)
			svc.PlaceBid(context.Background(), bidsvc.PlaceBidRequest{
				AuctionID: auctionID, AuctionSlug: "bench",
				LotID: lots[idx], LotNum: idx + 1,
				ClientID: fmt.Sprintf("c%d-%d", idx, a),
				Phone:    fmt.Sprintf("+7%010d", a),
				TgUserID: a,
				Amount:   int(a),
			})
		}
	})
}

// BenchmarkPlaceBid_Contention — all goroutines fight for the same lot.
// Worst-case: measures lock serialization overhead.
func BenchmarkPlaceBid_Contention(b *testing.B) {
	svc, _, _ := newTestService()
	auctionID := uuid.New().String()
	lot := seedLot(svc, nil, auctionID, "bench", 1)

	var amount atomic.Int64
	amount.Store(1000)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			a := amount.Add(1)
			svc.PlaceBid(context.Background(), bidsvc.PlaceBidRequest{
				AuctionID: auctionID, AuctionSlug: "bench",
				LotID: lot.ID, LotNum: 1,
				ClientID: fmt.Sprintf("c%d", a),
				Phone:    fmt.Sprintf("+7%010d", a),
				TgUserID: a,
				Amount:   int(a),
			})
		}
	})
}
