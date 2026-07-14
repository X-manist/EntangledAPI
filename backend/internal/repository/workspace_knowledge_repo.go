package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/knowledgefile"
	"github.com/Wei-Shaw/sub2api/ent/predicate"
	"github.com/Wei-Shaw/sub2api/ent/workspace"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type WorkspaceRepository struct{ client *ent.Client }

func NewWorkspaceRepository(client *ent.Client) *WorkspaceRepository {
	return &WorkspaceRepository{client: client}
}

func (r *WorkspaceRepository) ListWorkspaces(ctx context.Context, userID int64, params pagination.PaginationParams) ([]service.Workspace, *pagination.PaginationResult, error) {
	page, pageSize := normalizeRepoPagination(params)
	base := r.client.Workspace.Query().Where(workspace.UserIDEQ(userID), workspace.DeletedAtIsNil())
	total, err := base.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}
	rows, err := base.Order(workspace.ByUpdatedAt(entsql.OrderDesc())).Limit(pageSize).Offset((page - 1) * pageSize).All(ctx)
	if err != nil {
		return nil, nil, err
	}
	items := make([]service.Workspace, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapWorkspace(row))
	}
	return items, repoPaginationResult(total, page, pageSize), nil
}

func (r *WorkspaceRepository) GetWorkspace(ctx context.Context, userID, workspaceID int64) (*service.Workspace, error) {
	row, err := r.client.Workspace.Query().Where(workspace.IDEQ(workspaceID), workspace.UserIDEQ(userID), workspace.DeletedAtIsNil()).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, infraerrors.NotFound("WORKSPACE_NOT_FOUND", "workspace not found")
		}
		return nil, err
	}
	out := mapWorkspace(row)
	return &out, nil
}

func (r *WorkspaceRepository) CreateWorkspace(ctx context.Context, userID int64, input service.CreateWorkspaceInput) (*service.Workspace, error) {
	builder := r.client.Workspace.Create().SetUserID(userID).SetName(input.Name).SetDescription(input.Description)
	if input.Metadata != nil {
		builder.SetMetadata(input.Metadata)
	}
	row, err := builder.Save(ctx)
	if err != nil {
		return nil, err
	}
	out := mapWorkspace(row)
	return &out, nil
}

func (r *WorkspaceRepository) UpdateWorkspace(ctx context.Context, userID, workspaceID int64, input service.UpdateWorkspaceInput) (*service.Workspace, error) {
	if _, err := r.GetWorkspace(ctx, userID, workspaceID); err != nil {
		return nil, err
	}
	builder := r.client.Workspace.UpdateOneID(workspaceID)
	if input.Name != nil {
		builder.SetName(*input.Name)
	}
	if input.Description != nil {
		builder.SetDescription(*input.Description)
	}
	if input.Metadata != nil {
		builder.SetMetadata(input.Metadata)
	}
	row, err := builder.Save(ctx)
	if err != nil {
		return nil, err
	}
	out := mapWorkspace(row)
	return &out, nil
}

func (r *WorkspaceRepository) DeleteWorkspace(ctx context.Context, userID, workspaceID int64) error {
	if _, err := r.GetWorkspace(ctx, userID, workspaceID); err != nil {
		return err
	}
	now := time.Now().UTC()
	return r.client.Workspace.UpdateOneID(workspaceID).SetDeletedAt(now).Exec(ctx)
}

type KnowledgeRepository struct{ client *ent.Client }

func NewKnowledgeRepository(client *ent.Client) *KnowledgeRepository {
	return &KnowledgeRepository{client: client}
}

func (r *KnowledgeRepository) ListKnowledgeFiles(ctx context.Context, userID int64, workspaceID *int64, params pagination.PaginationParams) ([]service.KnowledgeFile, *pagination.PaginationResult, error) {
	page, pageSize := normalizeRepoPagination(params)
	predicates := []predicate.KnowledgeFile{knowledgefile.UserIDEQ(userID), knowledgefile.DeletedAtIsNil()}
	if workspaceID != nil {
		predicates = append(predicates, knowledgefile.WorkspaceIDEQ(*workspaceID))
	}
	base := r.client.KnowledgeFile.Query().Where(predicates...)
	total, err := base.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}
	rows, err := base.Order(knowledgefile.ByCreatedAt(entsql.OrderDesc())).Limit(pageSize).Offset((page - 1) * pageSize).All(ctx)
	if err != nil {
		return nil, nil, err
	}
	items := make([]service.KnowledgeFile, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapKnowledgeFile(row))
	}
	return items, repoPaginationResult(total, page, pageSize), nil
}

