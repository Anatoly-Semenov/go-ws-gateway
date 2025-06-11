package models

import "time"

type MessageType string

const (
	MessageTypeBalanceUpdate MessageType = "balance_update"
	MessageTypeNotification  MessageType = "notification"
	MessageTypeUserBlock     MessageType = "user_block"
)

type Message struct {
	ID        string      `json:"id"`
	Type      MessageType `json:"type"`
	UserID    string      `json:"user_id"`
	Payload   interface{} `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
}

type BalanceUpdate struct {
	Balance    float64 `json:"balance"`
	Delta      float64 `json:"delta"`
	Currency   string  `json:"currency"`
	Reason     string  `json:"reason,omitempty"`
	ExternalID string  `json:"external_id,omitempty"`
}

type UserBlock struct {
	Blocked   bool   `json:"blocked"`
	Reason    string `json:"reason,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type Notification struct {
	Title   string      `json:"title"`
	Body    string      `json:"body"`
	Data    interface{} `json:"data,omitempty"`
	Actions []Action    `json:"actions,omitempty"`
}

type Action struct {
	Label string `json:"label"`
	URL   string `json:"url,omitempty"`
	Type  string `json:"type"`
	Data  string `json:"data,omitempty"`
}
