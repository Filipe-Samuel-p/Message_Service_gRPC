package app

import (
	"context"
	"fmt"
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

func (s *ChatMessageServer) Register(ctx context.Context, user *chatMessage.User) (*chatMessage.RegisterResponse, error) {

	var err error

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
		UserId:      uuid.NewString(),
	}, nil
}

func (s *ChatMessageServer) SendMessage(ctx context.Context, message *chatMessage.Message) (*chatMessage.SendMessageResponse, error) {

	senderId, err := uuid.Parse(message.Sender)
	if err != nil {
		return nil, fmt.Errorf("Error on parse senderId: %w", err)
	}

	receiverId, err := uuid.Parse(message.Receiver)
	if err != nil {
		return nil, fmt.Errorf("Error on parse receiverId: %w", err)
	}

	newMessage := domain.Message{
		Sender:    senderId,
		Receiver:  receiverId,
		Content:   message.Content,
		Timestamp: message.Timestamp.AsTime(),
	}

	newMessage, err = s.repo.SaveMessage(newMessage)
	if err != nil {
		return &chatMessage.SendMessageResponse{
			Success:   false,
			MessageId: newMessage.MessageID.String(),
		}, nil
	}

	isOnline, _ := s.cache.IsOnline(ctx, receiverId)

	if isOnline {
		s.mu.RLock()
		targetStream, connected := s.connections[receiverId]
		s.mu.RUnlock()

		if connected {
			msgToSend := &chatMessage.Message{
				Id:        newMessage.MessageID.String(),
				Sender:    newMessage.Sender.String(),
				Receiver:  newMessage.Receiver.String(),
				Content:   newMessage.Content,
				Status:    toProtoStatus(newMessage.Status),
				Timestamp: timestamppb.New(newMessage.Timestamp),
			}

			_ = targetStream.Send(msgToSend)
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
		return nil, fmt.Errorf("ID do remetente com formato inválido: %w", err)
	}

	receiverID, err := uuid.Parse(req.ReceiverId)
	if err != nil {
		return nil, fmt.Errorf("ID do destinatário com formato inválido: %w", err)
	}

	historyDomain, err := s.repo.GetHistory(senderID, receiverID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar histórico: %w", err)
	}

	var grpcMessages []*chatMessage.Message

	for _, dbMsg := range historyDomain {
		grpcMsg := &chatMessage.Message{
			Id:        dbMsg.MessageID.String(),
			Sender:    dbMsg.Sender.String(),
			Receiver:  dbMsg.Receiver.String(),
			Content:   dbMsg.Content,
			Status:    toProtoStatus(dbMsg.Status),
			Timestamp: timestamppb.New(dbMsg.Timestamp),
		}
		grpcMessages = append(grpcMessages, grpcMsg)
	}

	return &chatMessage.HistoryResponse{
		Messages: grpcMessages,
	}, nil
}

func (s *ChatMessageServer) ChatStream(req *chatMessage.StreamRequest, stream chatMessage.ChatMessageService_ChatStreamServer) error {

	ctx := stream.Context()

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return fmt.Errorf("ID de usuário inválido: %w", err)
	}

	s.mu.Lock()
	s.connections[userID] = stream
	s.mu.Unlock()

	err = s.cache.SetUserOnline(ctx, userID)
	if err != nil {
		fmt.Printf("[Aviso] Falha ao setar usuário online no Redis: %v\n", err)
	}

	fmt.Printf("Usuário conectado: %s\n", userID.String())

	defer func() {
		s.mu.Lock()
		delete(s.connections, userID)
		s.mu.Unlock()
		fmt.Printf("Usuário desconectado: %s\n", userID.String())
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {

		case <-ctx.Done():
			return nil

		case <-ticker.C:
			err := s.cache.KeepAlive(ctx, userID)
			if err != nil {
				fmt.Printf("[Aviso] Falha ao renovar TTL no Redis para %s\n", userID.String())
			}
		}
	}
}

// Auxiliares  (Só para não embolar)

func toDomainStatus(protoStatus chatMessage.MessageStatus) domain.MessageStatus {

	name, thereIs := chatMessage.MessageStatus_name[int32(protoStatus)]
	if !thereIs {
		return domain.MessageStatus("SENT")
	}
	return domain.MessageStatus(name)
}

func toProtoStatus(domainStatus domain.MessageStatus) chatMessage.MessageStatus {
	valueInt, thereIs := chatMessage.MessageStatus_value[string(domainStatus)]
	if !thereIs {
		return chatMessage.MessageStatus_SENT
	}
	return chatMessage.MessageStatus(valueInt)
}
