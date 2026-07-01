package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
	"whatsapp_gRCP/src/cache"
	"whatsapp_gRCP/src/domain"
	"whatsapp_gRCP/src/pb/chatMessage"
	"whatsapp_gRCP/src/storage"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ChatMessageServer struct {
	repo        *storage.Repository
	cache       *cache.RedisClient
	mu          sync.RWMutex
	connections map[uuid.UUID]chatMessage.ChatMessageService_ChatStreamServer
	chatMessage.UnimplementedChatMessageServiceServer
}

func NewChatMessageServer(repository *storage.Repository, redisCache *cache.RedisClient) *ChatMessageServer {
	return &ChatMessageServer{
		repo:        repository,
		cache:       redisCache,
		connections: make(map[uuid.UUID]chatMessage.ChatMessageService_ChatStreamServer),
	}
}

func (s *ChatMessageServer) Register(ctx context.Context, user *chatMessage.User) (*chatMessage.RegisterResponse, error) {

	existingUser, err := s.repo.GetUserByPhone(user.Phone)

	if err == nil {
		return &chatMessage.RegisterResponse{
			Success:     true,
			MessageInfo: "Usuário já existe. Bem-vindo de volta!",
			UserId:      existingUser.UserID.String(),
		}, nil
	}

	if user.Name == "" {
		return &chatMessage.RegisterResponse{
			Success:     false,
			MessageInfo: "USER_NOT_FOUND",
		}, nil
	}

	if err != sql.ErrNoRows {
		return &chatMessage.RegisterResponse{
			Success:     false,
			MessageInfo: "Erro interno ao verificar usuário no banco de dados.",
		}, nil
	}

	newUser := domain.User{
		Phone:    user.Phone,
		Name:     user.Name,
		NickName: user.Nickname,
	}

	newUser, err = s.repo.SaveUser(newUser)
	if err != nil {
		return &chatMessage.RegisterResponse{
			Success:     false,
			MessageInfo: "Erro ao cadastrar: " + err.Error(),
		}, nil
	}

	return &chatMessage.RegisterResponse{
		Success:     true,
		MessageInfo: "Usuário cadastrado com sucesso!",
		UserId:      newUser.UserID.String(),
	}, nil
}

func (s *ChatMessageServer) SendMessage(ctx context.Context, message *chatMessage.Message) (*chatMessage.SendMessageResponse, error) {

	senderId, err := uuid.Parse(message.Sender)
	if err != nil {
		return nil, fmt.Errorf("erro no parse do senderId: %w", err)
	}

	receiverId, err := uuid.Parse(message.Receiver)
	if err != nil {
		return nil, fmt.Errorf("erro no parse do receiverId: %w", err)
	}

	_, err = s.repo.GetUserByID(receiverId)
	if err != nil {

		log.Printf("[AVISO] Tentativa de envio para usuário não cadastrado. Destinatário: %s", receiverId.String())
		return &chatMessage.SendMessageResponse{
			Success: false,
		}, nil
	}

	senderUser, _ := s.repo.GetUserByID(senderId)

	newMessage := domain.Message{
		Sender:    senderId,
		Receiver:  receiverId,
		Content:   message.Content,
		Timestamp: time.Now().UTC(),
	}

	newMessage, err = s.repo.SaveMessage(newMessage)
	if err != nil {
		log.Printf("[ERRO] Falha ao salvar mensagem no Postgres: %v", err)
		return &chatMessage.SendMessageResponse{Success: false}, nil
	}

	isOnline, _ := s.cache.IsOnline(ctx, receiverId)

	if isOnline {
		s.mu.RLock()
		targetStream, connected := s.connections[receiverId]
		s.mu.RUnlock()

		if connected {
			msgParaEnviar := &chatMessage.Message{
				Id:        newMessage.MessageID.String(),
				Sender:    newMessage.Sender.String() + "|" + senderUser.NickName,
				Receiver:  newMessage.Receiver.String(),
				Content:   newMessage.Content,
				Status:    toProtoStatus(newMessage.Status),
				Timestamp: timestamppb.New(newMessage.Timestamp),
			}
			_ = targetStream.Send(msgParaEnviar)
		}
	}

	return &chatMessage.SendMessageResponse{
		Success:   true,
		MessageId: newMessage.MessageID.String(),
	}, nil
}

func (s *ChatMessageServer) UpdateStatus(ctx context.Context, req *chatMessage.UpdateStatusRequest) (*chatMessage.UpdateStatusResponse, error) {

	messageID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, fmt.Errorf("invalid message ID")
	}

	statusDomain := toDomainStatus(req.Status)

	senderUUID, err := s.repo.UpdateMessageStatus(messageID, statusDomain)
	if err != nil {
		return &chatMessage.UpdateStatusResponse{Success: false}, nil
	}

	s.mu.RLock()
	targetStream, isSenderOnline := s.connections[senderUUID]
	s.mu.RUnlock()

	if isSenderOnline {
		notification := &chatMessage.Message{
			Id:     req.MessageId,
			Status: req.Status,
		}
		_ = targetStream.Send(notification)
	}

	return &chatMessage.UpdateStatusResponse{Success: true}, nil
}

