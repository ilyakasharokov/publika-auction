package clients_repo

import (
	"sync"
)

type ClientsRepository struct {
	mx      sync.RWMutex
	storage map[string]Client
	mng     DBService
}

type DBService interface {
}

func (c *ClientsRepository) GetClient(phone string) (Client, bool) {
	c.mx.RLock()
	_, found := c.storage[phone]
	if !found {
		return Client{}, false
	}
	defer c.mx.RUnlock()
	return c.storage[phone], true
}

func (c *ClientsRepository) SetClient(phone string, cl Client) {
	c.mx.RLock()
	defer c.mx.RUnlock()
	_, found := c.storage[phone]
	if found {
		c.storage[phone] = cl
		return
	}
	c.storage[phone] = cl
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
}

func New(srv DBService) *ClientsRepository {
	repo := ClientsRepository{
		mx:      sync.RWMutex{},
		storage: make(map[string]Client, 0),
		mng:     srv,
	}
	return &repo
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
	HasBets     bool
}
