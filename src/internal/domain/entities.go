package domain

import (
	"time"
)

type MessageStatus int

const (
	Sent MessageStatus = iota
	Delivered
	Read
)

type User struct {
	UserID   string `db:"user_id"`
	Name     string `db:"name"`
	NickName string `db:"nick_name"`
}

type Message struct {
	MessageID string        `db:"message_id"`
	Sender    string        `db:"sender"`
	Receiver  string        `db:"receiver"`
	Content   string        `db:"content"`
	Timestamp time.Time     `db:"time_stamp"`
	Status    MessageStatus `db:"status"`
}
