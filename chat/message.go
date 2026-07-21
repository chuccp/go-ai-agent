package chat

import "strings"

// 下列类型按 Anthropic Messages API (https://docs.anthropic.com/en/api/messages) 标准定义，
// 用于构造发给模型的请求。字段/JSON 键与官方 schema 一一对应，便于序列化后直接 POST。

// ==================== Tool ====================

// ToolFunction 是发给模型的工具定义。模型据此生成 tool_use content block。
type ToolFunction struct {
	Name          string           `json:"name"`                     // 工具名称（唯一标识）
	Description   string           `json:"description"`              // 工具功能描述（模型据此决定是否调用）
	InputSchema   map[string]any   `json:"input_schema"`             // 输入参数的 JSON Schema
	InputExamples []map[string]any `json:"input_examples,omitempty"` // 调用示例（可选）
}

// ==================== Content block ====================

// ContentType 是 content block 的类型标识
type ContentType string

const (
	ContentTypeText       ContentType = "text"
	ContentTypeImage      ContentType = "image"
	ContentTypeToolUse    ContentType = "tool_use"
	ContentTypeToolResult ContentType = "tool_result"
)

// ContentBlock 是 message.content 数组中的一个元素。
// 每种类型只使用对应的字段，其余为零值（不序列化）。
type ContentBlock struct {
	Type      ContentType `json:"type"`                  // "text" | "image" | "tool_use" | "tool_result"
	Text      string      `json:"text,omitempty"`        // text 类型：文本内容
	Input     any         `json:"input,omitempty"`       // tool_use 类型：工具入参（解析后的对象）
	ID        string      `json:"id,omitempty"`          // tool_use 类型：调用 ID；tool_result 类型：对应的 tool_use ID
	Name      string      `json:"name,omitempty"`        // tool_use 类型：工具名称
	ToolUseID string      `json:"tool_use_id,omitempty"` // tool_result 类型：对应的 tool_use block 的 ID

	// image 类型字段
	Source *ImageSource `json:"source,omitempty"`
}

// ImageSource 描述图片内容（当前仅支持 base64 内联）。
type ImageSource struct {
	Type      string `json:"type"`       // "base64"
	MediaType string `json:"media_type"` // "image/png" | "image/jpeg" | "image/gif" | "image/webp"
	Data      string `json:"data"`       // base64 编码的图片数据
}

// ==================== Message ====================

// Role 是消息发送方
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message 是 messages 数组中的一条消息。
// Content 为 block 数组（与 API 一致）；为兼容纯文本场景，Text 字段在内部可转换为单 text block。
type Message struct {
	Role    Role           `json:"role"`    // "user" | "assistant"
	Content []ContentBlock `json:"content"` // content block 数组
}

// Text 便捷构造：生成一条纯文本 user 消息。
func Text(text string) Message {
	return Message{
		Role:    RoleUser,
		Content: []ContentBlock{{Type: ContentTypeText, Text: text}},
	}
}

// ==================== Request ====================

// Messages 是发给 Claude Messages API 的完整请求体。
type Messages struct {
	Model     string    `json:"model"`      // 模型 ID，如 "claude-opus-4-8"
	MaxTokens int       `json:"max_tokens"` // 最大生成 token 数
	Messages  []Message `json:"messages"`   // 对话历史（user/assistant 交替）

	// 可选字段
	System        string         `json:"system,omitempty"`         // 系统提示（独立于 messages）
	Tools         []ToolFunction `json:"tools,omitempty"`          // 可用工具列表
	Stream        bool           `json:"stream,omitempty"`         // 是否流式返回
	Temperature   *float64       `json:"temperature,omitempty"`    // 采样温度 (0,1]
	TopP          *float64       `json:"top_p,omitempty"`          // nucleus 采样
	TopK          *int           `json:"top_k,omitempty"`          // top-k 采样
	StopSequences []string       `json:"stop_sequences,omitempty"` // 停止序列
	Metadata      map[string]any `json:"metadata,omitempty"`       // 自定义元数据（不透传给模型）
}

// ==================== Response ====================

// StopReason 是模型停止生成的原因
type StopReason string

const (
	StopReasonEndTurn   StopReason = "end_turn"      // 自然结束
	StopReasonMaxTokens StopReason = "max_tokens"    // 达到 max_tokens 上限
	StopReasonToolUse   StopReason = "tool_use"      // 需要调用工具
	StopReasonStopSeq   StopReason = "stop_sequence" // 命中停止序列
)

// Response 是 Messages API 的非流式响应体。
type Response struct {
	ID           string         `json:"id"`                      // 响应唯一 ID
	Type         string         `json:"type"`                    // 通常为 "message"
	Role         string         `json:"role"`                    // 通常为 "assistant"
	Model        string         `json:"model"`                   // 实际使用的模型
	Content      []ContentBlock `json:"content"`                 // 生成的 content block 数组
	StopReason   StopReason     `json:"stop_reason"`             // 停止原因
	StopSequence string         `json:"stop_sequence,omitempty"` // 命中的停止序列
	Usage        Usage          `json:"usage"`                   // token 用量
}

// Usage 记录本次请求的 token 消耗。
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Text 便捷方法：拼接响应中所有 text block 的内容。
func (r *Response) Text() string {
	var b strings.Builder
	for _, cb := range r.Content {
		if cb.Type == ContentTypeText {
			b.WriteString(cb.Text)
		}
	}
	return b.String()
}

// ToolCalls 便捷方法：提取响应中所有 tool_use block。
func (r *Response) ToolCalls() []ContentBlock {
	var calls []ContentBlock
	for _, b := range r.Content {
		if b.Type == ContentTypeToolUse {
			calls = append(calls, b)
		}
	}
	return calls
}
