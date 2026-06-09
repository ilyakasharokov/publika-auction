package bidsvc

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"publika-auction/internal/domain"
	"publika-auction/internal/lock"
	"publika-auction/internal/metrics"
	"publika-auction/internal/repo"
	"publika-auction/internal/repo/cache"
	"publika-auction/internal/tgqueue"
)

var (
	ErrBidTooLow  = errors.New("bid too low")
	ErrLotBusy    = errors.New("lot locked by concurrent bid")
	ErrLotSold    = errors.New("lot already sold")
	ErrBlocked    = errors.New("client is blocked")
	ErrNoAuction  = errors.New("no active auction")
)

type PlaceBidRequest struct {
	AuctionID   string
	AuctionSlug string
	LotID       string
	LotNum      int
	ClientID    string
	Phone       string
	TgUserID    int64
	Amount      int
}

type ErrBidTooLowDetail struct {
	Current int
}

func (e ErrBidTooLowDetail) Error() string {
	return fmt.Sprintf("bid too low: current max is %d", e.Current)
}

type Notifier = tgqueue.NotifierIface

type EventBroadcaster interface {
	Publish(e Event)
}

type Event struct {
	Type      string
	AuctionID string
	LotID     string
	LotNum    int
	BidID     string
	Amount    int
	Phone     string
	BidCount  int
}

type Service struct {
	bidRepo  repo.BidRepo
	lotRepo  repo.LotRepo
	bidCache *cache.BidCache
	lock     lock.Locker
	notifier Notifier
	events   EventBroadcaster
}

func (s *Service) SetNotifier(n Notifier) {
	s.notifier = n
}

func New(
	bidRepo repo.BidRepo,
	lotRepo repo.LotRepo,
	bidCache *cache.BidCache,
	lock lock.Locker,
	notifier Notifier,
	events EventBroadcaster,
) *Service {
	return &Service{
		bidRepo:  bidRepo,
		lotRepo:  lotRepo,
		bidCache: bidCache,
		lock:     lock,
		notifier: notifier,
		events:   events,
	}
}

func (s *Service) PlaceBid(ctx context.Context, req PlaceBidRequest) (*domain.Bid, error) {
	state, ok := s.bidCache.Get(req.AuctionID, req.LotID)
	if !ok {
		metrics.BidsRejectedTotal.WithLabelValues("lot_not_found").Inc()
		return nil, ErrNoAuction
	}
	if state.MaxAmount > 0 && state.MaxClientID == req.ClientID && req.Amount <= state.MaxAmount {
		metrics.BidsRejectedTotal.WithLabelValues("own_max").Inc()
		return nil, ErrBidTooLowDetail{Current: state.MaxAmount}
	}
	if req.Amount < state.MaxAmount+state.BidStep {
		metrics.BidsRejectedTotal.WithLabelValues("too_low").Inc()
		return nil, ErrBidTooLowDetail{Current: state.MaxAmount}
	}

	lockKey := fmt.Sprintf("lock:%s:%s", req.AuctionID, req.LotID)
	t := time.Now()
	token, acquired, err := s.lock.Acquire(ctx, lockKey, 500*time.Millisecond)
	metrics.LockAcquireDuration.Observe(time.Since(t).Seconds())
	if err != nil || !acquired {
		metrics.LockContentionTotal.Inc()
		return nil, ErrLotBusy
	}
	defer s.lock.Release(ctx, lockKey, token)

	// re-check under lock
	state, _ = s.bidCache.Get(req.AuctionID, req.LotID)
	if req.Amount < state.MaxAmount+state.BidStep {
		metrics.BidsRejectedTotal.WithLabelValues("too_low_recheck").Inc()
		return nil, ErrBidTooLowDetail{Current: state.MaxAmount}
	}

	bid := &domain.Bid{
		ID:        uuid.New().String(),
		AuctionID: req.AuctionID,
		LotID:     req.LotID,
		LotNum:    req.LotNum,
		ClientID:  req.ClientID,
		Phone:     req.Phone,
		TgUserID:  req.TgUserID,
		Amount:    req.Amount,
		PlacedAt:  time.Now(),
	}

	prevTgID := state.MaxTgID
	prevBidID := state.MaxBidID

	state.MaxAmount = req.Amount
	state.MaxBidID = bid.ID
	state.MaxClientID = req.ClientID
	state.MaxTgID = req.TgUserID
	state.BidCount++
	s.bidCache.Set(req.AuctionID, req.LotID, state)

	go func() {
		if err := s.bidRepo.Insert(context.Background(), bid); err != nil {
			log.Err(err).Str("bid_id", bid.ID).Msg("async bid insert failed")
		}
	}()

	if prevTgID != 0 && prevTgID != req.TgUserID && prevBidID != "" {
		go s.notifier.Send(prevTgID, "Вашу ставку на лот #"+strconv.Itoa(req.LotNum)+" перебили.\nНовая ставка: "+strconv.Itoa(req.Amount)+"₽")
	}

	metrics.BidsPlacedTotal.WithLabelValues(req.AuctionSlug, strconv.Itoa(req.LotNum)).Inc()
	metrics.BidAmountRub.WithLabelValues(req.AuctionSlug).Observe(float64(req.Amount))

	if s.events != nil {
		s.events.Publish(Event{
			Type:      "bid_placed",
			AuctionID: req.AuctionID,
			LotID:     req.LotID,
			LotNum:    req.LotNum,
			BidID:     bid.ID,
			Amount:    req.Amount,
			Phone:     req.Phone,
			BidCount:  state.BidCount,
		})
	}

	return bid, nil
}

