package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"whatsapp_gRCP/src/app"
	"whatsapp_gRCP/src/cache"
	"whatsapp_gRCP/src/pb/chatMessage"
	"whatsapp_gRCP/src/storage"

	"github.com/redis/go-redis/v9" // Importe o pacote do Redis
	"google.golang.org/grpc"
)

func main() {
	logFile, err := os.OpenFile("server_audit.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Falha ao abrir ou criar o arquivo de log: %v", err)
	}
	defer logFile.Close()

	log.SetOutput(logFile)

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	log.Println("========================================")
	log.Println("[SISTEMA] Servidor gRPC inicializando")
	log.Println("========================================")

	fmt.Println("Inicializando Servidor gRPC... (Auditoria redirecionada para server_audit.log)")

	db, err := storage.Connection()
	if err != nil {
		log.Fatalf("[ERRO FATAL] falha ao conectar no banco: %v", err)
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
		log.Fatalf("[ERRO FATAL] falha ao abrir porta 9090: %v", err)
	}

	log.Println("[SISTEMA] Servidor rodando na porta :9090")
	fmt.Println("Servidor rodando na porta :9090")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("[ERRO FATAL] falha ao servir gRPC: %v", err)
	}
}
