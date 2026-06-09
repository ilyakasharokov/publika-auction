package handlers

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"publika-auction/internal/domain"
	"publika-auction/internal/hub"
	"publika-auction/internal/repo/cache"
	auctionsvc "publika-auction/internal/service/auction"
	bidsvc "publika-auction/internal/service/bid"
	lotsvc "publika-auction/internal/service/lot"
)

type AuctionHandler struct {
	auctionSvc *auctionsvc.Service
	lotSvc     *lotsvc.Service
	bidSvc     *bidsvc.Service
	bidCache   *cache.BidCache
	hub        *hub.Hub
}

func NewAuctionHandler(as *auctionsvc.Service, ls *lotsvc.Service, bs *bidsvc.Service, bc *cache.BidCache, h *hub.Hub) *AuctionHandler {
	return &AuctionHandler{auctionSvc: as, lotSvc: ls, bidSvc: bs, bidCache: bc, hub: h}
}

type auctionListData struct {
	Auctions []*domain.Auction
}

func (h *AuctionHandler) List(w http.ResponseWriter, r *http.Request) {
	auctions, err := h.auctionSvc.List(r.Context())
	if err != nil {
		log.Err(err).Msg("list auctions")
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	render(w, r, "auction_list.html", auctionListData{Auctions: auctions})
}

func (h *AuctionHandler) NewForm(w http.ResponseWriter, r *http.Request) {
	render(w, r, "auction_form.html", map[string]interface{}{"Error": ""})
}

func (h *AuctionHandler) Create(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	bidStep := 2000
	if v := r.Form.Get("bid_step"); v != "" {
		n := 0
		for _, c := range v {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			}
		}
		if n > 0 {
			bidStep = n
		}
	}
	req := auctionsvc.CreateRequest{
		Slug:        r.Form.Get("slug"),
		Title:       r.Form.Get("title"),
		Description: r.Form.Get("description"),
		BidStep:     bidStep,
		StartAt:     time.Now(),
		EndAt:       time.Now().Add(24 * time.Hour),
	}
	if _, err := h.auctionSvc.Create(r.Context(), req); err != nil {
		render(w, r, "auction_form.html", map[string]interface{}{"Error": err.Error()})
		return
	}
	http.Redirect(w, r, "/admin/auctions", http.StatusFound)
}

type auctionDetailData struct {
	Auction    *domain.Auction
	Lots       []*domain.Lot
	LotStates  map[string]cache.LotState
}

func (h *AuctionHandler) Detail(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	a, err := h.auctionSvc.GetBySlug(r.Context(), slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	lots, _ := h.lotSvc.ListByAuction(r.Context(), a.ID)
	states := make(map[string]cache.LotState)
	for _, lot := range lots {
		if s, ok := h.bidCache.Get(a.ID, lot.ID); ok {
			states[lot.ID] = s
		}
	}
	render(w, r, "auction_detail.html", auctionDetailData{Auction: a, Lots: lots, LotStates: states})
}

func (h *AuctionHandler) Activate(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	a, err := h.auctionSvc.GetBySlug(r.Context(), slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := h.auctionSvc.Activate(r.Context(), a.ID); err != nil {
		log.Err(err).Str("slug", slug).Msg("activate auction")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.Status = domain.AuctionActive
	lots, _ := h.lotSvc.ListByAuction(r.Context(), a.ID)
	h.hub.SetActiveAuction(a, lots)
	http.Redirect(w, r, "/admin/auctions/"+slug, http.StatusFound)
}

func (h *AuctionHandler) End(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	a, err := h.auctionSvc.GetBySlug(r.Context(), slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := h.auctionSvc.End(r.Context(), a.ID); err != nil {
		log.Err(err).Str("slug", slug).Msg("end auction")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ended := *a
	ended.Status = domain.AuctionEnded
	h.hub.SetActiveAuction(&ended, nil)
	http.Redirect(w, r, "/admin/auctions/"+slug, http.StatusFound)
}

func (h *AuctionHandler) Broadcast(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	r.ParseForm()
	msg := r.Form.Get("message")
	if msg == "" {
		http.Redirect(w, r, "/admin/auctions/"+slug, http.StatusFound)
		return
	}
	// Send to all connected chats via hub.Out
	all := h.hub.GetAllChats()
	for _, ci := range all {
		if ci.Client != nil && ci.Client.TgUserID != 0 {
			h.hub.SendTo(ci.Client.TgUserID, msg)
		}
	}
	http.Redirect(w, r, "/admin/auctions/"+slug, http.StatusFound)
}

func (h *AuctionHandler) AddLotForm(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	a, err := h.auctionSvc.GetBySlug(r.Context(), slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	render(w, r, "lot_form.html", map[string]interface{}{"Auction": a, "Error": ""})
}

func (h *AuctionHandler) LotsRefresh(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	a, err := h.auctionSvc.GetBySlug(r.Context(), slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	lots, _ := h.lotSvc.ListByAuction(r.Context(), a.ID)
	states := make(map[string]cache.LotState)
	for _, lot := range lots {
		if s, ok := h.bidCache.Get(a.ID, lot.ID); ok {
			states[lot.ID] = s
		}
	}
	renderPartial(w, r, "lots_rows.html", map[string]interface{}{
		"Auction":   a,
		"Lots":      lots,
		"LotStates": states,
	})
}

func (h *AuctionHandler) AddLot(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	a, err := h.auctionSvc.GetBySlug(r.Context(), slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	r.ParseMultipartForm(32 << 20)
	num := 0
	lots, _ := h.lotSvc.ListByAuction(context.Background(), a.ID)
	for _, l := range lots {
		if l.Num > num {
			num = l.Num
		}
	}
	num++
	startPrice := 0
	for _, c := range r.Form.Get("start_price") {
		if c >= '0' && c <= '9' {
			startPrice = startPrice*10 + int(c-'0')
		}
	}

	photoURL := r.Form.Get("photo_url")
	if file, header, err := r.FormFile("photo_file"); err == nil {
		defer file.Close()
		ext := filepath.Ext(header.Filename)
		if ext == "" {
			ext = ".jpg"
		}
		if err := os.MkdirAll("uploads", 0755); err == nil {
			name := uuid.New().String() + ext
			if dst, err := os.Create(filepath.Join("uploads", name)); err == nil {
				defer dst.Close()
				io.Copy(dst, file)
				photoURL = "/uploads/" + name
			}
		}
	}

	req := lotsvc.CreateRequest{
		AuctionID:   a.ID,
		Num:         num,
		Title:       r.Form.Get("title"),
		Description: r.Form.Get("description"),
		PhotoURL:    photoURL,
		StartPrice:  startPrice,
	}
	lot, err := h.lotSvc.Create(r.Context(), req)
	if err != nil {
		render(w, r, "lot_form.html", map[string]interface{}{"Auction": a, "Error": err.Error()})
		return
	}
	if a.Status == domain.AuctionActive {
		h.bidSvc.HydrateLot(a.ID, a.Slug, lot, a.BidStep)
		lots, _ := h.lotSvc.ListByAuction(context.Background(), a.ID)
		h.hub.SetActiveAuction(a, lots)
	}
	http.Redirect(w, r, "/admin/auctions/"+slug, http.StatusFound)
}
