package main

import (
	"fmt"
	"whatsapp_gRCP/src/storage"
)

func main() {
	fmt.Println("Inicializando Servidor")
	storage.Connection()
}
