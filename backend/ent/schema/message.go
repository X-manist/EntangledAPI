package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Message holds the schema definition for the Message entity.
type Message struct {
	ent.Schema
}

func (Message) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "messages"},
	}
}

func (Message) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (Message) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("conversation_id").
			Comment("所属对话 ID"),
		field.String("role").
			MaxLen(20).
			Comment("角色：user/assistant/system"),
		field.Text("content").
			Comment("消息内容"),
		field.JSON("attachments", []map[string]interface{}{}).
			Optional().
			Comment("附件：图片、文件、代码块等"),
		field.Int("tokens_used").
			Default(0).
			Comment("消耗的 token 数量"),
		field.Float("cost").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0).
			Comment("本条消息的费用"),
		field.String("model").
			MaxLen(100).
			Optional().
			Comment("使用的模型"),
		field.JSON("metadata", map[string]interface{}{}).
			Optional().
			Comment("元数据：finish_reason、tool_calls 等"),
	}
}

func (Message) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("conversation", Conversation.Type).
			Ref("messages").
			Field("conversation_id").
			Unique().
			Required(),
	}
}

func (Message) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("conversation_id", "created_at").
			Annotations(entsql.Desc()),
	}
}
