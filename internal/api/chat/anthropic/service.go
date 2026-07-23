package anthropic

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/chuccp/go-ai-agent/chat"
	"resty.dev/v3"
)

const (
	AnthropicVersion = "2023-06-01"
	DefaultBaseURL   = "https://api.anthropic.com"
	DefaultMaxTokens = 4096
)

// Service 定义 Anthropic 聊天服务接口，嵌入通用的 chat.IChatService。
type Service interface {
	chat.IChatService
}

// serviceImpl 是 Service 的具体实现，封装 HTTP 客户端与配置。
type serviceImpl struct {
	config      *Config
	restyClient *resty.Client
}

// NewService 根据给定配置创建一个 Anthropic 聊天服务实例。
// 若 BaseURL 为空则默认使用 Anthropic 官方 API 地址。
func NewService(config *Config) Service {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &serviceImpl{
		config:      config,
		restyClient: resty.New().SetBaseURL(baseURL),
	}
}

// ChatWithStream 向 Anthropic Messages API 发送流式请求，
// 返回一个可逐事件读取的 *chat.Response。
func (s *serviceImpl) ChatWithStream(chatMessages *chat.Messages) (*chat.Response, error) {
	// 应用配置中的默认值
	s.applyDefaults(chatMessages)
	chatMessages.Stream = true

	r, err := s.restyClient.R().
		SetHeader("x-api-key", s.config.APIKey).
		SetHeader("anthropic-version", AnthropicVersion).
		SetHeader("Content-Type", "application/json").
		SetBody(chatMessages).
		SetResponseDoNotParse(true).
		Post("/v1/messages")
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	if r.StatusCode() != 200 {
		body, readErr := io.ReadAll(r.RawResponse.Body)
		r.RawResponse.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("API error (%d), failed to read body: %w", r.StatusCode(), readErr)
		}
		return nil, fmt.Errorf("API error (%d): %s", r.StatusCode(), string(body))
	}

	events := make(chan chat.Event, 16)
	resp := chat.NewResponse(events)

	go s.parseSSE(r.RawResponse.Body, events)

	return resp, nil
}

// applyDefaults 将 Config 中的默认值填入请求。
func (s *serviceImpl) applyDefaults(m *chat.Messages) {
	if m.Model == "" && s.config.Model != "" {
		m.Model = s.config.Model
	}
	if m.MaxTokens == 0 {
		m.MaxTokens = DefaultMaxTokens
	}
}

// -------- SSE 解析（goroutine 中运行） --------

// sseEvent 表示 Anthropic 流式响应中的一条原始 SSE 事件。
type sseEvent struct {
	Type         string             `json:"type"`
	Index        int                `json:"index"`
	Delta        *sseDelta          `json:"delta"`
	ContentBlock *chat.ContentBlock `json:"content_block"`
	Message      *sseMessage        `json:"message"`
	Usage        *chat.Usage        `json:"usage"`
}

type sseDelta struct {
	Type        string `json:"type"`
	Text        string `json:"text"`
	PartialJSON string `json:"partial_json"`
	StopReason  string `json:"stop_reason"`
}

type sseMessage struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Role       string         `json:"role"`
	Model      string         `json:"model"`
	Usage      chat.Usage     `json:"usage"`
	StopReason chat.StopReason `json:"stop_reason"`
}

// parseSSE 从 HTTP 响应体中读取 SSE 事件流，转换为 chat.Event 并发送到 channel。
// 解析完成后关闭 channel。
func (s *serviceImpl) parseSSE(body io.ReadCloser, events chan<- chat.Event) {
	defer close(events)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var raw sseEvent
		if err := json.Unmarshal([]byte(data), &raw); err != nil {
			continue
		}

		switch raw.Type {
		case "message_start":
			if raw.Message != nil {
				events <- &chat.MessageStartEvent{
					ID:    raw.Message.ID,
					Model: raw.Message.Model,
					Role:  raw.Message.Role,
					Usage: raw.Message.Usage,
				}
			}

		case "content_block_start":
			if raw.ContentBlock != nil {
				events <- &chat.ContentBlockStartEvent{
					Index:        raw.Index,
					ContentBlock: *raw.ContentBlock,
				}
			}

		case "content_block_delta":
			if raw.Delta != nil {
				events <- &chat.ContentBlockDeltaEvent{
					Index: raw.Index,
					Delta: chat.ContentDelta{
						Type:        raw.Delta.Type,
						Text:        raw.Delta.Text,
						PartialJSON: raw.Delta.PartialJSON,
					},
				}
			}

		case "content_block_stop":
			events <- &chat.ContentBlockStopEvent{Index: raw.Index}

		case "message_delta":
			evt := &chat.MessageDeltaEvent{}
			if raw.Delta != nil {
				evt.StopReason = chat.StopReason(raw.Delta.StopReason)
			}
			if raw.Usage != nil {
				evt.Usage = *raw.Usage
			}
			events <- evt

		case "message_stop":
			events <- &chat.MessageStopEvent{}
			return // 流正常结束
		}
	}

	if err := scanner.Err(); err != nil {
		events <- &chat.ErrorEvent{Err: fmt.Errorf("SSE stream read error: %w", err)}
	}
}
