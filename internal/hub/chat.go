package hub

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"

	"publika-auction/internal/domain"
	bidsvc "publika-auction/internal/service/bid"
	clientsvc "publika-auction/internal/service/client"
)

type Chat struct {
	ID         int64
	TGUserName string
	client     *domain.Client

	in  chan tgbotapi.Update
	out chan tgbotapi.Chattable

	hub        *Hub
	bidSvc     *bidsvc.Service
	clientSvc  *clientsvc.Service
	currentLot int
}

func (c *Chat) SendTo(update tgbotapi.Update) {
	c.in <- update
}

func (c *Chat) Run(onReturn func()) {
	defer onReturn()

	sharePhoneBtn := tgbotapi.NewKeyboardButtonContact("SHARE PHONE NUMBER")

	// Check if already known by TG ID
	if cl, ok := c.clientSvc.GetByTgID(context.Background(), c.ID); ok {
		c.client = cl
		select {
		case <-c.in:
		default:
		}
		goto authSuccess
	}

authLoop:
	for {
		select {
		case inUpd, ok := <-c.in:
			if !ok {
				return
			}
			if inUpd.CallbackQuery != nil || (inUpd.Message != nil && inUpd.Message.Text == "/start") {
				msg := tgbotapi.NewMessage(c.ID, "Hello! To participate in the auction we need your phone number.\n⬇️ Tap \"Share Phone Number\" below ⬇️")
				msg.ReplyMarkup = &tgbotapi.ReplyKeyboardMarkup{
					Keyboard:        [][]tgbotapi.KeyboardButton{{sharePhoneBtn}},
					ResizeKeyboard:  true,
					OneTimeKeyboard: false,
				}
				c.out <- msg
				continue
			}
			if inUpd.Message != nil && inUpd.Message.Contact != nil {
				phone := handlePhone(inUpd.Message.Contact.PhoneNumber)
				cl, found := c.clientSvc.GetByPhone(context.Background(), phone)
				if !found {
					cl = &domain.Client{
						Phone:       phone,
						Name:        inUpd.Message.Contact.FirstName,
						TgFirstName: inUpd.Message.Contact.FirstName,
						TgLastName:  inUpd.Message.Contact.LastName,
						TgUsername:  inUpd.Message.Chat.UserName,
						TgUserID:    inUpd.Message.Chat.ID,
						CreatedAt:   time.Now(),
					}
				} else {
					cl.TgFirstName = inUpd.Message.Contact.FirstName
					cl.TgLastName = inUpd.Message.Contact.LastName
					cl.TgUsername = inUpd.Message.Chat.UserName
					cl.TgUserID = inUpd.Message.Chat.ID
				}
				c.client = cl
				c.clientSvc.RegisterOrUpdate(context.Background(), c.client)
				break authLoop
			}
			msg := tgbotapi.NewMessage(c.ID, "⬇️ Tap \"Share Phone Number\" ⬇️")
			msg.ReplyMarkup = &tgbotapi.ReplyKeyboardMarkup{
				Keyboard:        [][]tgbotapi.KeyboardButton{{sharePhoneBtn}},
				ResizeKeyboard:  true,
				OneTimeKeyboard: false,
			}
			c.out <- msg
		}
	}

authSuccess:
	greeting := tgbotapi.NewMessage(c.ID, "Welcome, "+c.client.TgFirstName+"!")
	greeting.ReplyMarkup = &tgbotapi.ReplyKeyboardRemove{RemoveKeyboard: true}
	c.out <- greeting

	if !c.hub.IsStarted() {
		c.out <- tgbotapi.NewMessage(c.ID, "The auction will start soon. Stay tuned!")
	waitStart:
		for {
			select {
			case inUpd, ok := <-c.in:
				if !ok {
					return
				}
				if inUpd.Message != nil && inUpd.Message.Text != "" {
					c.clientSvc.RecordMessage(context.Background(), c.ID, c.TGUserName, inUpd.Message.Text)
				}
				if c.hub.IsStarted() {
					break waitStart
				}
				c.out <- tgbotapi.NewMessage(c.ID, "The auction will start soon. Stay tuned!")
			default:
				if !c.hub.IsStarted() {
					time.Sleep(3 * time.Second)
				} else {
					break waitStart
				}
			}
		}
	}

	for {
		select {
		case inUpd, ok := <-c.in:
			if !ok {
				log.Err(errors.New("channel closed")).Int64("id", c.ID).Msg("chat channel closed")
				return
			}
			if cl, found := c.clientSvc.GetByTgID(context.Background(), c.ID); found {
				c.client = cl
			}
			if inUpd.Message != nil {
				inMsg := inUpd.Message
				if inMsg.Text == "/start" {
					c.currentLot = 0
					c.sendLotsKeyboard()
					continue
				}
				sum, _ := strconv.Atoi(inMsg.Text)
				if c.currentLot != 0 && sum > 0 {
					if c.client.IsBlocked {
						c.out <- tgbotapi.NewMessage(c.ID, "You are blocked. Bids are not accepted.")
						continue
					}
					c.addBet(sum)
					continue
				}
				if inMsg.Text != "" {
					c.clientSvc.RecordMessage(context.Background(), c.ID, c.TGUserName, inMsg.Text)
					log.Info().Str("tg", c.TGUserName).Str("msg", inMsg.Text).Msg("free message")
				}
			} else if inUpd.CallbackQuery != nil {
				cq := inUpd.CallbackQuery
				c.out <- tgbotapi.NewDeleteMessage(cq.Message.Chat.ID, cq.Message.MessageID)
				cmnd := cq.Data
				sum, _ := strconv.Atoi(cmnd)
				switch {
				case strings.HasPrefix(cmnd, "lot"):
					lot, _ := strconv.Atoi(strings.TrimPrefix(cmnd, "lot"))
					c.currentLot = lot
					c.sendLotKeyboard()
				case sum > 0:
					if c.client.IsBlocked {
						c.out <- tgbotapi.NewMessage(c.ID, "You are blocked. Bids are not accepted.")
						continue
					}
					c.addBet(sum)
				case cmnd == "back":
					c.currentLot = 0
					c.sendLotsKeyboard()
				}
			}
		}
	}
}

