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
// 收集所有 SSE 事件并聚合为单个 *chat.Response 返回。
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
	defer r.RawResponse.Body.Close()

	if r.StatusCode() != 200 {
		body, readErr := io.ReadAll(r.RawResponse.Body)
		if readErr != nil {
			return nil, fmt.Errorf("API error (%d), failed to read body: %w", r.StatusCode(), readErr)
		}
		return nil, fmt.Errorf("API error (%d): %s", r.StatusCode(), string(body))
	}

	return s.parseSSE(r.RawResponse.Body)
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

// -------- SSE 解析 --------

// sseEvent 表示 Anthropic 流式响应中的一条 SSE 事件。
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
	ID        string             `json:"id"`
	Type      string             `json:"type"`
	Role      string             `json:"role"`
	Model     string             `json:"model"`
	Usage     chat.Usage         `json:"usage"`
	StopReason chat.StopReason   `json:"stop_reason"`
}

// parseSSE 从 HTTP 响应体中读取 SSE 事件流并聚合成 chat.Response。
func (s *serviceImpl) parseSSE(body io.Reader) (*chat.Response, error) {
	resp := &chat.Response{}
	// 当前正在构建的 tool_use block
	var curToolID, curToolName string
	var curToolArgs strings.Builder
	inToolBlock := false

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var evt sseEvent
		if err := json.Unmarshal([]byte(data), &evt); err != nil {
			continue
		}

		switch evt.Type {
		case "message_start":
			if evt.Message != nil {
				resp.ID = evt.Message.ID
				resp.Type = evt.Message.Type
				resp.Role = evt.Message.Role
				resp.Model = evt.Message.Model
				resp.Usage = evt.Message.Usage
			}

		case "content_block_start":
			if evt.ContentBlock != nil && evt.ContentBlock.Type == chat.ContentTypeToolUse {
				inToolBlock = true
				curToolID = evt.ContentBlock.ID
				curToolName = evt.ContentBlock.Name
				curToolArgs.Reset()
			}

		case "content_block_delta":
			if evt.Delta == nil {
				continue
			}
			switch evt.Delta.Type {
			case "text_delta":
				// 追加到最后一个 text block；若不存在则新建
				if n := len(resp.Content); n > 0 && resp.Content[n-1].Type == chat.ContentTypeText {
					resp.Content[n-1].Text += evt.Delta.Text
				} else {
					resp.Content = append(resp.Content, chat.ContentBlock{
						Type: chat.ContentTypeText,
						Text: evt.Delta.Text,
					})
				}
			case "input_json_delta":
				curToolArgs.WriteString(evt.Delta.PartialJSON)
			}

		case "content_block_stop":
			if inToolBlock {
				var input any
				if argsStr := curToolArgs.String(); argsStr != "" {
					_ = json.Unmarshal([]byte(argsStr), &input)
				}
				resp.Content = append(resp.Content, chat.ContentBlock{
					Type:  chat.ContentTypeToolUse,
					ID:    curToolID,
					Name:  curToolName,
					Input: input,
				})
				inToolBlock = false
			}

		case "message_delta":
			if evt.Delta != nil {
				resp.StopReason = chat.StopReason(evt.Delta.StopReason)
			}
			if evt.Usage != nil {
				resp.Usage.OutputTokens = evt.Usage.OutputTokens
			}

		case "message_stop":
			// 流结束，无需额外处理
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("SSE stream read error: %w", err)
	}

	return resp, nil
}