func (s *ChatMessageServer) GetHistory(ctx context.Context, req *chatMessage.GetHistoryRequest) (*chatMessage.HistoryResponse, error) {
	senderID, err := uuid.Parse(req.SenderId)
	if err != nil {
		return nil, fmt.Errorf("ID do remetente inválido: %w", err)
	}

	receiverID, err := uuid.Parse(req.ReceiverId)
	if err != nil {
		return nil, fmt.Errorf("ID do destinatário inválido: %w", err)
	}

	readMsgs, err := s.repo.MarkMessagesAsRead(receiverID, senderID)
	if err != nil {
		// Atualizado para manter o padrão
		log.Printf("[AVISO] Erro ao marcar mensagens como read: %v", err)
	} else {
		for _, msg := range readMsgs {
			s.mu.RLock()
			senderStream, online := s.connections[msg.Sender]
			s.mu.RUnlock()

			if online {
				_ = senderStream.Send(&chatMessage.Message{
					Id:     msg.MessageID.String(),
					Status: chatMessage.MessageStatus_READ,
				})
			}
		}
	}

	historyDomain, err := s.repo.GetHistory(senderID, receiverID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar histórico: %w", err)
	}

	var grpcMessages []*chatMessage.Message
	for _, dbMsg := range historyDomain {
		grpcMessages = append(grpcMessages, &chatMessage.Message{
			Id:        dbMsg.MessageID.String(),
			Sender:    dbMsg.Sender.String(),
			Receiver:  dbMsg.Receiver.String(),
			Content:   dbMsg.Content,
			Status:    toProtoStatus(dbMsg.Status),
			Timestamp: timestamppb.New(dbMsg.Timestamp),
		})
	}

	return &chatMessage.HistoryResponse{Messages: grpcMessages}, nil
}

func (s *ChatMessageServer) ChatStream(req *chatMessage.StreamRequest, stream chatMessage.ChatMessageService_ChatStreamServer) error {
	ctx := stream.Context()

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return fmt.Errorf("ID de usuário inválido: %w", err)
	}

	s.mu.RLock()
	_, alreadyConnected := s.connections[userID]
	s.mu.RUnlock()

	if alreadyConnected {

		log.Printf("[AUDITORIA - ALERTA] Tentativa de login simultâneo bloqueada para o UUID: %s", userID.String())
		return fmt.Errorf("acesso negado: este usuário já possui uma sessão ativa em outro terminal")
	}

	s.mu.Lock()
	s.connections[userID] = stream
	s.mu.Unlock()

	s.cache.SetUserOnline(ctx, userID)

	deliveredMsgs, err := s.repo.MarkMessagesAsDelivered(userID)
	if err != nil {
		log.Printf("[AVISO] Erro ao marcar mensagens como delivered para usuário %s: %v", userID.String(), err)
	} else {
		for _, msg := range deliveredMsgs {
			s.mu.RLock()
			senderStream, online := s.connections[msg.Sender]
			s.mu.RUnlock()

			if online {
				_ = senderStream.Send(&chatMessage.Message{
					Id:     msg.MessageID.String(),
					Status: chatMessage.MessageStatus_DELIVERED,
				})
			}

			senderUser, _ := s.repo.GetUserByID(msg.Sender)
			_ = stream.Send(&chatMessage.Message{
				Id:        msg.MessageID.String(),
				Sender:    msg.Sender.String() + "|" + senderUser.NickName,
				Receiver:  msg.Receiver.String(),
				Content:   msg.Content,
				Status:    toProtoStatus(msg.Status),
				Timestamp: timestamppb.New(msg.Timestamp),
			})
		}
	}

	log.Printf("[AUDITORIA] Usuário conectado (Stream Inciado): %s", userID.String())

	defer func() {
		s.mu.Lock()
		delete(s.connections, userID)
		s.mu.Unlock()

		log.Printf("[AUDITORIA] Usuário desconectado (Stream Encerrado): %s", userID.String())
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			s.cache.KeepAlive(ctx, userID)
		}
	}
}

// Auxiliares

func toDomainStatus(protoStatus chatMessage.MessageStatus) domain.MessageStatus {
	name, thereIs := chatMessage.MessageStatus_name[int32(protoStatus)]
	if !thereIs {
		return domain.MessageStatus("sent")
	}

	nomeMinusculo := strings.ToLower(name)
	return domain.MessageStatus(nomeMinusculo)
}

func toProtoStatus(domainStatus domain.MessageStatus) chatMessage.MessageStatus {
	upper := strings.ToUpper(string(domainStatus))
	valueInt, thereIs := chatMessage.MessageStatus_value[upper]
	if !thereIs {
		return chatMessage.MessageStatus_SENT
	}
	return chatMessage.MessageStatus(valueInt)
}
