package service

import (
	"context"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	KnowledgeFileStatusUploaded   = "uploaded"
	KnowledgeFileStatusProcessing = "processing"
	KnowledgeFileStatusReady      = "ready"
	KnowledgeFileStatusFailed     = "failed"
)

type Workspace struct {
	ID          int64
	UserID      int64
	Name        string
	Description string
	Metadata    map[string]any
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type KnowledgeFile struct {
	ID          int64
	UserID      int64
	WorkspaceID *int64
	Filename    string
	FileType    string
	FileSize    int64
	StoragePath string
	Content     string
	ContentHash string
	Metadata    map[string]any
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CreateWorkspaceInput struct {
	Name        string
	Description string
	Metadata    map[string]any
}

type UpdateWorkspaceInput struct {
	Name        *string
	Description *string
	Metadata    map[string]any
}

type CreateKnowledgeFileInput struct {
	WorkspaceID *int64
	Filename    string
	FileType    string
	Content     string
	Metadata    map[string]any
}

type UpdateKnowledgeFileInput struct {
	WorkspaceID *int64
	Filename    *string
	FileType    *string
	Content     *string
	Status      *string
	Metadata    map[string]any
}

type WorkspaceRepository interface {
	ListWorkspaces(ctx context.Context, userID int64, params pagination.PaginationParams) ([]Workspace, *pagination.PaginationResult, error)
	GetWorkspace(ctx context.Context, userID, workspaceID int64) (*Workspace, error)
	CreateWorkspace(ctx context.Context, userID int64, input CreateWorkspaceInput) (*Workspace, error)
	UpdateWorkspace(ctx context.Context, userID, workspaceID int64, input UpdateWorkspaceInput) (*Workspace, error)
	DeleteWorkspace(ctx context.Context, userID, workspaceID int64) error
}

type KnowledgeRepository interface {
	ListKnowledgeFiles(ctx context.Context, userID int64, workspaceID *int64, params pagination.PaginationParams) ([]KnowledgeFile, *pagination.PaginationResult, error)
	GetKnowledgeFile(ctx context.Context, userID, fileID int64) (*KnowledgeFile, error)
	CreateKnowledgeFile(ctx context.Context, userID int64, input CreateKnowledgeFileInput) (*KnowledgeFile, error)
	UpdateKnowledgeFile(ctx context.Context, userID, fileID int64, input UpdateKnowledgeFileInput) (*KnowledgeFile, error)
	DeleteKnowledgeFile(ctx context.Context, userID, fileID int64) error
}

type WorkspaceService struct {
	repo WorkspaceRepository
}

func NewWorkspaceService(repo WorkspaceRepository) *WorkspaceService {
	return &WorkspaceService{repo: repo}
}

func (s *WorkspaceService) ListWorkspaces(ctx context.Context, userID int64, params pagination.PaginationParams) ([]Workspace, *pagination.PaginationResult, error) {
	return s.repo.ListWorkspaces(ctx, userID, params)
}

func (s *WorkspaceService) GetWorkspace(ctx context.Context, userID, workspaceID int64) (*Workspace, error) {
	if workspaceID <= 0 {
		return nil, infraerrors.BadRequest("WORKSPACE_INVALID_ID", "invalid workspace id")
	}
	return s.repo.GetWorkspace(ctx, userID, workspaceID)
}

func (s *WorkspaceService) CreateWorkspace(ctx context.Context, userID int64, input CreateWorkspaceInput) (*Workspace, error) {
	input.Name = normalizeRequiredName(input.Name, "默认工作空间")
	input.Description = strings.TrimSpace(input.Description)
	return s.repo.CreateWorkspace(ctx, userID, input)
}

func (s *WorkspaceService) UpdateWorkspace(ctx context.Context, userID, workspaceID int64, input UpdateWorkspaceInput) (*Workspace, error) {
	if workspaceID <= 0 {
		return nil, infraerrors.BadRequest("WORKSPACE_INVALID_ID", "invalid workspace id")
	}
	if input.Name != nil {
		name := normalizeRequiredName(*input.Name, "默认工作空间")
		input.Name = &name
	}
	if input.Description != nil {
		description := strings.TrimSpace(*input.Description)
		input.Description = &description
	}
	return s.repo.UpdateWorkspace(ctx, userID, workspaceID, input)
}

func (s *WorkspaceService) DeleteWorkspace(ctx context.Context, userID, workspaceID int64) error {
	if workspaceID <= 0 {
		return infraerrors.BadRequest("WORKSPACE_INVALID_ID", "invalid workspace id")
	}
	return s.repo.DeleteWorkspace(ctx, userID, workspaceID)
}

type KnowledgeService struct {
	repo          KnowledgeRepository
	workspaceRepo WorkspaceRepository
}

func NewKnowledgeService(repo KnowledgeRepository, workspaceRepo WorkspaceRepository) *KnowledgeService {
	return &KnowledgeService{repo: repo, workspaceRepo: workspaceRepo}
}

func (s *KnowledgeService) ListKnowledgeFiles(ctx context.Context, userID int64, workspaceID *int64, params pagination.PaginationParams) ([]KnowledgeFile, *pagination.PaginationResult, error) {
	if workspaceID != nil {
		if *workspaceID <= 0 {
			return nil, nil, infraerrors.BadRequest("WORKSPACE_INVALID_ID", "invalid workspace id")
		}
		if _, err := s.workspaceRepo.GetWorkspace(ctx, userID, *workspaceID); err != nil {
			return nil, nil, err
		}
	}
	return s.repo.ListKnowledgeFiles(ctx, userID, workspaceID, params)
}

func (s *KnowledgeService) GetKnowledgeFile(ctx context.Context, userID, fileID int64) (*KnowledgeFile, error) {
	if fileID <= 0 {
		return nil, infraerrors.BadRequest("KNOWLEDGE_FILE_INVALID_ID", "invalid knowledge file id")
	}
	return s.repo.GetKnowledgeFile(ctx, userID, fileID)
}

func (s *KnowledgeService) CreateKnowledgeFile(ctx context.Context, userID int64, input CreateKnowledgeFileInput) (*KnowledgeFile, error) {
	if input.WorkspaceID != nil {
		if _, err := s.workspaceRepo.GetWorkspace(ctx, userID, *input.WorkspaceID); err != nil {
			return nil, err
		}
	}
	input.Filename = normalizeRequiredName(input.Filename, "未命名知识文件")
	input.FileType = normalizeFileType(input.FileType)
	input.Content = strings.TrimSpace(input.Content)
	if input.Content == "" {
		return nil, infraerrors.BadRequest("KNOWLEDGE_FILE_EMPTY_CONTENT", "knowledge file content cannot be empty")
	}
	return s.repo.CreateKnowledgeFile(ctx, userID, input)
}

func (s *KnowledgeService) UpdateKnowledgeFile(ctx context.Context, userID, fileID int64, input UpdateKnowledgeFileInput) (*KnowledgeFile, error) {
	if fileID <= 0 {
		return nil, infraerrors.BadRequest("KNOWLEDGE_FILE_INVALID_ID", "invalid knowledge file id")
	}
	if input.WorkspaceID != nil {
		if _, err := s.workspaceRepo.GetWorkspace(ctx, userID, *input.WorkspaceID); err != nil {
			return nil, err
		}
	}
	if input.Filename != nil {
		filename := normalizeRequiredName(*input.Filename, "未命名知识文件")
		input.Filename = &filename
	}
	if input.FileType != nil {
		fileType := normalizeFileType(*input.FileType)
		input.FileType = &fileType
	}
	if input.Content != nil {
		content := strings.TrimSpace(*input.Content)
		if content == "" {
			return nil, infraerrors.BadRequest("KNOWLEDGE_FILE_EMPTY_CONTENT", "knowledge file content cannot be empty")
		}
		input.Content = &content
	}
	if input.Status != nil {
		status := strings.TrimSpace(*input.Status)
		if status != KnowledgeFileStatusUploaded && status != KnowledgeFileStatusProcessing && status != KnowledgeFileStatusReady && status != KnowledgeFileStatusFailed {
			return nil, infraerrors.BadRequest("KNOWLEDGE_FILE_INVALID_STATUS", "invalid knowledge file status")
		}
		input.Status = &status
	}
	return s.repo.UpdateKnowledgeFile(ctx, userID, fileID, input)
}

func (s *KnowledgeService) DeleteKnowledgeFile(ctx context.Context, userID, fileID int64) error {
	if fileID <= 0 {
		return infraerrors.BadRequest("KNOWLEDGE_FILE_INVALID_ID", "invalid knowledge file id")
	}
	return s.repo.DeleteKnowledgeFile(ctx, userID, fileID)
}

func normalizeRequiredName(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	if len([]rune(value)) > 120 {
		return string([]rune(value)[:120])
	}
	return value
}

func normalizeFileType(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "text"
	}
	if len([]rune(value)) > 50 {
		return string([]rune(value)[:50])
	}
	return value
}
