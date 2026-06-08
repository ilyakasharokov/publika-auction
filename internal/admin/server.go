package admin

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"publika-auction/internal/admin/handlers"
	adminmw "publika-auction/internal/admin/middleware"
	"publika-auction/internal/hub"
	"publika-auction/internal/repo/cache"
	auctionsvc "publika-auction/internal/service/auction"
	bidsvc "publika-auction/internal/service/bid"
	clientsvc "publika-auction/internal/service/client"
	lotsvc "publika-auction/internal/service/lot"
	"publika-auction/internal/tg"
)

type Config struct {
	Addr          string
	AdminUser     string
	AdminPassword string
	SessionSecret string
}

func NewServer(
	cfg Config,
	auctionSvc *auctionsvc.Service,
	lotSvc *lotsvc.Service,
	bidSvc *bidsvc.Service,
	clientSvc *clientsvc.Service,
	bidCache *cache.BidCache,
	sseHub *handlers.SSEHub,
	h *hub.Hub,
	botManager *tg.Manager,
) *http.Server {
	r := chi.NewRouter()
	r.Use(chimw.Recoverer)
	r.Use(adminmw.PrometheusMiddleware)

	authH := handlers.NewAuthHandler(cfg.AdminUser, cfg.AdminPassword, cfg.SessionSecret)
	auctionH := handlers.NewAuctionHandler(auctionSvc, lotSvc, bidSvc, bidCache, h)
	lotH := handlers.NewLotHandler(auctionSvc, lotSvc, bidSvc, bidCache)
	clientH := handlers.NewClientHandler(clientSvc, bidSvc)
	settingsH := handlers.NewSettingsHandler(botManager)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	r.Handle("/metrics", promhttp.Handler())

	r.Get("/admin/login", authH.LoginPage)
	r.Post("/admin/login", authH.LoginSubmit)
	r.Post("/admin/logout", authH.Logout)
	r.Get("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/auctions", http.StatusFound)
	})

	r.Group(func(r chi.Router) {
		r.Use(adminmw.RequireSession(cfg.SessionSecret))

		r.Get("/admin/events", sseHub.ServeHTTP)

		// Auctions
		r.Get("/admin/auctions", auctionH.List)
		r.Get("/admin/auctions/new", auctionH.NewForm)
		r.Post("/admin/auctions", auctionH.Create)
		r.Get("/admin/auctions/{slug}", auctionH.Detail)
		r.Post("/admin/auctions/{slug}/activate", auctionH.Activate)
		r.Post("/admin/auctions/{slug}/end", auctionH.End)
		r.Post("/admin/auctions/{slug}/broadcast", auctionH.Broadcast)
		r.Get("/admin/auctions/{slug}/lots/new", auctionH.AddLotForm)
		r.Post("/admin/auctions/{slug}/lots", auctionH.AddLot)

		// htmx polling refresh for lot table
		r.Get("/admin/auctions/{slug}/lots-refresh", func(w http.ResponseWriter, r *http.Request) {
			auctionH.LotsRefresh(w, r)
		})

		// Lots
		r.Get("/admin/auctions/{slug}/lots/{num}", lotH.Detail)
		r.Get("/admin/auctions/{slug}/lots/{num}/bids", lotH.BidsFeed)
		r.Post("/admin/auctions/{slug}/lots/{num}/sell", lotH.Sell)
		r.Post("/admin/auctions/{slug}/lots/{num}/bids/{bid_id}/cancel", lotH.CancelBid)
		r.Post("/admin/auctions/{slug}/lots/{num}/pull", lotH.Pull)

		// Clients
		r.Get("/admin/clients", clientH.List)
		r.Get("/admin/clients/{phone}", clientH.Detail)
		r.Post("/admin/clients/{phone}", clientH.Detail)
		r.Post("/admin/clients/{phone}/block", clientH.Block)

		// Settings / Bot
		r.Get("/admin/settings", settingsH.Page)
		r.Post("/admin/settings/connect", settingsH.Connect)
		r.Post("/admin/settings/disconnect", settingsH.Disconnect)
	})

	return &http.Server{Addr: cfg.Addr, Handler: r}
}
