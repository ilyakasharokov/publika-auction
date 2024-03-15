package main

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"publika-auction/cmd/configuration"
	bids2 "publika-auction/internal/app/bids"
	clients_repo "publika-auction/internal/app/clients-repo"
	"publika-auction/internal/app/hub"
	"publika-auction/internal/app/mng"
	"publika-auction/internal/app/server"
	"publika-auction/internal/app/tg"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg := configuration.New()

	mg, err := mng.New()
	if err != nil {
		panic(err)
	}

	// cls := csv_service.New()
	// clients, err := cls.Read()

	clRepo := clients_repo.New(mg)
	// clRepo.SetAll(clients)

	rds := redis.NewClient(&redis.Options{
		Addr:     cfg.REDIS_ADDR,
		Password: cfg.REDIS_PWD,
		DB:       cfg.REDIS_DB,
	})

	// orclApp.InsertMechs(ctx, repo.GetMechs())

	bds, bdsOut := bids2.New(mg)

	hb := hub.New(rds, clRepo, bds)

	tgbot, err := tg.New(tg.Config{
		cfg.TG_TOKEN,
		cfg.TG_ENDPOINT,
	}, hb, bdsOut)
	if err != nil {
		//panic(err)
	}
	fmt.Println(tgbot)
	// go tgbot.Start(ctx)

	srv := server.New(&cfg, bds, hb, mg, clRepo)
	go handleSignals(ctx, srv.Cancel)
	err = srv.Start()
	if err != nil {
		log.Err(err).Msg("HTTP server error")
	}
	log.Info().Msg("have a nice day!")
}

func handleSignals(ctx context.Context, f func(ctx2 context.Context) error) {
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)
	select {
	case <-sigint:
		f(ctx)
		ctx.Done()
		return
	}
}
