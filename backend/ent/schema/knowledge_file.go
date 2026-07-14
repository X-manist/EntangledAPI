package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// KnowledgeFile holds the schema definition for the KnowledgeFile entity.
type KnowledgeFile struct {
	ent.Schema
}

func (KnowledgeFile) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "knowledge_files"},
	}
}

func (KnowledgeFile) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (KnowledgeFile) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").
			Comment("所属用户 ID"),
		field.Int64("workspace_id").
			Optional().
			Nillable().
			Comment("所属工作空间 ID"),
		field.String("filename").
			MaxLen(255).
			Comment("原始文件名"),
		field.String("file_type").
			MaxLen(50).
			Comment("文件类型：pdf/txt/doc/csv 等"),
		field.Int64("file_size").
			Default(0).
			Comment("文件大小（字节）"),
		field.String("storage_path").
			MaxLen(512).
			Comment("存储路径或 key"),
		field.Text("content").
			Optional().
			Comment("提取的文本内容"),
		field.String("content_hash").
			MaxLen(64).
			Optional().
			Comment("内容哈希（用于去重）"),
		field.JSON("metadata", map[string]interface{}{}).
			Optional().
			Comment("元数据：作者、标题、摘要等"),
		field.String("status").
			MaxLen(20).
			Default("uploaded").
			Comment("状态：uploaded/processing/ready/failed"),
	}
}

func (KnowledgeFile) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("knowledge_files").
			Field("user_id").
			Unique().
			Required(),
		edge.From("workspace", Workspace.Type).
			Ref("knowledge_files").
			Field("workspace_id").
			Unique(),
	}
}

func (KnowledgeFile) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "created_at").
			Annotations(entsql.Desc()),
		index.Fields("workspace_id", "created_at"),
		index.Fields("user_id", "deleted_at"),
		index.Fields("content_hash"),
	}
}
