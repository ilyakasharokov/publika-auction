package clients_repo

import (
	"sync"
	"time"
)

type ClientsRepository struct {
	mx      sync.RWMutex
	storage map[string]Client
	mng     DBService
}

type DBService interface {
	GetClients() []Client
	SetClient(client Client)
}

func (c *ClientsRepository) GetClientByTGID(id int64) (Client, bool) {
	c.mx.RLock()
	defer c.mx.RUnlock()
	for _, c := range c.storage {
		if c.TgUserId == id {
			return c, true
		}
	}
	return Client{}, false
}

func (c *ClientsRepository) GetClient(phone string) (Client, bool) {
	c.mx.RLock()
	defer c.mx.RUnlock()
	_, found := c.storage[phone]
	if !found {
		return Client{}, false
	}
	return c.storage[phone], true
}

func (c *ClientsRepository) SetClient(phone string, cl Client) {
	c.mx.Lock()
	c.storage[phone] = cl
	c.mx.Unlock()
	c.mng.SetClient(cl)
}

func (c *ClientsRepository) Block(phone string) {
	c.mx.Lock()
	defer c.mx.Unlock()
	_, found := c.storage[phone]
	if found {
		cl := c.storage[phone]
		cl.IsBlocked = true
		c.storage[phone] = cl
		c.mng.SetClient(cl)
		return
	}
}

func (c *ClientsRepository) GetPhones() []string {
	keys := make([]string, 0)
	for key := range c.storage {
		keys = append(keys, key)
	}
	return keys
}

func (c *ClientsRepository) SetAll(m map[string]Client) {
	c.storage = m
	cls := c.mng.GetClients()
	for _, cl := range cls {
		c.storage[cl.Phone] = cl
	}
}

func (c *ClientsRepository) GetAllWithId() []Client {
	ar := make([]Client, 0)
	for _, cl := range c.storage {
		if cl.TgUserId != 0 {
			ar = append(ar, cl)
		}
	}
	return ar
}

func New(srv DBService) *ClientsRepository {
	repo := ClientsRepository{
		mx:      sync.RWMutex{},
		storage: make(map[string]Client, 0),
		mng:     srv,
	}
	cls := srv.GetClients()
	for _, c := range cls {
		repo.storage[c.Phone] = c
	}
	return &repo
}

type Message struct {
	Author string
	Text   string
	Date   time.Time
}

type Client struct {
	Name  string
	Names []string
	Phone string
	Email string

	TgUsername  string
	TgUserId    int64
	TgFirstName string
	TgLastName  string

	HasBets   bool
	IsBlocked bool
	Messages  []Message
}
