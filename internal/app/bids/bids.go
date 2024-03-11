package bids

import (
	"errors"
	"github.com/rs/zerolog/log"
	clients_repo "publika-auction/internal/app/clients-repo"
	"publika-auction/internal/app/mng"
	"publika-auction/internal/app/models"
	"sort"
	"strconv"
	"sync"
	"time"
)

type BidsStorage struct {
	mx    sync.RWMutex
	Items map[int]models.Item
	out   chan Msg
	mng   *mng.MngSrv
	Start bool
}

type Msg struct {
	ChatId  int64
	Message string
	NewLot  int
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
	bs.mx.Lock()
	defer bs.mx.Unlock()
	item, ok := bs.Items[lot]
	if ok {
		if item.SoldFor > 0 {
			return 123123, errors.New("sold")
		}
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
				go bs.SendToOut(item.MaxBid.Client.TgUserId, "Вашу ставку на лот #"+strconv.Itoa(item.Id)+" перебили. \nНовая ставка "+strconv.Itoa(newBid.Summ)+"р", item.Id)
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

func (bs *BidsStorage) SellItem(lot int, bidId int) {
	bs.mx.Lock()
	defer bs.mx.Unlock()
	item, ok := bs.Items[lot]
	if ok {
		for _, b := range item.Bids {
			if b.Id == bidId {
				if b.Client != nil {
					item.SoldFor = b.Summ
					bs.Items[lot] = item
					go bs.SendToOut(b.Client.TgUserId, "Поздравляем, лот #"+strconv.Itoa(item.Id)+" продан вам за "+strconv.Itoa(b.Summ)+"р", 0)
					log.Info().Str("phone", b.ClientsPhone).Str("tg", b.Client.TgUsername).Int("lot", lot).Int("summ", b.Summ).Msg("superalarm sold")
					return
				}
			}
		}

	}
}

func (bs *BidsStorage) SendToOut(id int64, msg string, itemId int) {
	bs.out <- Msg{id, msg, itemId}
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

func (bs *BidsStorage) CancelBet(lot int, id int) error {
	bs.mx.Lock()
	defer bs.mx.Unlock()
	item, ok := bs.Items[lot]
	if ok {
		bid := models.Bid{}
		bidIndex := 0
		for i, b := range item.Bids {
			if b.Id == id {
				bidIndex = i
				bid = b
				break
			}
		}
		if bid.Id == 0 {
			return errors.New("bid not found")
		}
		prev := item.Bids[:bidIndex]
		after := item.Bids[bidIndex+1:]
		item.Bids = append(prev, after...)
		if bid.Id == item.MaxBid.Id {
			prevBid := item.Bids[bidIndex-1]
			item.MaxConfirmed = prevBid.Summ
			item.MaxBid = prevBid
		}
		bs.Items[lot] = item
		go bs.mng.DeleteBid(bid)
	}
	return nil
}

func (bs *BidsStorage) GetItems() Items {
	bs.mx.RLock()
	defer bs.mx.RUnlock()
	ar := make([]models.Item, 0)
	for _, item := range bs.Items {
		if item.SoldFor == 0 {
			ar = append(ar, item)
		}
	}
	sort.SliceStable(ar, func(i, j int) bool {
		return ar[i].Id < ar[j].Id
	})
	return ar
}

func (bs *BidsStorage) GetAllItems() Items {
	bs.mx.RLock()
	defer bs.mx.RUnlock()
	ar := make([]models.Item, 0)
	for _, item := range bs.Items {
		ar = append(ar, item)
	}
	sort.SliceStable(ar, func(i, j int) bool {
		return ar[i].Id < ar[j].Id
	})
	return ar
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
		Start: false,
	}
	for i := 1; i < 19; i++ {
		bs.Items[i] = models.Item{
			Id:           i,
			Bids:         make([]models.Bid, 0),
			MaxConfirmed: 15000,
			Photo:        "https://dimanova.space/images/num/jpeg-optimizer_" + strconv.Itoa(i) + ".jpg",
			Description:  "можно длинное или нет кто его знает, может вообще не будет",
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
				if bid.Summ > item.MaxConfirmed {
					item.MaxConfirmed = bid.Summ
					item.MaxBid = bid
				}
			}
			bs.Items[bid.ItemId] = item
		}
	}
	return bs, out
}
