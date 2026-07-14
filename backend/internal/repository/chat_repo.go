package repository

import (
	"context"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/conversation"
	"github.com/Wei-Shaw/sub2api/ent/message"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type ChatRepository struct {
	client *ent.Client
}

func NewChatRepository(client *ent.Client) *ChatRepository {
	return &ChatRepository{client: client}
}

func (r *ChatRepository) ListConversations(ctx context.Context, userID int64, params pagination.PaginationParams) ([]service.ChatConversation, *pagination.PaginationResult, error) {
	page := params.Page
	if page <= 0 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	base := r.client.Conversation.Query().
		Where(conversation.UserIDEQ(userID), conversation.DeletedAtIsNil())
	total, err := base.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	rows, err := base.
		Order(conversation.ByUpdatedAt(entsql.OrderDesc())).
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	items := make([]service.ChatConversation, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapConversation(row, nil))
	}
	return items, &pagination.PaginationResult{
		Total:    int64(total),
		Page:     page,
		PageSize: pageSize,
		Pages:    (total + pageSize - 1) / pageSize,
	}, nil
}

func (r *ChatRepository) GetConversation(ctx context.Context, userID, conversationID int64) (*service.ChatConversation, error) {
	row, err := r.client.Conversation.Query().
		Where(conversation.IDEQ(conversationID), conversation.UserIDEQ(userID), conversation.DeletedAtIsNil()).
		WithMessages(func(q *ent.MessageQuery) {
			q.Order(message.ByCreatedAt())
		}).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, infraerrors.NotFound("CHAT_CONVERSATION_NOT_FOUND", "conversation not found")
		}
		return nil, err
	}
	messages := make([]service.ChatMessage, 0, len(row.Edges.Messages))
	for _, msg := range row.Edges.Messages {
		messages = append(messages, mapMessage(msg))
	}
	out := mapConversation(row, messages)
	return &out, nil
}

func (r *ChatRepository) CreateConversation(ctx context.Context, userID int64, input service.CreateChatConversationInput) (*service.ChatConversation, error) {
	builder := r.client.Conversation.Create().
		SetUserID(userID).
		SetTitle(input.Title).
		SetKnowledgeIds(input.KnowledgeIDs)
	if input.Metadata != nil {
		builder.SetMetadata(input.Metadata)
	}
	row, err := builder.Save(ctx)
	if err != nil {
		return nil, err
	}
	out := mapConversation(row, nil)
	return &out, nil
}

func (r *ChatRepository) UpdateConversation(ctx context.Context, userID, conversationID int64, input service.UpdateChatConversationInput) (*service.ChatConversation, error) {
	if _, err := r.GetConversation(ctx, userID, conversationID); err != nil {
		return nil, err
	}
	builder := r.client.Conversation.UpdateOneID(conversationID)
	if input.Title != nil {
		builder.SetTitle(*input.Title)
	}
	if input.KnowledgeIDs != nil {
		builder.SetKnowledgeIds(input.KnowledgeIDs)
	}
	if input.Metadata != nil {
		builder.SetMetadata(input.Metadata)
	}
	row, err := builder.Save(ctx)
	if err != nil {
		return nil, err
	}
	out := mapConversation(row, nil)
	return &out, nil
}

func (r *ChatRepository) DeleteConversation(ctx context.Context, userID, conversationID int64) error {
	if _, err := r.GetConversation(ctx, userID, conversationID); err != nil {
		return err
	}
	now := time.Now().UTC()
	return r.client.Conversation.UpdateOneID(conversationID).SetDeletedAt(now).Exec(ctx)
}

func (r *ChatRepository) CreateMessage(ctx context.Context, userID, conversationID int64, input service.CreateChatMessageInput) (*service.ChatMessage, error) {
	if _, err := r.GetConversation(ctx, userID, conversationID); err != nil {
		return nil, err
	}
	builder := r.client.Message.Create().
		SetConversationID(conversationID).
		SetRole(input.Role).
		SetContent(input.Content).
		SetAttachments(input.Attachments).
		SetTokensUsed(input.TokensUsed).
		SetCost(input.Cost)
	if input.Model != "" {
		builder.SetModel(input.Model)
	}
	if input.Metadata != nil {
		builder.SetMetadata(input.Metadata)
	}
	row, err := builder.Save(ctx)
	if err != nil {
		return nil, err
	}
	_, _ = r.client.Conversation.UpdateOneID(conversationID).SetUpdatedAt(time.Now().UTC()).Save(ctx)
	out := mapMessage(row)
	return &out, nil
}

func mapConversation(row *ent.Conversation, messages []service.ChatMessage) service.ChatConversation {
	if row.Metadata == nil {
		row.Metadata = map[string]any{}
	}
	return service.ChatConversation{
		ID:           row.ID,
		UserID:       row.UserID,
		Title:        row.Title,
		KnowledgeIDs: row.KnowledgeIds,
		Metadata:     row.Metadata,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
		Messages:     messages,
	}
}

func mapMessage(row *ent.Message) service.ChatMessage {
	if row.Metadata == nil {
		row.Metadata = map[string]any{}
	}
	return service.ChatMessage{
		ID:             row.ID,
		ConversationID: row.ConversationID,
		Role:           row.Role,
		Content:        row.Content,
		Attachments:    row.Attachments,
		TokensUsed:     row.TokensUsed,
		Cost:           row.Cost,
		Model:          row.Model,
		Metadata:       row.Metadata,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}
