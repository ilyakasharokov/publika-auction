package handlers

import (
	_ "embed"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"html/template"
	"net/http"
	"publika-auction/cmd/configuration"
	"publika-auction/internal/app/bids"
	clients_repo "publika-auction/internal/app/clients-repo"
	"publika-auction/internal/app/hub"
	"publika-auction/internal/app/mng"
	"strconv"
	"time"
)

func Responder(w http.ResponseWriter, _ *http.Request, response interface{}, code int) {
	body, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(code)
	_, err = w.Write(body)
	if err != nil {
		log.Err(err).Msg("Body write error")
	}
	return
}

type Response struct {
	Error string `json:"error"`
}

type MainObj struct {
	Items bids.Items
	Sent  bool
	Start bool
	Now   time.Time
}

func Main(_ *configuration.Config, bs *bids.BidsStorage, hb *hub.Hub) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var indexTemplate, _ = template.ParseFiles("index.html")
		mo := MainObj{}
		mo.Now = time.Now()
		mo.Items = bs.GetAllItems()
		r.ParseForm()
		msg := r.Form.Get("message")
		if msg != "" {
			hb.SendToAll(msg)
			mo.Sent = true
		}
		start := r.Form.Get("start")
		if start == "start" {
			bs.Start = true
		}
		mo.Start = bs.Start
		err := indexTemplate.Execute(w, mo)
		if err != nil {
			log.Err(err).Msg("Execute error")
			return
		}
	}
}

func Lot(_ *configuration.Config, bs *bids.BidsStorage, clRepo *clients_repo.ClientsRepository) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var indexTemplate, _ = template.ParseFiles("lot.html")
		numStr := chi.URLParam(r, "num")
		num, err := strconv.Atoi(numStr)
		if err != nil {
			log.Err(err).Msg("lot can't parse num")
			return
		}
		r.ParseForm()
		phone := r.Form.Get("phone")
		if phone != "" {
			clRepo.Block(phone)
		}
		bidid := r.Form.Get("bidid")
		if bidid != "" {
			bidNum, err := strconv.Atoi(bidid)
			if err != nil {
				log.Err(err).Str("uri", r.RequestURI).Msg("bidNum strconv")
			} else {
				bs.SellItem(num, bidNum)
			}
		}
		lot, err := bs.GetItem(num)
		if err != nil {
			log.Err(err).Int("lot", num).Msg("lot not found")
			return
		}
		err = indexTemplate.Execute(w, lot)
		if err != nil {
			log.Err(err).Msg("Execute error")
			return
		}
	}
}

func Chats(_ *configuration.Config, hb *hub.Hub) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var indexTemplate, _ = template.ParseFiles("chats.html")
		chats := hb.GetChats()
		err := indexTemplate.Execute(w, chats)
		if err != nil {
			log.Err(err).Msg("Chats Execute error")
			return
		}
	}
}

func Registered(_ *configuration.Config, clRepo *clients_repo.ClientsRepository) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var indexTemplate, _ = template.ParseFiles("registered.html")
		chats := clRepo.GetAllWithId()
		err := indexTemplate.Execute(w, chats)
		if err != nil {
			log.Err(err).Msg("Registered Execute error")
			return
		}
	}
}

type Form struct {
	Message string
}

func ChatBids(_ *configuration.Config, hb *hub.Hub, mngSrv *mng.MngSrv) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var indexTemplate, _ = template.ParseFiles("chatbids.html")
		numStr := chi.URLParam(r, "id")
		num, err := strconv.Atoi(numStr)
		chatId := int64(num)
		if err != nil {
			log.Err(err).Msg("ChatBids can't parse num")
			return
		}

		chat := hb.GetChatById(chatId)
		if chat.Client != nil {
			bids := mngSrv.GetBidsByPhone(chat.Client.Phone)
			chat.Bids = bids
			r.ParseForm()
			msg := r.Form.Get("message")
			if msg != "" {
				hb.SendTo(chatId, chat.TGUsername, msg)
				chat.Sent = true
			}
		}
		err = indexTemplate.Execute(w, chat)
		if err != nil {
			log.Err(err).Msg("ChatBids Execute error")
			return
		}
	}
}

func PhoneBids(_ *configuration.Config, _ *hub.Hub, mngSrv *mng.MngSrv) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var indexTemplate, _ = template.ParseFiles("phonebids.html")
		phone := chi.URLParam(r, "phone")
		bids := mngSrv.GetBidsByPhone(phone)
		err := indexTemplate.Execute(w, bids)
		if err != nil {
			log.Err(err).Msg("PhoneBids Execute error")
			return
		}
	}
}

func NotFound(w http.ResponseWriter, r *http.Request) {
	log.Err(errors.New("method not found")).Str("url", r.RequestURI).Msg("")
	w.WriteHeader(http.StatusNotFound)
}
