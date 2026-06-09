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
	out chan tgbotapi.Chattable // structured messages: photos, keyboards

	hub       *Hub
	bidSvc    *bidsvc.Service
	clientSvc *clientsvc.Service
	currentLot int
}

// sendText routes plain text through the TG queue (rate-limited, retry on 429).
// Falls back to the direct out channel if broadcaster is not yet connected.
func (c *Chat) sendText(text string) {
	c.hub.mu.RLock()
	b := c.hub.broadcaster
	c.hub.mu.RUnlock()
	if b != nil {
		b.Send(c.ID, text)
		return
	}
	select {
	case c.out <- tgbotapi.NewMessage(c.ID, text):
	default:
		log.Warn().Int64("id", c.ID).Msg("chat out full, text message dropped")
	}
}

func (c *Chat) SendTo(update tgbotapi.Update) {
	c.in <- update
}

func (c *Chat) Run(onReturn func()) {
	defer onReturn()

	sharePhoneBtn := tgbotapi.NewKeyboardButtonContact("ПОДЕЛИТЬСЯ НОМЕРОМ")

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
				msg := tgbotapi.NewMessage(c.ID, "Привет! Для участия в аукционе нужен ваш номер телефона.\n⬇️ Нажмите «Поделиться номером» ⬇️")
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
			msg := tgbotapi.NewMessage(c.ID, "⬇️ Нажмите «Поделиться номером» ⬇️")
			msg.ReplyMarkup = &tgbotapi.ReplyKeyboardMarkup{
				Keyboard:        [][]tgbotapi.KeyboardButton{{sharePhoneBtn}},
				ResizeKeyboard:  true,
				OneTimeKeyboard: false,
			}
			c.out <- msg
		}
	}

authSuccess:
	greeting := tgbotapi.NewMessage(c.ID, "Привет, "+c.client.TgFirstName+"!")
	greeting.ReplyMarkup = &tgbotapi.ReplyKeyboardRemove{RemoveKeyboard: true}
	c.out <- greeting

	if !c.hub.IsStarted() {
		c.sendText("Аукцион скоро начнётся, ожидайте!")
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
				c.sendText("Аукцион скоро начнётся, ожидайте!")
			default:
				if !c.hub.IsStarted() {
					time.Sleep(3 * time.Second)
				} else {
					break waitStart
				}
			}
		}
	}

	c.sendLotsKeyboard()

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
						c.sendText("Вы в чёрном списке. Ставки не принимаются.")
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
						c.sendText("Вы в чёрном списке. Ставки не принимаются.")
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
		c.sendText("Аукцион не активен.")
		return
	}
	lot := c.hub.GetLotByNum(c.currentLot)
	if lot == nil {
		c.sendText("Лот не найден.")
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
			c.sendText("Текущая ставка " + strconv.Itoa(tooLow.Current) + "₽ (минимальный шаг " + strconv.Itoa(auction.BidStep) + "₽)")
			return
		}
		if errors.Is(err, bidsvc.ErrLotSold) {
			c.sendText("Лот уже продан.")
			return
		}
		c.sendText("Попробуйте ещё раз.")
		return
	}
	// Bid accepted — send with Back keyboard via the direct channel (structural msg).
	msg := tgbotapi.NewMessage(c.ID, "Ставка принята: Лот #"+strconv.Itoa(c.currentLot)+" — "+strconv.Itoa(sum)+"₽")
	rows := tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("◀ Назад", "back"))
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows)
	select {
	case c.out <- msg:
	default:
		// Queue full — fall back to plain text via broadcaster.
		c.sendText("Ставка принята: Лот #" + strconv.Itoa(c.currentLot) + " — " + strconv.Itoa(sum) + "₽")
	}
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
		c.sendText("Активных лотов пока нет.")
		return
	}
	msg := tgbotapi.NewMessage(c.ID, "Выберите лот:")
	rows := make([][]tgbotapi.InlineKeyboardButton, 0)
	row := make([]tgbotapi.InlineKeyboardButton, 0)
	for _, lot := range activeLots {
		if len(row) == 3 {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(row...))
			row = make([]tgbotapi.InlineKeyboardButton, 0)
		}
		row = append(row, tgbotapi.NewInlineKeyboardButtonData("Лот #"+strconv.Itoa(lot.Num), "lot"+strconv.Itoa(lot.Num)))
	}
	if len(row) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(row...))
	}
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	select {
	case c.out <- msg:
	default:
		log.Warn().Int64("id", c.ID).Msg("chat out full, lots keyboard dropped")
	}
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
		select {
		case c.out <- photo:
		default:
		}
	}

	msg := tgbotapi.NewMessage(c.ID,
		"Лот #"+strconv.Itoa(c.currentLot)+" — "+lot.Title+
			"\nТекущая ставка: "+strconv.Itoa(current)+"₽"+
			"\nМинимальный шаг: "+strconv.Itoa(step)+"₽"+
			"\n\nВведите сумму или выберите быструю ставку:")
	msg.ParseMode = "html"
	rows := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("+"+strconv.Itoa(step)+" → "+s1+"₽", s1),
			tgbotapi.NewInlineKeyboardButtonData("+"+strconv.Itoa(step*3)+" → "+s2+"₽", s2),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("+"+strconv.Itoa(step*6)+" → "+s3+"₽", s3),
			tgbotapi.NewInlineKeyboardButtonData("◀ К списку", "back"),
		},
	}
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	select {
	case c.out <- msg:
	default:
		log.Warn().Int64("id", c.ID).Msg("chat out full, lot keyboard dropped")
	}
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
	msg := tgbotapi.NewMessage(c.ID, "Лот #"+strconv.Itoa(lotNum)+" — "+lot.Title+"\nТекущая ставка: "+strconv.Itoa(current)+"₽")
	rows := tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Поднять до "+s1+"₽", s1))
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
