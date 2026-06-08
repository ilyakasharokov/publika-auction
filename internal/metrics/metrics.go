package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	BidsPlacedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "auction_bids_placed_total",
		Help: "Total number of bids placed",
	}, []string{"auction_slug", "lot_num"})

	BidsRejectedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "auction_bids_rejected_total",
		Help: "Total number of rejected bids",
	}, []string{"reason"})

	BidAmountRub = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "auction_bid_amount_rub",
		Help:    "Bid amounts in RUB",
		Buckets: []float64{1000, 5000, 10000, 50000, 100000, 500000},
	}, []string{"auction_slug"})

	LockAcquireDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "auction_redis_lock_acquire_duration_seconds",
		Help:    "Time to acquire Redis bid lock",
		Buckets: prometheus.DefBuckets,
	})

	LockContentionTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "auction_redis_lock_contention_total",
		Help: "Times a Redis bid lock was already held",
	})

	TGQueueDepth = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "auction_tg_queue_depth",
		Help: "Current TG send queue depth",
	})

	TGMessagesSentTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "auction_tg_messages_sent_total",
		Help: "TG messages sent by type",
	}, []string{"type"})

	TGMessagesDroppedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "auction_tg_messages_dropped_total",
		Help: "TG messages dropped (queue full)",
	})

	SSESubscribers = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "auction_sse_subscribers",
		Help: "Current number of SSE subscribers",
	})

	SSEEventsPublishedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "auction_sse_events_published_total",
		Help: "SSE events published by type",
	}, []string{"event_type"})

	AuctionsActive = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "auction_auctions_active",
		Help: "Number of active auctions",
	})

	LotsActive = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "auction_lots_active",
		Help: "Number of active lots",
	})

	LotsSoldTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "auction_lots_sold_total",
		Help: "Total lots sold",
	}, []string{"auction_slug"})

	HTTPRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "auction_http_request_duration_seconds",
		Help:    "HTTP request latency",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route", "status"})
)

func Register() {
	prometheus.MustRegister(
		BidsPlacedTotal,
		BidsRejectedTotal,
		BidAmountRub,
		LockAcquireDuration,
		LockContentionTotal,
		TGQueueDepth,
		TGMessagesSentTotal,
		TGMessagesDroppedTotal,
		SSESubscribers,
		SSEEventsPublishedTotal,
		AuctionsActive,
		LotsActive,
		LotsSoldTotal,
		HTTPRequestDuration,
	)
}
