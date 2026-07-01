package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"whatsapp_gRCP/src/pb/chatMessage"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[90m"
)

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

type OutboxMessage struct {
	ReceiverID string
	Content    string
}

var (
	outboxFila []OutboxMessage
	outboxMu   sync.Mutex
)

func workerEnvioMensagens(client chatMessage.ChatMessageServiceClient, myUserID string) {
	for {
		outboxMu.Lock()
		if len(outboxFila) == 0 {
			outboxMu.Unlock()
			time.Sleep(500 * time.Millisecond)
			continue
		}

		msgAtual := outboxFila[0]
		outboxMu.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		res, err := client.SendMessage(ctx, &chatMessage.Message{
			Sender:   myUserID,
			Receiver: msgAtual.ReceiverID,
			Content:  msgAtual.Content,
		})
		cancel()

		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		if res != nil && !res.Success {
			fmt.Printf("\n" + ColorRed + "[Falha no Envio] Destinatário não encontrado ou erro interno." + ColorReset + "\n> ")
			outboxMu.Lock()
			outboxFila = outboxFila[1:]
			outboxMu.Unlock()
			continue
		}

		outboxMu.Lock()
		outboxFila = outboxFila[1:]
		outboxMu.Unlock()

		fmt.Print("\n" + ColorGreen + "[✔] Mensagem pendente enviada com sucesso!" + ColorReset + "\n> ")
	}
}