func (r *KnowledgeRepository) GetKnowledgeFile(ctx context.Context, userID, fileID int64) (*service.KnowledgeFile, error) {
	row, err := r.client.KnowledgeFile.Query().Where(knowledgefile.IDEQ(fileID), knowledgefile.UserIDEQ(userID), knowledgefile.DeletedAtIsNil()).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, infraerrors.NotFound("KNOWLEDGE_FILE_NOT_FOUND", "knowledge file not found")
		}
		return nil, err
	}
	out := mapKnowledgeFile(row)
	return &out, nil
}

func (r *KnowledgeRepository) CreateKnowledgeFile(ctx context.Context, userID int64, input service.CreateKnowledgeFileInput) (*service.KnowledgeFile, error) {
	hash := contentHash(input.Content)
	builder := r.client.KnowledgeFile.Create().SetUserID(userID).SetNillableWorkspaceID(input.WorkspaceID).SetFilename(input.Filename).SetFileType(input.FileType).SetFileSize(int64(len([]byte(input.Content)))).SetStoragePath(fmt.Sprintf("knowledge://user/%d/%s", userID, hash)).SetContent(input.Content).SetContentHash(hash).SetStatus(service.KnowledgeFileStatusReady)
	if input.Metadata != nil {
		builder.SetMetadata(input.Metadata)
	}
	row, err := builder.Save(ctx)
	if err != nil {
		return nil, err
	}
	out := mapKnowledgeFile(row)
	return &out, nil
}

func (r *KnowledgeRepository) UpdateKnowledgeFile(ctx context.Context, userID, fileID int64, input service.UpdateKnowledgeFileInput) (*service.KnowledgeFile, error) {
	if _, err := r.GetKnowledgeFile(ctx, userID, fileID); err != nil {
		return nil, err
	}
	builder := r.client.KnowledgeFile.UpdateOneID(fileID)
	if input.WorkspaceID != nil {
		builder.SetWorkspaceID(*input.WorkspaceID)
	}
	if input.Filename != nil {
		builder.SetFilename(*input.Filename)
	}
	if input.FileType != nil {
		builder.SetFileType(*input.FileType)
	}
	if input.Content != nil {
		hash := contentHash(*input.Content)
		builder.SetContent(*input.Content).SetFileSize(int64(len([]byte(*input.Content)))).SetContentHash(hash).SetStoragePath(fmt.Sprintf("knowledge://user/%d/%s", userID, hash))
	}
	if input.Status != nil {
		builder.SetStatus(*input.Status)
	}
	if input.Metadata != nil {
		builder.SetMetadata(input.Metadata)
	}
	row, err := builder.Save(ctx)
	if err != nil {
		return nil, err
	}
	out := mapKnowledgeFile(row)
	return &out, nil
}

func (r *KnowledgeRepository) DeleteKnowledgeFile(ctx context.Context, userID, fileID int64) error {
	if _, err := r.GetKnowledgeFile(ctx, userID, fileID); err != nil {
		return err
	}
	now := time.Now().UTC()
	return r.client.KnowledgeFile.UpdateOneID(fileID).SetDeletedAt(now).Exec(ctx)
}

func normalizeRepoPagination(params pagination.PaginationParams) (int, int) {
	page, pageSize := params.Page, params.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func repoPaginationResult(total, page, pageSize int) *pagination.PaginationResult {
	return &pagination.PaginationResult{Total: int64(total), Page: page, PageSize: pageSize, Pages: (total + pageSize - 1) / pageSize}
}

func mapWorkspace(row *ent.Workspace) service.Workspace {
	if row.Metadata == nil {
		row.Metadata = map[string]any{}
	}
	return service.Workspace{ID: row.ID, UserID: row.UserID, Name: row.Name, Description: row.Description, Metadata: row.Metadata, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func mapKnowledgeFile(row *ent.KnowledgeFile) service.KnowledgeFile {
	if row.Metadata == nil {
		row.Metadata = map[string]any{}
	}
	return service.KnowledgeFile{ID: row.ID, UserID: row.UserID, WorkspaceID: row.WorkspaceID, Filename: row.Filename, FileType: row.FileType, FileSize: row.FileSize, StoragePath: row.StoragePath, Content: row.Content, ContentHash: row.ContentHash, Metadata: row.Metadata, Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func contentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}