func (c *Chat) addBet(sum int) {
	auction := c.hub.GetActiveAuction()
	if auction == nil {
		c.out <- tgbotapi.NewMessage(c.ID, "The auction is not active.")
		return
	}
	lot := c.hub.GetLotByNum(c.currentLot)
	if lot == nil {
		c.out <- tgbotapi.NewMessage(c.ID, "Lot not found.")
		return
	}

	_, err := c.bidSvc.PlaceBid(context.Background(), bidsvc.PlaceBidRequest{
		AuctionID:   auction.ID,
		AuctionSlug: auction.Slug,
		LotID:       lot.ID,
		LotNum:      lot.Num,
		ClientID:    c.client.ID,
		Phone:       c.client.Phone,
		TgUserID:    c.ID,
		Amount:      sum,
	})
	if err != nil {
		var tooLow bidsvc.ErrBidTooLowDetail
		if errors.As(err, &tooLow) {
			c.out <- tgbotapi.NewMessage(c.ID, "Current max bid is "+strconv.Itoa(tooLow.Current)+"₽ (min step "+strconv.Itoa(auction.BidStep)+"₽)")
			return
		}
		if errors.Is(err, bidsvc.ErrLotSold) {
			c.out <- tgbotapi.NewMessage(c.ID, "This lot has already been sold.")
			return
		}
		c.out <- tgbotapi.NewMessage(c.ID, "Please try again.")
		return
	}
	msg := tgbotapi.NewMessage(c.ID, "Bid accepted: Lot #"+strconv.Itoa(c.currentLot)+" — "+strconv.Itoa(sum)+"₽")
	rows := tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("◀ Back", "back"))
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows)
	c.out <- msg
}

