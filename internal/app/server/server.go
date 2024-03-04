package server

import (
	"context"
	_ "expvar"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"net/http"
	"publika-auction/cmd/configuration"
	"publika-auction/internal/app/bids"
	"publika-auction/internal/app/handlers"
	"publika-auction/internal/app/hub"
	"publika-auction/internal/app/mng"
)

type Server struct {
	srv *http.Server
}

type PushClient interface {
	Send(e []interface{}) error
	Shutdown() error
}

func New(cfg *configuration.Config, bs *bids.BidsStorage, hb *hub.Hub, ms *mng.MngSrv) *Server {
	r := chi.NewRouter()
	// r.Get("/", handlers.WS(cfg, hb))
	// r.Get("/events", handlers.GetEvents(repo))
	r.Get("/main", handlers.Main(cfg, bs))
	r.Get("/lot{num:([0-9]+)}", handlers.Lot(cfg, bs))
	r.Get("/chats", handlers.Chats(cfg, hb))
	r.Get("/chat/{id:([0-9]+)}", handlers.ChatBids(cfg, hb, ms))
	r.Get("/phone/{phone:(\\+[0-9]+)}", handlers.PhoneBids(cfg, hb, ms))

	r.NotFound(handlers.NotFound)

	srv := &http.Server{
		Addr:    cfg.ADDR,
		Handler: r,
	}

	return &Server{
		srv: srv,
	}
}

func (s *Server) Cancel(ctx context.Context) error {
	log.Info().Msg("Stoping http server")
	return s.srv.Shutdown(ctx)
}

func (s *Server) Start() error {
	log.Info().Str("addr", s.srv.Addr).Msg("Starting http server")
	err := s.srv.ListenAndServe()
	return err
}
