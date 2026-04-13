package storage

import (
	"fmt"
	"whatsapp_gRCP/src/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	DB *sqlx.DB
}

func (r *Repository) SaveUser(user domain.User) (domain.User, error) {

	newID := uuid.New()
	user.UserID = newID

	query := `
		INSERT INTO tb_users (user_id, phone, name, nick_name) 
		VALUES ($1, $2, $3, $4)
	`

	_, err := r.DB.Exec(query, user.UserID, user.Phone, user.Name, user.NickName)
	if err != nil {
		return domain.User{}, fmt.Errorf("erro fatal ao inserir user: %w", err)
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

func (r *Repository) GetUserByPhone(phone string) (domain.User, error) {
	var user domain.User
	err := r.DB.Get(&user, "SELECT * FROM tb_users WHERE phone = $1", phone)
	return user, err
}

func (r *Repository) SaveMessage(msg domain.Message) (domain.Message, error) {
	msg.MessageID = uuid.New()

	if msg.Status == "" {
		msg.Status = domain.Sent
	}

	query := `
		INSERT INTO tb_messages (message_id, sender, receiver, content, time_stamp, status) 
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.DB.Exec(query, msg.MessageID, msg.Sender, msg.Receiver, msg.Content, msg.Timestamp, msg.Status)
	if err != nil {
		return domain.Message{}, fmt.Errorf("erro fatal ao inserir message: %w", err)
	}

	return msg, nil
}

func (r *Repository) UpdateMessageStatus(messageID uuid.UUID, status domain.MessageStatus) (uuid.UUID, error) {
	var senderID uuid.UUID

	query := `
        UPDATE tb_messages 
        SET status = $1
        WHERE message_id = $2
        RETURNING sender
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

func (r *Repository) MarkMessagesAsDelivered(receiverID uuid.UUID) ([]domain.Message, error) {
	query := `
        UPDATE tb_messages
        SET status = 'delivered'
        WHERE receiver = $1 AND status = 'sent'
        RETURNING message_id, sender, receiver, content, time_stamp, status
    `
	var messages []domain.Message
	err := r.DB.Select(&messages, query, receiverID)
	if err != nil {
		return nil, fmt.Errorf("erro ao marcar mensagens como delivered: %w", err)
	}
	return messages, nil
}

func (r *Repository) MarkMessagesAsRead(senderID, receiverID uuid.UUID) ([]domain.Message, error) {
	query := `
        UPDATE tb_messages
        SET status = 'read'
        WHERE sender = $1 AND receiver = $2 AND status != 'read'
        RETURNING message_id, sender, receiver, content, time_stamp, status
    `
	var messages []domain.Message
	err := r.DB.Select(&messages, query, senderID, receiverID)
	if err != nil {
		return nil, fmt.Errorf("erro ao marcar mensagens como read: %w", err)
	}
	return messages, nil
}
