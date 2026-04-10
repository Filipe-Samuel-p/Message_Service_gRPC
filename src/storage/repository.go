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

	user_id, err := uuid.NewRandom()
	if err != nil {
		fmt.Errorf("Error new uuid. Error: %w", err)
	}

	user.UserID = user_id

	_, err = r.DB.NamedExec("INSERT INTO tb_users (user_id, name, nick_name) VALUES (:user_id, :name,:nick_name)", user)
	if err != nil {
		fmt.Errorf("error saving user on DB: %w", err)
	}

	return user, nil
}
