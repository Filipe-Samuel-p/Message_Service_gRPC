package storage

import (
	"fmt"
	"time"
	"whatsapp_gRCP/src/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	DB *sqlx.DB
}

func (r *Repository) SaveUser(user domain.User) (domain.User, error) {

	user_id, err := uuid.NewRandom()
	if err != nil {
		return domain.User{}, fmt.Errorf("Error new uuid. Error: %w", err)
	}

	user.UserID = user_id

	_, err = r.DB.NamedExec("INSERT INTO tb_users (user_id, phone, name, nick_name) VALUES (:user_id, :phone, :name, :nick_name)", user)
	if err != nil {
		return domain.User{}, fmt.Errorf("error saving user on DB: %w", err)
	}

	return user, nil
}

func (r *Repository) GetUserByID(id uuid.UUID) (domain.User, error) {
	var user domain.User
	err := r.DB.Get(&user, "SELECT * FROM tb_users WHERE user_id = $1", id)
	if err != nil {
		return domain.User{}, fmt.Errorf("error getting user by id: %w", err)
	}
	return user, nil
}

func (r *Repository) SaveMessage(message domain.Message) (domain.Message, error) {
	messageID, err := uuid.NewRandom()
	if err != nil {
		return domain.Message{}, fmt.Errorf("Error new uuid. Error: %w", err)
	}

	message.MessageID = messageID
	message.Timestamp = time.Now().UTC()
	message.Status = domain.Sent

	query := ` INSERT INTO tb_messages (message_id, sender, receiver, content, time_stamp, status) 
				VALUES (:message_id, :sender, :receiver, :content, :time_stamp, :status)
	`

	_, err = r.DB.NamedExec(query, message)
	if err != nil {
		return domain.Message{}, fmt.Errorf("error saving message on DB: %w", err)
	}

	return message, nil

}

func (r *Repository) UpdateMessageStatus(messageID uuid.UUID, status domain.MessageStatus) (uuid.UUID, error) {
	var senderID uuid.UUID

	query := `
        UPDATE tb_messages 
        SET status = $1
        WHERE message_id = $2
        RETURNING sender_id
    `

	err := r.DB.QueryRow(query, status, messageID).Scan(&senderID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("error updating status on DB: %w", err)
	}

	return senderID, nil
}

func (r *Repository) GetHistory(userID_A uuid.UUID, userID_B uuid.UUID) ([]domain.Message, error) {
	query := ` SELECT * 
			   FROM tb_messages
			   WHERE (receiver = $1 AND sender = $2) OR (receiver = $2 AND sender = $1)
			   ORDER BY time_stamp ASC
	`

	var history []domain.Message
	err := r.DB.Select(&history, query, userID_A, userID_B)
	if err != nil {
		return nil, fmt.Errorf("Error on Get History: %w", err)
	}

	return history, nil

}
