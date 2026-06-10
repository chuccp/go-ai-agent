// Package common defines the shared interfaces and types used across the chat system.
package common

import (
	"context"
	"strings"

	"github.com/chuccp/go-ai-agent/ai/types"
	"github.com/chuccp/go-web-frame/config"
	"github.com/spf13/cast"
)

// ---- Options constants ----

const (
	OptionThinking    = "thinking"
	OptionReasoning   = "reasoning"
	OptionJSONMode    = "json_mode"
	OptionTools       = "tools"
	OptionModel       = "model"
	OptionTemperature = "temperature"
	OptionMaxTokens   = "max_tokens"
	OptionTopP        = "top_p"
)

// ---- Provider Info ----

// ProviderInfo is a type alias for the canonical definition in ai/types.
type ProviderInfo = types.ProviderInfo

// ---- Provider & Service interfaces ----

// ChatProvider 聊天提供商接口
type ChatProvider interface {
	Name() string
	Init(ctx context.Context, cfg config.IConfig) error
	GetChat(model string) (ChatService, error)
	GetModels() []string
	GetProviderInfo() ProviderInfo
}

// ChatService 聊天服务接口
type ChatService interface {
	Chat(text string, options *LLMOptions) (string, error)
	ChatWithContext(ctx context.Context, text string, options *LLMOptions) (string, error)
	ChatWithHistory(history []ChatMessage, text string, options *LLMOptions) (string, error)
	ChatWithHistoryWithContext(ctx context.Context, history []ChatMessage, text string, options *LLMOptions) (string, error)
	ChatStream(text string, handler *StreamHandler, options *LLMOptions) error
	ChatStreamWithContext(ctx context.Context, history []ChatMessage, text string, handler *StreamHandler, options *LLMOptions) error
	ChatWithTools(ctx context.Context, history []ChatMessage, text string, opts *LLMOptions) (*ChatResponse, error)
	GetModel() string
	SetModel(model string)
}

// ---- Shared types ----

// ContentPart 多模态消息的单个部分
type ContentPart struct {
	Type     string `json:"type"`                // "text" | "image"
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"` // 图片 URL 或 base64 data URL
}

// ChatMessage 聊天消息
type ChatMessage struct {
	Role         string        `json:"role"`
	Content      string        `json:"content"`
	ContentParts []ContentPart `json:"content_parts,omitempty"` // 多模态内容；非空时 provider 使用此字段
}

// HasContentParts 判断消息是否包含多模态内容
func (m *ChatMessage) HasContentParts() bool {
	return len(m.ContentParts) > 0
}

// GetText 获取消息的纯文本内容（用于存储/显示）
func (m *ChatMessage) GetText() string {
	if m.Content == "" && m.HasContentParts() {
		var texts []string
		for _, p := range m.ContentParts {
			if p.Type == "text" && p.Text != "" {
				texts = append(texts, p.Text)
			}
		}
		return strings.Join(texts, "\n")
	}
	return m.Content
}

// ToolCall 工具调用
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolResult 工具执行结果
type ToolResult struct {
	CallID string
	Output string
}

// ChatResponse 聊天响应（含工具调用）
type ChatResponse struct {
	Text      string
	ToolCalls []ToolCall
}

// ---- StreamHandler ----

type StreamHandler struct {
	OnContentFunc  func(content string, reasoning bool)
	OnCompleteFunc func(content string, reasoning string)
	OnErrorFunc    func(err error)
}

func NewStreamHandler() *StreamHandler {
	return &StreamHandler{}
}

func (h *StreamHandler) OnContent(fn func(content string, reasoning bool)) *StreamHandler {
	h.OnContentFunc = fn
	return h
}

func (h *StreamHandler) OnComplete(fn func(content string, reasoning string)) *StreamHandler {
	h.OnCompleteFunc = fn
	return h
}

func (h *StreamHandler) OnError(fn func(err error)) *StreamHandler {
	h.OnErrorFunc = fn
	return h
}

// ---- LLMOptions ----

type LLMOptions struct {
	options map[string]any
}

func NewLLMOptions() *LLMOptions {
	return &LLMOptions{options: make(map[string]any)}
}

func (o *LLMOptions) Set(key string, value any) *LLMOptions {
	if o.options == nil {
		o.options = make(map[string]any)
	}
	o.options[key] = value
	return o
}

func (o *LLMOptions) Get(key string) (any, bool) {
	if o.options == nil {
		return nil, false
	}
	v, ok := o.options[key]
	return v, ok
}

func (o *LLMOptions) GetString(key string) string {
	if v, ok := o.Get(key); ok {
		return cast.ToString(v)
	}
	return ""
}

func (o *LLMOptions) GetBool(key string) bool {
	if v, ok := o.Get(key); ok {
		return cast.ToBool(v)
	}
	return false
}

func (o *LLMOptions) GetInt(key string) int {
	if v, ok := o.Get(key); ok {
		return cast.ToInt(v)
	}
	return 0
}

func (o *LLMOptions) GetFloat64(key string) float64 {
	if v, ok := o.Get(key); ok {
		return cast.ToFloat64(v)
	}
	return 0
}

func (o *LLMOptions) SetThinking(thinking bool) *LLMOptions {
	return o.Set(OptionThinking, thinking)
}

func (o *LLMOptions) GetThinking() bool {
	return o.GetBool(OptionThinking)
}

func (o *LLMOptions) SetReasoning(reasoning string) *LLMOptions {
	return o.Set(OptionReasoning, reasoning)
}

func (o *LLMOptions) GetReasoning() string {
	return o.GetString(OptionReasoning)
}

func (o *LLMOptions) SetJSONMode(jsonMode bool) *LLMOptions {
	return o.Set(OptionJSONMode, jsonMode)
}

func (o *LLMOptions) GetJSONMode() bool {
	return o.GetBool(OptionJSONMode)
}

func (o *LLMOptions) SetModel(model string) *LLMOptions {
	return o.Set(OptionModel, model)
}

func (o *LLMOptions) GetModel() string {
	return o.GetString(OptionModel)
}

func (o *LLMOptions) SetTemperature(temp float64) *LLMOptions {
	return o.Set(OptionTemperature, temp)
}

func (o *LLMOptions) GetTemperature() float64 {
	return o.GetFloat64(OptionTemperature)
}

func (o *LLMOptions) SetMaxTokens(maxTokens int) *LLMOptions {
	return o.Set(OptionMaxTokens, maxTokens)
}

func (o *LLMOptions) GetMaxTokens() int {
	return o.GetInt(OptionMaxTokens)
}

func (o *LLMOptions) SetTopP(topP float64) *LLMOptions {
	return o.Set(OptionTopP, topP)
}

func (o *LLMOptions) GetTopP() float64 {
	return o.GetFloat64(OptionTopP)
}

func (o *LLMOptions) SetTools(tools any) *LLMOptions {
	return o.Set(OptionTools, tools)
}

func (o *LLMOptions) GetTools() any {
	if v, ok := o.Get(OptionTools); ok {
		return v
	}
	return nil
}

func (o *LLMOptions) Clone() *LLMOptions {
	newOpts := NewLLMOptions()
	for k, v := range o.options {
		newOpts.options[k] = v
	}
	return newOpts
}
