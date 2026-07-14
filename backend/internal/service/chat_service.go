package service

import (
	"context"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	ChatRoleUser      = "user"
	ChatRoleAssistant = "assistant"
	ChatRoleSystem    = "system"
)

type ChatConversation struct {
	ID           int64
	UserID       int64
	Title        string
	KnowledgeIDs []int64
	Metadata     map[string]any
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Messages     []ChatMessage
}

type ChatMessage struct {
	ID             int64
	ConversationID int64
	Role           string
	Content        string
	Attachments    []map[string]any
	TokensUsed     int
	Cost           float64
	Model          string
	Metadata       map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CreateChatConversationInput struct {
	Title        string
	KnowledgeIDs []int64
	Metadata     map[string]any
}

type UpdateChatConversationInput struct {
	Title        *string
	KnowledgeIDs []int64
	Metadata     map[string]any
}

type CreateChatMessageInput struct {
	Role        string
	Content     string
	Attachments []map[string]any
	TokensUsed  int
	Cost        float64
	Model       string
	Metadata    map[string]any
}

type ChatRepository interface {
	ListConversations(ctx context.Context, userID int64, params pagination.PaginationParams) ([]ChatConversation, *pagination.PaginationResult, error)
	GetConversation(ctx context.Context, userID, conversationID int64) (*ChatConversation, error)
	CreateConversation(ctx context.Context, userID int64, input CreateChatConversationInput) (*ChatConversation, error)
	UpdateConversation(ctx context.Context, userID, conversationID int64, input UpdateChatConversationInput) (*ChatConversation, error)
	DeleteConversation(ctx context.Context, userID, conversationID int64) error
	CreateMessage(ctx context.Context, userID, conversationID int64, input CreateChatMessageInput) (*ChatMessage, error)
}

type ChatService struct {
	repo ChatRepository
}

func NewChatService(repo ChatRepository) *ChatService {
	return &ChatService{repo: repo}
}

func (s *ChatService) ListConversations(ctx context.Context, userID int64, params pagination.PaginationParams) ([]ChatConversation, *pagination.PaginationResult, error) {
	return s.repo.ListConversations(ctx, userID, params)
}

func (s *ChatService) GetConversation(ctx context.Context, userID, conversationID int64) (*ChatConversation, error) {
	if conversationID <= 0 {
		return nil, infraerrors.BadRequest("CHAT_INVALID_CONVERSATION_ID", "invalid conversation id")
	}
	return s.repo.GetConversation(ctx, userID, conversationID)
}

func (s *ChatService) CreateConversation(ctx context.Context, userID int64, input CreateChatConversationInput) (*ChatConversation, error) {
	input.Title = normalizeChatTitle(input.Title)
	return s.repo.CreateConversation(ctx, userID, input)
}

func (s *ChatService) UpdateConversation(ctx context.Context, userID, conversationID int64, input UpdateChatConversationInput) (*ChatConversation, error) {
	if conversationID <= 0 {
		return nil, infraerrors.BadRequest("CHAT_INVALID_CONVERSATION_ID", "invalid conversation id")
	}
	if input.Title != nil {
		title := normalizeChatTitle(*input.Title)
		input.Title = &title
	}
	return s.repo.UpdateConversation(ctx, userID, conversationID, input)
}

func (s *ChatService) DeleteConversation(ctx context.Context, userID, conversationID int64) error {
	if conversationID <= 0 {
		return infraerrors.BadRequest("CHAT_INVALID_CONVERSATION_ID", "invalid conversation id")
	}
	return s.repo.DeleteConversation(ctx, userID, conversationID)
}

func (s *ChatService) CreateMessage(ctx context.Context, userID, conversationID int64, input CreateChatMessageInput) (*ChatMessage, error) {
	if conversationID <= 0 {
		return nil, infraerrors.BadRequest("CHAT_INVALID_CONVERSATION_ID", "invalid conversation id")
	}
	input.Role = strings.TrimSpace(input.Role)
	if input.Role == "" {
		input.Role = ChatRoleUser
	}
	if input.Role != ChatRoleUser && input.Role != ChatRoleAssistant && input.Role != ChatRoleSystem {
		return nil, infraerrors.BadRequest("CHAT_INVALID_MESSAGE_ROLE", "invalid message role")
	}
	input.Content = strings.TrimSpace(input.Content)
	if input.Content == "" {
		return nil, infraerrors.BadRequest("CHAT_EMPTY_MESSAGE", "message content cannot be empty")
	}
	return s.repo.CreateMessage(ctx, userID, conversationID, input)
}

func normalizeChatTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return "新对话"
	}
	if len([]rune(title)) > 120 {
		return string([]rune(title)[:120])
	}
	return title
}
