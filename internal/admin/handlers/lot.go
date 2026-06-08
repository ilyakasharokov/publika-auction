package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"publika-auction/internal/domain"
	"publika-auction/internal/repo/cache"
	auctionsvc "publika-auction/internal/service/auction"
	bidsvc "publika-auction/internal/service/bid"
	lotsvc "publika-auction/internal/service/lot"
)

type LotHandler struct {
	auctionSvc *auctionsvc.Service
	lotSvc     *lotsvc.Service
	bidSvc     *bidsvc.Service
	bidCache   *cache.BidCache
}

func NewLotHandler(as *auctionsvc.Service, ls *lotsvc.Service, bs *bidsvc.Service, bc *cache.BidCache) *LotHandler {
	return &LotHandler{auctionSvc: as, lotSvc: ls, bidSvc: bs, bidCache: bc}
}

type lotDetailData struct {
	Auction  *domain.Auction
	Lot      *domain.Lot
	Bids     []*domain.Bid
	State    cache.LotState
}

func (h *LotHandler) Detail(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	lotNumStr := chi.URLParam(r, "num")
	lotNum, _ := strconv.Atoi(lotNumStr)

	a, err := h.auctionSvc.GetBySlug(r.Context(), slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	lots, _ := h.lotSvc.ListByAuction(r.Context(), a.ID)
	var lot *domain.Lot
	for _, l := range lots {
		if l.Num == lotNum {
			lot = l
			break
		}
	}
	if lot == nil {
		http.NotFound(w, r)
		return
	}

	bids, _ := h.bidSvc.GetLotBids(r.Context(), lot.ID)
	state, _ := h.bidCache.Get(a.ID, lot.ID)

	render(w, r, "lot_detail.html", lotDetailData{
		Auction: a,
		Lot:     lot,
		Bids:    bids,
		State:   state,
	})
}

func (h *LotHandler) BidsFeed(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	lotNumStr := chi.URLParam(r, "num")
	lotNum, _ := strconv.Atoi(lotNumStr)

	a, err := h.auctionSvc.GetBySlug(r.Context(), slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	lots, _ := h.lotSvc.ListByAuction(r.Context(), a.ID)
	var lot *domain.Lot
	for _, l := range lots {
		if l.Num == lotNum {
			lot = l
			break
		}
	}
	if lot == nil {
		http.NotFound(w, r)
		return
	}

	bids, _ := h.bidSvc.GetLotBids(r.Context(), lot.ID)
	state, _ := h.bidCache.Get(a.ID, lot.ID)

	renderPartial(w, r, "bids_table.html", map[string]interface{}{
		"Auction": a,
		"Bids":    bids,
		"State":   state,
		"Lot":     lot,
	})
}

func (h *LotHandler) Sell(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	lotNumStr := chi.URLParam(r, "num")
	lotNum, _ := strconv.Atoi(lotNumStr)
	r.ParseForm()
	bidID := r.Form.Get("bid_id")
	if bidID == "" {
		http.Error(w, "bid_id required", http.StatusBadRequest)
		return
	}

	a, err := h.auctionSvc.GetBySlug(r.Context(), slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	lots, _ := h.lotSvc.ListByAuction(r.Context(), a.ID)
	var lot *domain.Lot
	for _, l := range lots {
		if l.Num == lotNum {
			lot = l
			break
		}
	}
	if lot == nil {
		http.NotFound(w, r)
		return
	}

	if err := h.bidSvc.SellLot(r.Context(), lot.ID, bidID); err != nil {
		log.Err(err).Str("lot_id", lot.ID).Str("bid_id", bidID).Msg("sell lot")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/auctions/"+slug+"/lots/"+lotNumStr, http.StatusFound)
}

func (h *LotHandler) CancelBid(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	lotNumStr := chi.URLParam(r, "num")
	bidID := chi.URLParam(r, "bid_id")

	if err := h.bidSvc.CancelBid(r.Context(), bidID); err != nil {
		log.Err(err).Str("bid_id", bidID).Msg("cancel bid")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/auctions/"+slug+"/lots/"+lotNumStr, http.StatusFound)
}

func (h *LotHandler) Pull(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	lotNumStr := chi.URLParam(r, "num")
	lotNum, _ := strconv.Atoi(lotNumStr)

	a, err := h.auctionSvc.GetBySlug(r.Context(), slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	lots, _ := h.lotSvc.ListByAuction(r.Context(), a.ID)
	for _, l := range lots {
		if l.Num == lotNum {
			h.lotSvc.Pull(r.Context(), l.ID)
			break
		}
	}
	http.Redirect(w, r, "/admin/auctions/"+slug, http.StatusFound)
}