func main() {
	clearScreen()

	conn, err := grpc.Dial("localhost:9090", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf(ColorRed+"Falha ao conectar: %v"+ColorReset, err)
	}
	defer conn.Close()

	client := chatMessage.NewChatMessageServiceClient(conn)
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(ColorCyan + "===========================" + ColorReset)
	fmt.Println(ColorCyan + "        CHAT gRPC          " + ColorReset)
	fmt.Println(ColorCyan + "===========================" + ColorReset)
	fmt.Print(ColorYellow + "Telefone: " + ColorReset)
	phone, _ := reader.ReadString('\n')
	phone = strings.TrimSpace(phone)

	resp, err := client.Register(context.Background(), &chatMessage.User{
		Phone: phone,
	})

	if err == nil && !resp.Success && resp.MessageInfo == "USER_NOT_FOUND" {
		fmt.Println("\n" + ColorPurple + "[Novo Usuário] Complete seu perfil:" + ColorReset)
		fmt.Print(ColorYellow + "Nome: " + ColorReset)
		name, _ := reader.ReadString('\n')
		fmt.Print(ColorYellow + "Apelido: " + ColorReset)
		nickname, _ := reader.ReadString('\n')

		resp, err = client.Register(context.Background(), &chatMessage.User{
			Phone:    phone,
			Name:     strings.TrimSpace(name),
			Nickname: strings.TrimSpace(nickname),
		})
	}

	if err != nil || !resp.Success {
		log.Fatalf(ColorRed+"Falha na autenticação: %v"+ColorReset, err)
	}

	myUserID := resp.UserId
	fmt.Printf("\n"+ColorGreen+"[Sucesso] Conectado como: %s"+ColorReset+"\n", myUserID)

	go workerEnvioMensagens(client, myUserID)

	var currentReceiverID string
	var lastSenderID string

	go func() {
		for {
			stream, err := client.ChatStream(context.Background(), &chatMessage.StreamRequest{
				UserId: myUserID,
			})

			if err != nil {
				time.Sleep(2 * time.Second)
				continue
			}

			for {
				msg, err := stream.Recv()
				if err != nil {
					fmt.Printf("\n" + ColorRed + "[Conexão Perdida] Servidor indisponível. Tentando reconectar..." + ColorReset + "\n> ")
					break
				}

				if msg.Content == "" {
					if msg.Status == chatMessage.MessageStatus_READ {
						fmt.Printf("\n" + ColorBlue + "[✔✔] Mensagem lida!" + ColorReset + "\n> ")
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

				horarioLive := msg.Timestamp.AsTime().Local().Format("15:04:05")

				fmt.Printf("\n\n" + ColorCyan + "-----------------------------------------" + ColorReset)
				fmt.Printf("\n"+ColorPurple+"🔔 MENSAGEM DE: %s "+ColorGray+"(%s)"+ColorReset, senderNick, realSenderID)
				fmt.Printf("\n"+ColorGray+"🕛 Enviada às: %s"+ColorReset, horarioLive)
				fmt.Printf("\n"+ColorGreen+"💬 %s"+ColorReset, msg.Content)
				fmt.Printf("\n" + ColorYellow + "👉 Digite /r para responder ou /s para sair" + ColorReset)
				fmt.Printf("\n" + ColorCyan + "-----------------------------------------\n" + ColorReset + "> ")

				_, _ = client.UpdateStatus(context.Background(), &chatMessage.UpdateStatusRequest{
					MessageId: msg.Id,
					Status:    chatMessage.MessageStatus_READ,
				})
			}
		}
	}()

	loadHistory := func(receiverID string) {
		history, err := client.GetHistory(context.Background(), &chatMessage.GetHistoryRequest{
			SenderId:   myUserID,
			ReceiverId: receiverID,
		})
		if err == nil && history != nil {
			for _, m := range history.Messages {
				prefix := ColorPurple + "[Destinatário]" + ColorReset
				if m.Sender == myUserID {
					prefix = ColorGreen + "[Você]" + ColorReset
				}

				horario := ColorGray + "[" + m.Timestamp.AsTime().Local().Format("15:04:05") + "]" + ColorReset

				fmt.Printf("%s %s: %s\n", horario, prefix, m.Content)
			}
		} else if err != nil {
			fmt.Printf(ColorRed + "[Aviso] Não foi possível carregar o histórico: servidor offline.\n" + ColorReset)
		}
	}

	for {
		if currentReceiverID == "" {
			fmt.Print("\n" + ColorCyan + "> ID do Destinatário (ou /r para última, /s para sair): " + ColorReset)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if input == "/s" {
				return
			}

			if input == "/r" {
				if lastSenderID == "" {
					fmt.Println(ColorRed + "[Aviso] Nenhuma mensagem recebida ainda." + ColorReset)
					continue
				}
				input = lastSenderID
			}

			if _, err := uuid.Parse(input); err != nil {
				fmt.Println(ColorRed + "[Erro] ID inválido." + ColorReset)
				continue
			}

			currentReceiverID = input
			clearScreen()
			fmt.Printf("\n" + ColorBlue + "=== Conversa Iniciada ===" + ColorReset + "\n")
			loadHistory(currentReceiverID)
			fmt.Println(ColorGray + "\n(Digite a mensagem ou /s para sair da conversa)" + ColorReset)
			continue
		}

		fmt.Print(ColorGreen + "> " + ColorReset)
		content, _ := reader.ReadString('\n')
		content = strings.TrimSpace(content)

		if content == "" {
			continue
		}

		if content == "/s" {
			currentReceiverID = ""
			clearScreen()
			fmt.Println(ColorYellow + "Você saiu da conversa atual." + ColorReset)
			continue
		}

		if content == "/r" {
			if lastSenderID == "" || lastSenderID == currentReceiverID {
				fmt.Println(ColorRed + "[Aviso] Não há outra pessoa para responder." + ColorReset)
				continue
			}
			currentReceiverID = lastSenderID
			clearScreen()
			fmt.Printf("\n"+ColorBlue+"=== Mudou para a conversa com %s ==="+ColorReset+"\n", currentReceiverID)
			loadHistory(currentReceiverID)
			continue
		}

		novaMsg := OutboxMessage{
			ReceiverID: currentReceiverID,
			Content:    content,
		}

		outboxMu.Lock()
		outboxFila = append(outboxFila, novaMsg)
		outboxMu.Unlock()

		fmt.Printf(ColorYellow + "[⏳ Na fila...]\n" + ColorReset)
	}
}
