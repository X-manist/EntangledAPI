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

// Workspace holds the schema definition for a user-owned research workspace.
type Workspace struct {
	ent.Schema
}

func (Workspace) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "workspaces"},
	}
}

func (Workspace) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (Workspace) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").
			Comment("所属用户 ID"),
		field.String("name").
			MaxLen(120).
			NotEmpty().
			Comment("工作空间名称"),
		field.String("description").
			SchemaType(map[string]string{"postgres": "text"}).
			Default("").
			Comment("工作空间描述"),
		field.JSON("metadata", map[string]interface{}{}).
			Optional().
			Comment("元数据：默认模型、偏好设置等"),
	}
}

func (Workspace) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("workspaces").
			Field("user_id").
			Unique().
			Required(),
		edge.To("conversations", Conversation.Type),
		edge.To("knowledge_files", KnowledgeFile.Type),
	}
}

func (Workspace) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "updated_at").
			Annotations(entsql.Desc()),
		index.Fields("user_id", "deleted_at"),
	}
}
