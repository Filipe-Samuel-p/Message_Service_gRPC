package storage

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func Connection() {
	dns := "host=localhost user=postgres_whatsapp password=1234 dbname=whatsapp_grpc port=5432 sslmode=disable"

	db, err := sqlx.Connect("postgres", dns)
	if err != nil {

		log.Fatalln("Error on database conection. Error: ", err)
	}

	defer db.Close()

	if err := createTables(db); err != nil {
		log.Fatal("Error ao criar as tabelas do banco. Error: ", err)
	}

	fmt.Println("Tabelas do banco criadas")

	fmt.Println("Banco de dados conectado")
}

func createTables(db *sqlx.DB) error {
	schema := `
		CREATE TABLE IF NOT EXISTS users (
			user_id   TEXT PRIMARY KEY,
			name      TEXT NOT NULL,
			nick_name TEXT NOT NULL
		);

		CREATE TYPE message_status AS ENUM ('sent', 'delivered', 'read');

		CREATE TABLE IF NOT EXISTS messages (
			message_id TEXT PRIMARY KEY,
			sender     TEXT NOT NULL REFERENCES users(user_id),
			receiver   TEXT NOT NULL REFERENCES users(user_id),
			content    TEXT NOT NULL,
			time_stamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			status     message_status NOT NULL DEFAULT 'sent'
		);
	`

	_, err := db.Exec(schema)
	return err
}