func (s *Service) SellLot(ctx context.Context, lotID, bidID string) error {
	bid, err := s.bidRepo.GetByID(ctx, bidID)
	if err != nil {
		return err
	}
	lot, err := s.lotRepo.GetByID(ctx, lotID)
	if err != nil {
		return err
	}
	if err := s.lotRepo.MarkSold(ctx, lotID, bidID, bid.Amount); err != nil {
		return err
	}

	state, ok := s.bidCache.Get(bid.AuctionID, lotID)
	if ok {
		state.MaxBidID = bidID
		s.bidCache.Set(bid.AuctionID, lotID, state)
	}

	go s.notifier.Send(bid.TgUserID, "Поздравляем! Лот #"+strconv.Itoa(lot.Num)+" продан вам за "+strconv.Itoa(bid.Amount)+"₽")

	metrics.LotsSoldTotal.WithLabelValues(state.AuctionSlug).Inc()
	metrics.LotsActive.Dec()

	if s.events != nil {
		s.events.Publish(Event{
			Type:   "lot_sold",
			LotID:  lotID,
			LotNum: lot.Num,
			BidID:  bidID,
			Amount: bid.Amount,
		})
	}
	return nil
}

func (s *Service) CancelBid(ctx context.Context, bidID string) error {
	bid, err := s.bidRepo.GetByID(ctx, bidID)
	if err != nil {
		return err
	}
	if err := s.bidRepo.MarkCancelled(ctx, bidID); err != nil {
		return err
	}
	state, ok := s.bidCache.Get(bid.AuctionID, bid.LotID)
	if ok && state.MaxBidID == bidID {
		state.MaxAmount = 0
		state.MaxBidID = ""
		state.MaxClientID = ""
		state.MaxTgID = 0
		if state.BidCount > 0 {
			state.BidCount--
		}
		s.bidCache.Set(bid.AuctionID, bid.LotID, state)
	}
	return nil
}

func (s *Service) GetLotState(auctionID, lotID string) (cache.LotState, bool) {
	return s.bidCache.Get(auctionID, lotID)
}

func (s *Service) GetLotBids(ctx context.Context, lotID string) ([]*domain.Bid, error) {
	return s.bidRepo.ListByLot(ctx, lotID)
}

func (s *Service) GetBidsByPhone(ctx context.Context, phone string) ([]*domain.Bid, error) {
	return s.bidRepo.ListByPhone(ctx, phone)
}

func (s *Service) HydrateLot(auctionID, auctionSlug string, lot *domain.Lot, bidStep int) {
	state := cache.LotState{
		LotID:       lot.ID,
		LotNum:      lot.Num,
		AuctionSlug: auctionSlug,
		StartPrice:  lot.StartPrice,
		BidStep:     bidStep,
		MaxAmount:   lot.StartPrice,
	}
	if lot.Status == domain.LotSold {
		state.MaxAmount = lot.SoldFor
	}
	existing, ok := s.bidCache.Get(auctionID, lot.ID)
	if ok {
		existing.LotID = lot.ID
		existing.LotNum = lot.Num
		existing.AuctionSlug = auctionSlug
		existing.StartPrice = lot.StartPrice
		existing.BidStep = bidStep
		s.bidCache.Set(auctionID, lot.ID, existing)
		return
	}
	s.bidCache.Set(auctionID, lot.ID, state)
}
