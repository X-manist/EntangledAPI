-- Workspaces and knowledge-base ownership wiring.

CREATE TABLE IF NOT EXISTS workspaces (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(120) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_workspaces_user_updated_at
    ON workspaces(user_id, updated_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_workspaces_user_deleted_at
    ON workspaces(user_id, deleted_at);

ALTER TABLE conversations
    ADD COLUMN IF NOT EXISTS workspace_id BIGINT NULL REFERENCES workspaces(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_conversations_workspace_updated_at
    ON conversations(workspace_id, updated_at DESC)
    WHERE deleted_at IS NULL AND workspace_id IS NOT NULL;

ALTER TABLE knowledge_files
    ADD COLUMN IF NOT EXISTS workspace_id BIGINT NULL REFERENCES workspaces(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_knowledge_files_workspace_created_at
    ON knowledge_files(workspace_id, created_at DESC)
    WHERE deleted_at IS NULL AND workspace_id IS NOT NULL;