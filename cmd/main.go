package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"publika-auction/cmd/configuration"
	"publika-auction/internal/admin"
	"publika-auction/internal/admin/handlers"
	"publika-auction/internal/hub"
	"publika-auction/internal/lock"
	"publika-auction/internal/metrics"
	"publika-auction/internal/repo/cache"
	mongorepo "publika-auction/internal/repo/mongo"
	auctionsvc "publika-auction/internal/service/auction"
	bidsvc "publika-auction/internal/service/bid"
	clientsvc "publika-auction/internal/service/client"
	lotsvc "publika-auction/internal/service/lot"
	"publika-auction/internal/tg"
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

func main() {
	cfg := configuration.New()
	metrics.Register()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// MongoDB
	db, err := mongorepo.Connect(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatal().Err(err).Msg("mongodb connect failed")
	}
	log.Info().Str("uri", cfg.MongoURI).Str("db", cfg.MongoDB).Msg("mongodb connected")

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.REDIS_ADDR,
		Password: cfg.REDIS_PWD,
		DB:       cfg.REDIS_DB,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Warn().Err(err).Msg("redis ping failed — distributed locking disabled")
	} else {
		log.Info().Str("addr", cfg.REDIS_ADDR).Msg("redis connected")
	}

	// Repos
	auctionRepo := mongorepo.NewAuctionRepo(db)
	lotRepo := mongorepo.NewLotRepo(db)
	bidRepo := mongorepo.NewBidRepo(db)
	clientRepo := mongorepo.NewClientRepo(db)

	// Caches
	bidCache := cache.NewBidCache()
	clientCache := cache.NewClientCache()

	// SSE hub
	sseHub := handlers.NewSSEHub()

	// Distributed lock
	redisLock := lock.New(rdb)

	// Services
	bidService := bidsvc.New(bidRepo, lotRepo, bidCache, redisLock, nil, sseHub)
	auctionService := auctionsvc.New(auctionRepo, lotRepo, bidService, sseHub)
	lotService := lotsvc.New(lotRepo)
	clientService := clientsvc.New(clientRepo, clientCache, nil)

	// Hub
	h := hub.New(bidService, clientService)

	// Bot manager — handles hot-plug connect/disconnect from admin panel
	botManager := tg.NewManager(h, bidService, clientService)

	// Auto-connect from env if token is provided
	if cfg.TG_TOKEN != "" {
		if err := botManager.Connect(cfg.TG_TOKEN, cfg.TG_ENDPOINT); err != nil {
			log.Warn().Err(err).Msg("telegram bot auto-connect failed")
		} else {
			h.SetBroadcaster(botManager.Queue())
		}
	} else {
		log.Info().Msg("no TG_TOKEN set — configure bot via /admin/settings")
	}

	// Load all clients into cache
	if err := clientService.LoadAll(ctx); err != nil {
		log.Warn().Err(err).Msg("load clients failed")
	}

	// Restore active auction into hub
	activeAuction, err := auctionService.GetActiveAuction(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("get active auction failed")
	}
	if activeAuction != nil {
		lots, err := lotService.ListByAuction(ctx, activeAuction.ID)
		if err != nil {
			log.Warn().Err(err).Msg("list lots failed")
		}
		for _, lot := range lots {
			bidService.HydrateLot(activeAuction.ID, activeAuction.Slug, lot, activeAuction.BidStep)
		}
		h.SetActiveAuction(activeAuction, lots)
		log.Info().Str("slug", activeAuction.Slug).Msg("active auction restored")
	}

	// Admin server
	srv := admin.NewServer(admin.Config{
		Addr:          cfg.ADDR,
		AdminUser:     cfg.AdminUser,
		AdminPassword: cfg.AdminPassword,
		SessionSecret: cfg.SessionSecret,
	}, auctionService, lotService, bidService, clientService, bidCache, sseHub, h, botManager)

	go func() {
		log.Info().Str("addr", cfg.ADDR).Msg("admin server starting")
		if err := srv.ListenAndServe(); err != nil {
			log.Err(err).Msg("admin server error")
		}
	}()

	// Graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Info().Msg("shutting down...")
	botManager.Disconnect()
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)
	log.Info().Msg("stopped")
}
