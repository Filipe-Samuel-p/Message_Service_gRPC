package main

import (
	"fmt"
	"log"
	"whatsapp_gRCP/src/storage"
)

func main() {
	fmt.Println("Inicializando Servidor")

	db, err := storage.Connection()
	if err != nil {
		log.Fatalf("falha ao conectar no banco: %v", err)
	}

	defer db.Close()

	/* repository := *&storage.Repository{DB: db}

	newUser := domain.User{
		Name:     "Filipe Samuel",
		NickName: "Samuca",
	}

	user, err := repository.SaveUser(newUser)
	if err != nil {
		fmt.Print("Errrorrr")
	}

	fmt.Print(user) */

}
