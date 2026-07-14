package handler

import (
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	chatService *service.ChatService
}

func NewChatHandler(chatService *service.ChatService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

type chatConversationDTO struct {
	ID           int64            `json:"id"`
	Title        string           `json:"title"`
	KnowledgeIDs []int64          `json:"knowledge_ids"`
	Metadata     map[string]any   `json:"metadata"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
	Messages     []chatMessageDTO `json:"messages,omitempty"`
}

type chatMessageDTO struct {
	ID             int64            `json:"id"`
	ConversationID int64            `json:"conversation_id"`
	Role           string           `json:"role"`
	Content        string           `json:"content"`
	Attachments    []map[string]any `json:"attachments"`
	TokensUsed     int              `json:"tokens_used"`
	Cost           float64          `json:"cost"`
	Model          string           `json:"model,omitempty"`
	Metadata       map[string]any   `json:"metadata"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

type createChatConversationRequest struct {
	Title        string         `json:"title"`
	KnowledgeIDs []int64        `json:"knowledge_ids"`
	Metadata     map[string]any `json:"metadata"`
}

type updateChatConversationRequest struct {
	Title        *string        `json:"title"`
	KnowledgeIDs []int64        `json:"knowledge_ids"`
	Metadata     map[string]any `json:"metadata"`
}

type createChatMessageRequest struct {
	Role        string           `json:"role"`
	Content     string           `json:"content" binding:"required"`
	Attachments []map[string]any `json:"attachments"`
	TokensUsed  int              `json:"tokens_used"`
	Cost        float64          `json:"cost"`
	Model       string           `json:"model"`
	Metadata    map[string]any   `json:"metadata"`
}

func (h *ChatHandler) ListConversations(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, result, err := h.chatService.ListConversations(c.Request.Context(), subject.UserID, servicePagination(page, pageSize))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]chatConversationDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toChatConversationDTO(item, false))
	}
	response.Paginated(c, out, result.Total, result.Page, result.PageSize)
}

func (h *ChatHandler) GetConversation(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	conversationID, ok := parseIDParam(c, "conversationId")
	if !ok {
		return
	}
	conversation, err := h.chatService.GetConversation(c.Request.Context(), subject.UserID, conversationID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, toChatConversationDTO(*conversation, true))
}

func (h *ChatHandler) CreateConversation(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req createChatConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	conversation, err := h.chatService.CreateConversation(c.Request.Context(), subject.UserID, service.CreateChatConversationInput{
		Title:        req.Title,
		KnowledgeIDs: req.KnowledgeIDs,
		Metadata:     req.Metadata,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, toChatConversationDTO(*conversation, false))
}

func (h *ChatHandler) UpdateConversation(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	conversationID, ok := parseIDParam(c, "conversationId")
	if !ok {
		return
	}
	var req updateChatConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	conversation, err := h.chatService.UpdateConversation(c.Request.Context(), subject.UserID, conversationID, service.UpdateChatConversationInput{
		Title:        req.Title,
		KnowledgeIDs: req.KnowledgeIDs,
		Metadata:     req.Metadata,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, toChatConversationDTO(*conversation, false))
}

func (h *ChatHandler) DeleteConversation(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	conversationID, ok := parseIDParam(c, "conversationId")
	if !ok {
		return
	}
	if err := h.chatService.DeleteConversation(c.Request.Context(), subject.UserID, conversationID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *ChatHandler) CreateMessage(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	conversationID, ok := parseIDParam(c, "conversationId")
	if !ok {
		return
	}
	var req createChatMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	message, err := h.chatService.CreateMessage(c.Request.Context(), subject.UserID, conversationID, service.CreateChatMessageInput{
		Role:        req.Role,
		Content:     req.Content,
		Attachments: req.Attachments,
		TokensUsed:  req.TokensUsed,
		Cost:        req.Cost,
		Model:       req.Model,
		Metadata:    req.Metadata,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, toChatMessageDTO(*message))
}

func parseIDParam(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid ID")
		return 0, false
	}
	return id, true
}

func servicePagination(page, pageSize int) pagination.PaginationParams {
	return pagination.PaginationParams{Page: page, PageSize: pageSize, SortOrder: pagination.SortOrderDesc}
}

func toChatConversationDTO(in service.ChatConversation, includeMessages bool) chatConversationDTO {
	out := chatConversationDTO{
		ID:           in.ID,
		Title:        in.Title,
		KnowledgeIDs: in.KnowledgeIDs,
		Metadata:     in.Metadata,
		CreatedAt:    in.CreatedAt,
		UpdatedAt:    in.UpdatedAt,
	}
	if includeMessages {
		out.Messages = make([]chatMessageDTO, 0, len(in.Messages))
		for _, msg := range in.Messages {
			out.Messages = append(out.Messages, toChatMessageDTO(msg))
		}
	}
	return out
}

func toChatMessageDTO(in service.ChatMessage) chatMessageDTO {
	return chatMessageDTO{
		ID:             in.ID,
		ConversationID: in.ConversationID,
		Role:           in.Role,
		Content:        in.Content,
		Attachments:    in.Attachments,
		TokensUsed:     in.TokensUsed,
		Cost:           in.Cost,
		Model:          in.Model,
		Metadata:       in.Metadata,
		CreatedAt:      in.CreatedAt,
		UpdatedAt:      in.UpdatedAt,
	}
}
