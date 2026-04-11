package storage

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func Connection() (*sqlx.DB, error) {
	dns := "host=localhost user=postgres_whatsapp password=1234 dbname=whatsapp_grpc port=5432 sslmode=disable"

	db, err := sqlx.Connect("postgres", dns)
	if err != nil {
		log.Fatalln("Error on database conection. Error: ", err)
	}

	if err := createTables(db); err != nil {
		log.Fatal("Error ao criar as tabelas do banco. Error: ", err)
	}

	fmt.Println("Tabelas do banco criadas")

	fmt.Println("Banco de dados conectado")

	return db, nil
}

func createTables(db *sqlx.DB) error {
	schema := `
		CREATE TABLE IF NOT EXISTS tb_users (
			user_id   UUID PRIMARY KEY,
			phone     TEXT NOT NULL UNIQUE,
			name      TEXT NOT NULL,
			nick_name TEXT NOT NULL
		);

		DO $$ 
		BEGIN
    		IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'message_status') THEN
        	CREATE TYPE message_status AS ENUM ('sent', 'delivered', 'read');
    		END IF;
		END $$;

		CREATE TABLE IF NOT EXISTS tb_messages (
			message_id UUID PRIMARY KEY,
			sender     UUID NOT NULL REFERENCES tb_users(user_id),
			receiver   UUID NOT NULL REFERENCES tb_users(user_id),
			content    TEXT NOT NULL,
			time_stamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			status     message_status NOT NULL DEFAULT 'sent'
		);
	`

	_, err := db.Exec(schema)
	return err
}
