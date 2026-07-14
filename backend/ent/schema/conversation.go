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

// Conversation holds the schema definition for the Conversation entity.
type Conversation struct {
	ent.Schema
}

func (Conversation) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "conversations"},
	}
}

func (Conversation) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (Conversation) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").
			Comment("所属用户 ID"),
		field.Int64("workspace_id").
			Optional().
			Nillable().
			Comment("所属工作空间 ID"),
		field.String("title").
			MaxLen(255).
			Default("").
			Comment("对话标题（自动生成或用户编辑）"),
		field.JSON("knowledge_ids", []int64{}).
			Optional().
			Comment("关联的知识库文件 ID 列表"),
		field.JSON("metadata", map[string]interface{}{}).
			Optional().
			Comment("元数据：模型配置、系统提示词等"),
	}
}

func (Conversation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("conversations").
			Field("user_id").
			Unique().
			Required(),
		edge.From("workspace", Workspace.Type).
			Ref("conversations").
			Field("workspace_id").
			Unique(),
		edge.To("messages", Message.Type),
	}
}

func (Conversation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "created_at").
			Annotations(entsql.Desc()),
		index.Fields("workspace_id", "updated_at"),
		index.Fields("user_id", "deleted_at"),
	}
}
