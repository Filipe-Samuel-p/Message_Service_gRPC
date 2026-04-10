package domain

import (
	"time"

	"github.com/google/uuid"
)

type MessageStatus string

const (
	Sent      MessageStatus = "sent"
	Delivered MessageStatus = "delivered"
	Read      MessageStatus = "read"
)

type User struct {
	UserID   uuid.UUID `db:"user_id"`
	Name     string    `db:"name"`
	NickName string    `db:"nick_name"`
}

type Message struct {
	MessageID uuid.UUID     `db:"message_id"`
	Sender    string        `db:"sender"`
	Receiver  string        `db:"receiver"`
	Content   string        `db:"content"`
	Timestamp time.Time     `db:"time_stamp"`
	Status    MessageStatus `db:"status"`
}