func (c *Chat) sendLotsKeyboard() {
	lots := c.hub.GetActiveLots()
	activeLots := make([]*domain.Lot, 0)
	for _, l := range lots {
		if l.Status == domain.LotActive {
			activeLots = append(activeLots, l)
		}
	}
	if len(activeLots) == 0 {
		c.out <- tgbotapi.NewMessage(c.ID, "No active lots at the moment.")
		return
	}
	msg := tgbotapi.NewMessage(c.ID, "Choose a lot:")
	rows := make([][]tgbotapi.InlineKeyboardButton, 0)
	row := make([]tgbotapi.InlineKeyboardButton, 0)
	for _, lot := range activeLots {
		if len(row) == 3 {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(row...))
			row = make([]tgbotapi.InlineKeyboardButton, 0)
		}
		row = append(row, tgbotapi.NewInlineKeyboardButtonData("Lot #"+strconv.Itoa(lot.Num), "lot"+strconv.Itoa(lot.Num)))
	}
	if len(row) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(row...))
	}
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	c.out <- msg
}

func (c *Chat) sendLotKeyboard() {
	auction := c.hub.GetActiveAuction()
	if auction == nil {
		return
	}
	lot := c.hub.GetLotByNum(c.currentLot)
	if lot == nil {
		return
	}
	state, _ := c.bidSvc.GetLotState(auction.ID, lot.ID)
	current := state.MaxAmount
	if current == 0 {
		current = lot.StartPrice
	}
	step := auction.BidStep
	s1 := strconv.Itoa(current + step)
	s2 := strconv.Itoa(current + step*3)
	s3 := strconv.Itoa(current + step*6)

	if lot.PhotoURL != "" {
		photo := tgbotapi.NewPhoto(c.ID, tgbotapi.FileURL(lot.PhotoURL))
		photo.Caption = lot.Title
		c.out <- photo
	}

	msg := tgbotapi.NewMessage(c.ID,
		"Lot #"+strconv.Itoa(c.currentLot)+" — "+lot.Title+
			"\nCurrent bid: "+strconv.Itoa(current)+"₽"+
			"\nMin step: "+strconv.Itoa(step)+"₽"+
			"\n\nEnter an amount or pick a quick bid:")
	msg.ParseMode = "html"
	rows := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("+"+strconv.Itoa(step)+" → "+s1+"₽", s1),
			tgbotapi.NewInlineKeyboardButtonData("+"+strconv.Itoa(step*3)+" → "+s2+"₽", s2),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("+"+strconv.Itoa(step*6)+" → "+s3+"₽", s3),
			tgbotapi.NewInlineKeyboardButtonData("◀ Back to list", "back"),
		},
	}
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	c.out <- msg
}

func (c *Chat) SendLotKeyboard(lotNum int) tgbotapi.MessageConfig {
	c.currentLot = lotNum
	auction := c.hub.GetActiveAuction()
	lot := c.hub.GetLotByNum(lotNum)
	if auction == nil || lot == nil {
		return tgbotapi.MessageConfig{}
	}
	state, _ := c.bidSvc.GetLotState(auction.ID, lot.ID)
	current := state.MaxAmount
	if current == 0 {
		current = lot.StartPrice
	}
	step := auction.BidStep
	s1 := strconv.Itoa(current + step)
	msg := tgbotapi.NewMessage(c.ID, "Lot #"+strconv.Itoa(lotNum)+" — "+lot.Title+"\nCurrent bid: "+strconv.Itoa(current)+"₽")
	rows := tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Bid "+s1+"₽", s1))
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows)
	return msg
}

func handlePhone(phone string) string {
	p := strings.NewReplacer("-", "", "(", "", ")", "").Replace(phone)
	if !strings.HasPrefix(p, "+") {
		p = "+" + p
	}
	return p
}
