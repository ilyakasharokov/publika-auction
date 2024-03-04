package bids

import (
	"errors"
	"github.com/rs/zerolog/log"
	clients_repo "publika-auction/internal/app/clients-repo"
	"publika-auction/internal/app/mng"
	"publika-auction/internal/app/models"
	"strconv"
	"sync"
	"time"
)

type BidsStorage struct {
	mx    sync.RWMutex
	Items map[int]models.Item
	out   chan Msg
	mng   *mng.MngSrv
}

type Msg struct {
	ChatId  int64
	Message string
}

var step = 1000

func (bs *BidsStorage) GetItem(lot int) (models.Item, error) {
	bs.mx.RLock()
	defer bs.mx.RUnlock()
	item, ok := bs.Items[lot]
	if ok {
		item.ViewCount++
		bs.Items[lot] = item
		return item, nil
	}
	return item, errors.New("not found")
}

func (bs *BidsStorage) AddBet(lot int, summ int, clientsPhone string, client *clients_repo.Client) (int, error) {
	bs.mx.RLock()
	defer bs.mx.RUnlock()
	item, ok := bs.Items[lot]
	if ok {
		if summ >= item.MaxConfirmed+step {
			newBid := models.Bid{
				Id:           item.Id*10000000 + len(item.Bids),
				ItemId:       item.Id,
				ClientsPhone: clientsPhone,
				Date:         time.Now(),
				Summ:         summ,
				Confirmed:    false,
				Client:       client,
			}
			if item.MaxBid.Client != nil && item.MaxBid.Client.TgUserId != client.TgUserId {
				go bs.SendToOut(item.MaxBid.Client.TgUserId, "Вашу ставку перебили на лот #"+strconv.Itoa(item.Id)+" перебили. \nНовая ставка "+strconv.Itoa(newBid.Summ)+"р")
			}
			item.MaxBid = newBid
			item.Bids = append(item.Bids, newBid)
			item.MaxConfirmed = summ
			bs.Items[lot] = item
			go bs.mng.InsertBid(newBid)
			return 0, nil
		} else {
			return item.MaxConfirmed, errors.New("less than current")
		}
	}
	return 0, errors.New("not found")
}

func (bs *BidsStorage) SendToOut(id int64, msg string) {
	bs.out <- Msg{id, msg}
	log.Info().Msg("t")
}

func (bs *BidsStorage) ConfirmBet(lot int, id int) error {
	bs.mx.RLock()
	defer bs.mx.RUnlock()
	item, ok := bs.Items[lot]
	if ok {
		bid := models.Bid{}
		for _, b := range item.Bids {
			if b.Id == id {
				bid = b
				break
			}
		}
		if bid.Id == 0 {
			return errors.New("bid not found")
		}
		bid.Confirmed = true
		item.MaxConfirmed = bid.Summ
		item.Bids = []models.Bid{bid}
		bs.Items[lot] = item
	}
	return nil
}

func (bs *BidsStorage) GetItems() Items {
	bs.mx.RLock()
	defer bs.mx.RUnlock()
	ar := make([]models.Item, len(bs.Items)+1)
	for i, item := range bs.Items {
		ar[i] = item
	}
	return ar[1:]
}

type Items []models.Item

func (it Items) ById(id int) *models.Item {
	for _, i := range it {
		if i.Id == id {
			return &i
		}
	}
	return nil
}

/*func (bs *BidsStorage) GetBidsByPhone(phone string) []models.Bid {
	bs.mx.RLock()
	defer bs.mx.RUnlock()
	ar := make([]models.Bid, 0)
	for _, item := range bs.Items {
		for b := range item.Bids {

		}
	}
	return ar[1:]
}*/

func New(mg *mng.MngSrv) (*BidsStorage, chan Msg) {
	bs := &BidsStorage{
		Items: make(map[int]models.Item),
		mng:   mg,
	}
	for i := 1; i < 19; i++ {
		bs.Items[i] = models.Item{
			Id:           i,
			Bids:         make([]models.Bid, 0),
			MaxConfirmed: 1000,
			Photo:        "https://static.insales-cdn.com/images/products/1/2639/787352143/large_PG_4_copy.png",
			Description:  "Здесь какое то описание лота, возможно длинное или нет кто его знает, может вообще не будет. десь какое то описание лота, возможно длинное или нет кто его знает, может вообще не будет",
		}
	}
	out := make(chan Msg, 0)
	bs.out = out

	bids := mg.GetBids()
	if bids != nil {
		for _, bid := range bids {
			item, ok := bs.Items[bid.ItemId]
			if ok {
				item.Bids = append(item.Bids, bid)
				item.MaxConfirmed = bid.Summ
			}
			bs.Items[bid.ItemId] = item
		}
	}
	/*	bs.Items[2] = Item{
			Id:           2,
			Bids:         make([]Bid, 0),
			MaxConfirmed: 1000,
			Photo:        "https://static.insales-cdn.com/images/products/1/2639/787352143/large_PG_4_copy.png",
		}
		bs.Items[3] = Item{
			Id:           3,
			Bids:         make([]Bid, 0),
			MaxConfirmed: 1000,
			Photo:        "https://static.insales-cdn.com/images/products/1/2639/787352143/large_PG_4_copy.png",
		}*/
	return bs, out
}
