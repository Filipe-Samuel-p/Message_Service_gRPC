package main

import (
	"fmt"
	"log"
	"net"
	"whatsapp_gRCP/src/app"
	"whatsapp_gRCP/src/cache"
	"whatsapp_gRCP/src/pb/chatMessage"
	"whatsapp_gRCP/src/storage"

	"github.com/redis/go-redis/v9" // Importe o pacote do Redis
	"google.golang.org/grpc"
)

func main() {
	fmt.Println("Inicializando Servidor gRPC...")

	db, err := storage.Connection()
	if err != nil {
		log.Fatalf("falha ao conectar no banco: %v", err)
	}
	defer db.Close()
	repo := &storage.Repository{DB: db}

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	redisClient := cache.NewRedisClient(rdb)

	serverApp := app.NewChatMessageServer(repo, redisClient)

	grpcServer := grpc.NewServer()
	chatMessage.RegisterChatMessageServiceServer(grpcServer, serverApp)

	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("falha ao abrir porta 9090: %v", err)
	}

	fmt.Println("Servidor rodando na porta :9090")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("falha ao servir gRPC: %v", err)
	}
}
