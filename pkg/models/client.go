package models

import (
	"sync"
	"time"
)

type Client struct {
	ID           string
	UserID       string
	ConnectionID string
	Connected    time.Time
	LastActive   time.Time
	Metadata     map[string]interface{}
	Mutex        sync.RWMutex
}

func NewClient(id string, userID string) *Client {
	now := time.Now()
	return &Client{
		ID:         id,
		UserID:     userID,
		Connected:  now,
		LastActive: now,
		Metadata:   make(map[string]interface{}),
	}
}

func (c *Client) UpdateLastActive() {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	c.LastActive = time.Now()
}

func (c *Client) SetMetadata(key string, value interface{}) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	c.Metadata[key] = value
}

func (c *Client) GetMetadata(key string) (interface{}, bool) {
	c.Mutex.RLock()
	defer c.Mutex.RUnlock()
	val, ok := c.Metadata[key]
	return val, ok
}
