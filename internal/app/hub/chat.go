package hub

import (
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"
	"publika-auction/internal/app/bids"
	clients_repo "publika-auction/internal/app/clients-repo"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Chat struct {
	ID         int64
	TGUserName string
	client     *clients_repo.Client
	lastAuth   time.Time

	redis  *redis.Client
	dbApp  DB
	clRepo *clients_repo.ClientsRepository
	bds    *bids.BidsStorage

	in  chan tgbotapi.Update
	out chan tgbotapi.Chattable

	currentLot              int
	currentLotMessageID     int
	lotsPicturesMessagesIds []int
}

type DB interface {
	// Auth(context.Context, string) (models.User, error)
}

func (c *Chat) Run(onReturn func()) {
	/*	result, err := c.redis.Get(context.Background(), c.TGUserName).Result()
		if err != nil {*/
begin:

	defer onReturn()
	/*_, err := c.Auth()
	if err != nil {
		return
	}*/
	msg := tgbotapi.MessageConfig{}
	sharePhoneBtn := tgbotapi.NewKeyboardButtonContact("ПОДЕЛИТЬСЯ НОМЕРОМ / SHARE YOUR PHONE NUMBER")
	foundByPhone := false

	cl, ok := c.clRepo.GetClientByTGID(c.ID)
	if ok {
		c.client = &cl
		select {
		case <-c.in:
		default:
		}
		goto authSuccess
	}

	for {
		select {
		case inUpd, ok := <-c.in:
			if !ok {
				log.Err(errors.New("channel is closed")).Int64("id", c.ID).Msg("channel is closed")
				return
			}
			var inMsg *tgbotapi.Message
			if inUpd.Message != nil {
				inMsg = inUpd.Message
			}
			if inUpd.CallbackQuery != nil || inUpd.Message != nil && inUpd.Message.Text == "/start" {
				msg := tgbotapi.NewMessage(c.ID, "Привет, для участия в аукционе нужен ваш номер телефона \n⬇️ Нажми на кнопку \"Поделиться номером\" ⬇️")
				msg.ReplyMarkup = &tgbotapi.ReplyKeyboardMarkup{
					Keyboard:        [][]tgbotapi.KeyboardButton{{sharePhoneBtn}},
					ResizeKeyboard:  true,
					OneTimeKeyboard: false,
					Selective:       false,
				}

				c.out <- msg
				continue
			}
			if inMsg.Contact != nil {
				p := HandlePhone(inMsg.Contact.PhoneNumber)
				cl, found := c.clRepo.GetClient(p)
				if found {
					c.client = &cl
					c.client.TgFirstName = inMsg.Contact.FirstName
					c.client.TgLastName = inMsg.Contact.LastName
					c.client.TgUsername = inMsg.Chat.UserName
					c.client.TgUserId = inMsg.Chat.ID
					if c.client.Messages == nil {
						c.client.Messages = make([]clients_repo.Message, 0)
					}
					foundByPhone = true
					log.Info().Interface("client", cl).Bool("f", foundByPhone).Msg("found by phone")
					goto authSuccess
				} else {
					cl := &clients_repo.Client{
						Name:        inMsg.Contact.FirstName,
						Phone:       inMsg.Contact.PhoneNumber,
						TgFirstName: inMsg.Contact.FirstName,
						TgLastName:  inMsg.Contact.LastName,
						TgUsername:  inMsg.Chat.UserName,
						TgUserId:    inMsg.Chat.ID,
						Messages:    make([]clients_repo.Message, 0),
					}
					c.client = cl
					goto authSuccess
				}
			} else {
				msg := tgbotapi.NewMessage(c.ID, "⬇️ Нажми на кнопку \"Поделиться номером\" ⬇️")
				msg.ReplyMarkup = &tgbotapi.ReplyKeyboardMarkup{
					Keyboard:        [][]tgbotapi.KeyboardButton{{sharePhoneBtn}},
					ResizeKeyboard:  true,
					OneTimeKeyboard: false,
					Selective:       false,
				}
				c.out <- msg
				continue
			}
		}
	}

authSuccess:

	c.clRepo.SetClient(cl.Phone, *c.client)

	msg = tgbotapi.NewMessage(c.ID, "Привет, "+c.client.TgFirstName)
	msg.ReplyMarkup = &tgbotapi.ReplyKeyboardRemove{
		RemoveKeyboard: true,
	}
	c.out <- msg

	if !c.bds.Start {
		msg = tgbotapi.NewMessage(c.ID, "Аукцион скоро начнется...")
		c.out <- msg
	waiting:
		for {
			select {
			case inUpd, ok := <-c.in:
				if !ok {
					log.Err(errors.New("channel is closed")).Int64("id", c.ID).Msg("channel is closed")
					return
				}
				if inUpd.Message != nil {
					c.client.Messages = append(c.client.Messages, clients_repo.Message{c.TGUserName, inUpd.Message.Text, time.Now()})
					c.clRepo.SetClient(cl.Phone, *c.client)
					msg = tgbotapi.NewMessage(c.ID, "Аукцион скоро начнется...")
					c.out <- msg
				}
			default:
				if c.bds.Start == false {
					time.Sleep(3 * time.Second)
				} else {
					break waiting
				}
			}
		}
	}

	c.sendPhotos()

	c.sendLotsKeyboard()

	for {
		select {
		case inUpd, ok := <-c.in:
			if !ok {
				log.Err(errors.New("channel is closed")).Int64("id", c.ID).Msg("channel is closed")
				return
			}
			cl, ok := c.clRepo.GetClientByTGID(c.ID)
			if ok {
				c.client = &cl
			}
			var inMsg *tgbotapi.Message
			if inUpd.Message != nil {
				// message
				inMsg = inUpd.Message
				if inMsg.Text == "/start" {
					goto begin
				}
				sum, _ := strconv.Atoi(inMsg.Text)
				if c.currentLot != 0 && sum != 0 {
					if cl.IsBlocked {
						msg := tgbotapi.NewMessage(c.ID, "Вы в черном списке. Ваши ставки не принимаются")
						c.out <- msg
						continue
					}
					c.AddBet(sum)
					continue
				}
				if inMsg.Text == "/help" {
					c.sendLotsKeyboard()
				}
				if inMsg.Text != "" {
					c.client.Messages = append(c.client.Messages, clients_repo.Message{c.TGUserName, inMsg.Text, time.Now()})
					c.clRepo.SetClient(cl.Phone, *c.client)
				}
				log.Info().Str("tgusername", c.TGUserName).Str("msg", inMsg.Text).Msg("freemessage")
			} else if inUpd.CallbackQuery != nil {
				d := tgbotapi.NewDeleteMessage(inUpd.CallbackQuery.Message.Chat.ID, inUpd.CallbackQuery.Message.MessageID)
				c.out <- d

				cmnd := inUpd.CallbackQuery.Data
				sum, _ := strconv.Atoi(inUpd.CallbackQuery.Data)
				switch {
				case strings.Contains(cmnd, "lot"):
					lotStr := strings.Replace(cmnd, "lot", "", 1)
					lot, _ := strconv.Atoi(lotStr)
					c.currentLot = lot
					c.sendLotKeyboard()
				case sum > 1000:
					if cl.IsBlocked {
						msg := tgbotapi.NewMessage(c.ID, "Вы в черном списке. Ваши ставки не принимаются")
						c.out <- msg
						continue
					}
					c.AddBet(sum)
					// c.sendLotsKeyboard()
				case cmnd == "back":
					c.currentLot = 0
					c.sendLotsKeyboard()
				}

			}

			fmt.Println(inMsg)

		}
	}
}

func HandlePhone(phone string) string {
	p := strings.Replace(phone, "-", "", 10)
	p = strings.Replace(p, "(", "", 10)
	p = "+" + strings.Replace(p, ")", "", 10)
	return p
}

func (c *Chat) AddBet(sum int) {
	newSum, err := c.bds.AddBet(c.currentLot, sum, c.client.Phone, c.client)
	if err != nil && newSum == 123123 {
		msg := tgbotapi.NewMessage(c.ID, "Лот уже продан")
		c.out <- msg
	} else {
		if err != nil && newSum > 0 {
			msg := tgbotapi.NewMessage(c.ID, "Ставка уже выросла до "+strconv.Itoa(newSum)+"р ( минимальный шаг 1000р )")
			c.out <- msg
		}
	}
	if err == nil {
		msg := tgbotapi.NewMessage(c.ID, "Ставка принята ( Лот #"+strconv.Itoa(c.currentLot)+" "+strconv.Itoa(sum)+"р)")
		c.out <- msg
		msg = tgbotapi.NewMessage(c.ID, "...")
		rows := tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Назад", "back"))
		markup := tgbotapi.NewInlineKeyboardMarkup(rows)
		msg.ReplyMarkup = markup
		c.out <- msg
	}
}

func (c *Chat) sendPhotos() {
	imgs := make([][]interface{}, 0)
	row := make([]interface{}, 0)
	for i, item := range c.bds.GetItems() {
		if i%10 == 0 && i != 0 {
			imgs = append(imgs, row)
			row = make([]interface{}, 0)
		}
		row = append(row, tgbotapi.NewInputMediaPhoto(tgbotapi.FileURL(item.Photo)))
	}
	imgs = append(imgs, row)
	for _, d := range imgs {
		nmsg := tgbotapi.NewMediaGroup(c.ID, d)
		c.out <- nmsg
	}
}

func (c *Chat) sendLotsKeyboard() {
	lots := c.bds.GetItems()
	msg := tgbotapi.NewMessage(c.ID, "Выбери лот")
	rows := make([][]tgbotapi.InlineKeyboardButton, 0)
	for i := 0; i < 6; i++ {
		row := make([]tgbotapi.InlineKeyboardButton, 0)
		for j := 0; j < 3; j++ {
			id := i*3 + j + 1
			lot := lots.ById(id)
			if lot == nil {
				continue
			}
			// row = append(row, tgbotapi.NewInlineKeyboardButtonData("Лот #"+strconv.Itoa(i*3+j+1)+" ("+strconv.Itoa(lot.MaxConfirmed)+"р)", "lot"+strconv.Itoa(i*3+j+1)))
			row = append(row, tgbotapi.NewInlineKeyboardButtonData("Лот #"+strconv.Itoa(i*3+j+1), "lot"+strconv.Itoa(i*3+j+1)))
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(row...))
	}
	// rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Обновить", "back")))
	markup := tgbotapi.NewInlineKeyboardMarkup(rows...)
	msg.ReplyMarkup = markup
	c.out <- msg
}

func (c *Chat) sendLotKeyboard() {
	item, err := c.bds.GetItem(c.currentLot)
	if err != nil {
		log.Err(err).Int("currentLot", c.currentLot).Msg("sendLotKeyboard getitem error")
		return
	}
	imgs := []interface{}{tgbotapi.NewInputMediaPhoto(tgbotapi.FileURL(item.Photo))}
	nmsg := tgbotapi.NewMediaGroup(c.ID, imgs)
	c.out <- nmsg

	newSum := strconv.Itoa(item.MaxConfirmed + 1000)
	newSum5 := strconv.Itoa(item.MaxConfirmed + 5000)
	newSum10 := strconv.Itoa(item.MaxConfirmed + 10000)
	msg := tgbotapi.NewMessage(c.ID, "Лот #"+strconv.Itoa(c.currentLot)+". \nТекущая ставка "+strconv.Itoa(item.MaxConfirmed)+"р. \nДля того чтобы предложить свою ставку отправьте сумму ( минимальный шаг 1000р ).\n")
	msg.ParseMode = "html"
	rows := [][]tgbotapi.InlineKeyboardButton{{
		tgbotapi.NewInlineKeyboardButtonData("Поднять до "+newSum, newSum),
		tgbotapi.NewInlineKeyboardButtonData("Поднять до "+newSum5, newSum5),
	}, {
		tgbotapi.NewInlineKeyboardButtonData("Поднять до "+newSum10, newSum10),
		tgbotapi.NewInlineKeyboardButtonData("Назад к списку", "back"),
	},
	}
	markup := tgbotapi.NewInlineKeyboardMarkup(rows...)
	msg.ReplyMarkup = markup
	c.out <- msg
}

/*
	func (c *Chat) Auth() (user models.User, err error) {
		user, err = c.dbApp.Auth(context.Background(), c.TGUserName)
		if err != nil {
			log.Err(err).Str("tg user name", c.TGUserName).Msg("db login error")
			c.Out <- tgbotapi.NewMessage(c.ID, "Ошибка авторизации")
			return user, err
		}
		if user.Error != 0 {
			log.Err(errors.New("error != 0")).Str("tg user name", c.TGUserName).Msg("db login error")
			c.Out <- tgbotapi.NewMessage(c.ID, "Ошибка авторизации")
			return user, err
		}
		c.client = &user
		return user, nil
	}
*/
var datelayout = "02.01.2006"
var phoneRegExp, _ = regexp.Compile("/+[0-9]{7-10}")
