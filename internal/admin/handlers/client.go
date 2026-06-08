package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"publika-auction/internal/domain"
	bidsvc "publika-auction/internal/service/bid"
	clientsvc "publika-auction/internal/service/client"
)

type ClientHandler struct {
	clientSvc *clientsvc.Service
	bidSvc    *bidsvc.Service
}

func NewClientHandler(cs *clientsvc.Service, bs *bidsvc.Service) *ClientHandler {
	return &ClientHandler{clientSvc: cs, bidSvc: bs}
}

type clientListData struct {
	Clients []*domain.Client
}

func (h *ClientHandler) List(w http.ResponseWriter, r *http.Request) {
	clients, err := h.clientSvc.ListAll(r.Context())
	if err != nil {
		log.Err(err).Msg("list clients")
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	render(w, r, "client_list.html", clientListData{Clients: clients})
}

type clientDetailData struct {
	Client   *domain.Client
	Bids     []*domain.Bid
	Messages []*domain.ChatMessage
	Sent     bool
}

func (h *ClientHandler) Detail(w http.ResponseWriter, r *http.Request) {
	phone := chi.URLParam(r, "phone")
	client, found := h.clientSvc.GetByPhone(r.Context(), phone)
	if !found {
		http.NotFound(w, r)
		return
	}
	bids, _ := h.bidSvc.GetBidsByPhone(r.Context(), phone)
	msgs, _ := h.clientSvc.GetMessages(r.Context(), client.TgUserID)

	sent := false
	if r.Method == http.MethodPost {
		r.ParseForm()
		msg := r.Form.Get("message")
		if msg != "" && client.TgUserID != 0 {
			h.clientSvc.SendMessage(r.Context(), client.TgUserID, msg)
			sent = true
			msgs, _ = h.clientSvc.GetMessages(r.Context(), client.TgUserID)
		}
	}

	render(w, r, "client_detail.html", clientDetailData{
		Client:   client,
		Bids:     bids,
		Messages: msgs,
		Sent:     sent,
	})
}

func (h *ClientHandler) Block(w http.ResponseWriter, r *http.Request) {
	phone := chi.URLParam(r, "phone")
	if err := h.clientSvc.Block(r.Context(), phone); err != nil {
		log.Err(err).Str("phone", phone).Msg("block client")
	}
	http.Redirect(w, r, "/admin/clients/"+phone, http.StatusFound)
}
