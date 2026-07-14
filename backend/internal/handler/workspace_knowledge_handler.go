package handler

import (
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type WorkspaceHandler struct{ workspaceService *service.WorkspaceService }

func NewWorkspaceHandler(workspaceService *service.WorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{workspaceService: workspaceService}
}

type KnowledgeHandler struct{ knowledgeService *service.KnowledgeService }

func NewKnowledgeHandler(knowledgeService *service.KnowledgeService) *KnowledgeHandler {
	return &KnowledgeHandler{knowledgeService: knowledgeService}
}

type workspaceDTO struct {
	ID          int64          `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Metadata    map[string]any `json:"metadata"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type knowledgeFileDTO struct {
	ID          int64          `json:"id"`
	WorkspaceID *int64         `json:"workspace_id,omitempty"`
	Filename    string         `json:"filename"`
	FileType    string         `json:"file_type"`
	FileSize    int64          `json:"file_size"`
	Content     string         `json:"content,omitempty"`
	ContentHash string         `json:"content_hash,omitempty"`
	Metadata    map[string]any `json:"metadata"`
	Status      string         `json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type createWorkspaceRequest struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Metadata    map[string]any `json:"metadata"`
}
type updateWorkspaceRequest struct {
	Name        *string        `json:"name"`
	Description *string        `json:"description"`
	Metadata    map[string]any `json:"metadata"`
}
type createKnowledgeFileRequest struct {
	WorkspaceID *int64         `json:"workspace_id"`
	Filename    string         `json:"filename"`
	FileType    string         `json:"file_type"`
	Content     string         `json:"content" binding:"required"`
	Metadata    map[string]any `json:"metadata"`
}
type updateKnowledgeFileRequest struct {
	WorkspaceID *int64         `json:"workspace_id"`
	Filename    *string        `json:"filename"`
	FileType    *string        `json:"file_type"`
	Content     *string        `json:"content"`
	Status      *string        `json:"status"`
	Metadata    map[string]any `json:"metadata"`
}

func (h *WorkspaceHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, result, err := h.workspaceService.ListWorkspaces(c.Request.Context(), subject.UserID, servicePagination(page, pageSize))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]workspaceDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toWorkspaceDTO(item))
	}
	response.Paginated(c, out, result.Total, result.Page, result.PageSize)
}

func (h *WorkspaceHandler) Get(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	id, ok := parseIDParam(c, "workspaceId")
	if !ok {
		return
	}
	item, err := h.workspaceService.GetWorkspace(c.Request.Context(), subject.UserID, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, toWorkspaceDTO(*item))
}

func (h *WorkspaceHandler) Create(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req createWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.workspaceService.CreateWorkspace(c.Request.Context(), subject.UserID, service.CreateWorkspaceInput{Name: req.Name, Description: req.Description, Metadata: req.Metadata})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, toWorkspaceDTO(*item))
}

func (h *WorkspaceHandler) Update(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	id, ok := parseIDParam(c, "workspaceId")
	if !ok {
		return
	}
	var req updateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.workspaceService.UpdateWorkspace(c.Request.Context(), subject.UserID, id, service.UpdateWorkspaceInput{Name: req.Name, Description: req.Description, Metadata: req.Metadata})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, toWorkspaceDTO(*item))
}

func (h *WorkspaceHandler) Delete(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	id, ok := parseIDParam(c, "workspaceId")
	if !ok {
		return
	}
	if err := h.workspaceService.DeleteWorkspace(c.Request.Context(), subject.UserID, id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *KnowledgeHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	page, pageSize := response.ParsePagination(c)
	workspaceID, ok := optionalInt64Query(c, "workspace_id")
	if !ok {
		return
	}
	items, result, err := h.knowledgeService.ListKnowledgeFiles(c.Request.Context(), subject.UserID, workspaceID, servicePagination(page, pageSize))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]knowledgeFileDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toKnowledgeFileDTO(item, false))
	}
	response.Paginated(c, out, result.Total, result.Page, result.PageSize)
}

func (h *KnowledgeHandler) Get(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	id, ok := parseIDParam(c, "fileId")
	if !ok {
		return
	}
	item, err := h.knowledgeService.GetKnowledgeFile(c.Request.Context(), subject.UserID, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, toKnowledgeFileDTO(*item, true))
}

func (h *KnowledgeHandler) Create(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req createKnowledgeFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.knowledgeService.CreateKnowledgeFile(c.Request.Context(), subject.UserID, service.CreateKnowledgeFileInput{WorkspaceID: req.WorkspaceID, Filename: req.Filename, FileType: req.FileType, Content: req.Content, Metadata: req.Metadata})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, toKnowledgeFileDTO(*item, true))
}

func (h *KnowledgeHandler) Update(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	id, ok := parseIDParam(c, "fileId")
	if !ok {
		return
	}
	var req updateKnowledgeFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.knowledgeService.UpdateKnowledgeFile(c.Request.Context(), subject.UserID, id, service.UpdateKnowledgeFileInput{WorkspaceID: req.WorkspaceID, Filename: req.Filename, FileType: req.FileType, Content: req.Content, Status: req.Status, Metadata: req.Metadata})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, toKnowledgeFileDTO(*item, true))
}

func (h *KnowledgeHandler) Delete(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	id, ok := parseIDParam(c, "fileId")
	if !ok {
		return
	}
	if err := h.knowledgeService.DeleteKnowledgeFile(c.Request.Context(), subject.UserID, id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func optionalInt64Query(c *gin.Context, name string) (*int64, bool) {
	raw := c.Query(name)
	if raw == "" {
		return nil, true
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid ID")
		return nil, false
	}
	return &id, true
}
func toWorkspaceDTO(in service.Workspace) workspaceDTO {
	return workspaceDTO{ID: in.ID, Name: in.Name, Description: in.Description, Metadata: in.Metadata, CreatedAt: in.CreatedAt, UpdatedAt: in.UpdatedAt}
}
func toKnowledgeFileDTO(in service.KnowledgeFile, includeContent bool) knowledgeFileDTO {
	content := ""
	if includeContent {
		content = in.Content
	}
	return knowledgeFileDTO{ID: in.ID, WorkspaceID: in.WorkspaceID, Filename: in.Filename, FileType: in.FileType, FileSize: in.FileSize, Content: content, ContentHash: in.ContentHash, Metadata: in.Metadata, Status: in.Status, CreatedAt: in.CreatedAt, UpdatedAt: in.UpdatedAt}
}
