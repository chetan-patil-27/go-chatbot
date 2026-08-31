package models

import "time"

type Message struct {
	ID        int64     `json:"id"`
	ChatID    int64     `json:"chat_id"`
	Sender    string    `json:"sender"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}
