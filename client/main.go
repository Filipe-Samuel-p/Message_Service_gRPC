package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"whatsapp_gRCP/src/pb/chatMessage"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.Dial("localhost:9090", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Falha ao conectar: %v", err)
	}
	defer conn.Close()

	client := chatMessage.NewChatMessageServiceClient(conn)
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=== CHAT gRPC ===")
	fmt.Print("Telefone: ")
	phone, _ := reader.ReadString('\n')
	phone = strings.TrimSpace(phone)

	resp, err := client.Register(context.Background(), &chatMessage.User{
		Phone: phone,
	})

	if err == nil && !resp.Success && resp.MessageInfo == "USER_NOT_FOUND" {
		fmt.Println("\n[Novo Usuário] Complete seu perfil:")
		fmt.Print("Nome: ")
		name, _ := reader.ReadString('\n')
		fmt.Print("Apelido: ")
		nickname, _ := reader.ReadString('\n')

		resp, err = client.Register(context.Background(), &chatMessage.User{
			Phone:    phone,
			Name:     strings.TrimSpace(name),
			Nickname: strings.TrimSpace(nickname),
		})
	}

	if err != nil || !resp.Success {
		log.Fatalf("Falha na autenticação: %v", err)
	}

	myUserID := resp.UserId

	fmt.Printf("\n[Sucesso] Conectado como: %s\n", myUserID)

	stream, err := client.ChatStream(context.Background(), &chatMessage.StreamRequest{
		UserId: myUserID,
	})
	if err != nil {
		log.Fatalf("Erro ao abrir stream: %v", err)
	}

	var currentReceiverID string
	var lastSenderID string

	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				log.Fatalf("\n[Conexão Perdida]\n")
			}

			if msg.Content == "" {
				if msg.Status == chatMessage.MessageStatus_READ {
					fmt.Printf("\n[✔✔] Mensagem lida!\n> ")
				}
				continue
			}

			senderParts := strings.Split(msg.Sender, "|")
			realSenderID := senderParts[0]
			senderNick := "Desconhecido"
			if len(senderParts) > 1 {
				senderNick = senderParts[1]
			}

			lastSenderID = realSenderID

			fmt.Printf("\n\n-----------------------------------------")
			fmt.Printf("\n🔔 MENSAGEM DE: %s (%s)", senderNick, realSenderID)
			fmt.Printf("\n💬 %s", msg.Content)
			fmt.Printf("\n👉 Digite /r para responder ou /s para sair")
			fmt.Printf("\n-----------------------------------------\n> ")

			_, _ = client.UpdateStatus(context.Background(), &chatMessage.UpdateStatusRequest{
				MessageId: msg.Id,
				Status:    chatMessage.MessageStatus_READ,
			})
		}
	}()

	loadHistory := func(receiverID string) {
		history, err := client.GetHistory(context.Background(), &chatMessage.GetHistoryRequest{
			SenderId:   myUserID,
			ReceiverId: receiverID,
		})
		if err == nil && history != nil {
			for _, m := range history.Messages {

				prefix := "[Destinatário]"
				if m.Sender == myUserID {
					prefix = "[Você]"
				}
				fmt.Printf("%s: %s\n", prefix, m.Content)
			}
		}
	}

	for {
		if currentReceiverID == "" {
			fmt.Print("\n> ID do Destinatário (ou /r para última mensagem, /s para sair do app): ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if input == "/s" {
				return
			}

			if input == "/r" {
				if lastSenderID == "" {
					fmt.Println("[Aviso] Nenhuma mensagem recebida ainda.")
					continue
				}
				input = lastSenderID
			}

			if _, err := uuid.Parse(input); err != nil {
				fmt.Println("[Erro] ID inválido.")
				continue
			}

			currentReceiverID = input
			fmt.Printf("\n=== Conversa Iniciada ===\n")
			loadHistory(currentReceiverID)
			fmt.Println("\n(Digite a mensagem ou /s para sair da conversa)")
			continue
		}

		fmt.Print("> ")
		content, _ := reader.ReadString('\n')
		content = strings.TrimSpace(content)

		if content == "" {
			continue
		}

		if content == "/s" {
			currentReceiverID = ""
			fmt.Println("Você saiu da conversa atual.")
			continue
		}

		if content == "/r" {
			if lastSenderID == "" || lastSenderID == currentReceiverID {
				fmt.Println("[Aviso] Não há outra pessoa para responder.")
				continue
			}
			currentReceiverID = lastSenderID
			fmt.Printf("\n=== Mudou para a conversa com %s ===\n", currentReceiverID)
			loadHistory(currentReceiverID)
			continue
		}

		_, err := client.SendMessage(context.Background(), &chatMessage.Message{
			Sender:   myUserID,
			Receiver: currentReceiverID,
			Content:  content,
		})

		if err != nil {
			fmt.Printf("\n[Erro]: %v\n> ", err)
		}
	}
}
