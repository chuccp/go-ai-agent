package anthropic

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/chuccp/go-agent-sdk/chat"
	"resty.dev/v3"
)

const (
	AnthropicVersion = "2023-06-01"
	DefaultBaseURL   = "https://api.anthropic.com"
	DefaultMaxTokens = 4096
)

// Service 瀹氫箟 Anthropic 鑱婂ぉ鏈嶅姟鎺ュ彛锛屽祵鍏ラ€氱敤鐨?chat.ChatService銆?
type Service interface {
	chat.ChatService
}

// serviceImpl 鏄?Service 鐨勫叿浣撳疄鐜帮紝灏佽 HTTP 瀹㈡埛绔笌閰嶇疆銆?
type serviceImpl struct {
	config      *Config
	restyClient *resty.Client
}

// NewService 鏍规嵁缁欏畾閰嶇疆鍒涘缓涓€涓?Anthropic 鑱婂ぉ鏈嶅姟瀹炰緥銆?
// 鑻?BaseURL 涓虹┖鍒欓粯璁や娇鐢?Anthropic 瀹樻柟 API 鍦板潃銆?
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

// ChatWithStream 鍚?Anthropic Messages API 鍙戦€佹祦寮忚姹傦紝
// 灏嗕簨浠跺啓鍏?response锛屽畬鎴愬悗鍏抽棴銆?
func (s *serviceImpl) ChatWithStream(ctx context.Context, chatMessages *chat.Request, response chat.StreamWriter) error {
	// 搴旂敤閰嶇疆涓殑榛樿鍊?
	s.applyDefaults(chatMessages)
	chatMessages.Stream = true

	r, err := s.restyClient.R().
		SetContext(ctx).
		SetHeader("x-api-key", s.config.APIKey).
		SetHeader("anthropic-version", AnthropicVersion).
		SetHeader("Content-Type", "application/json").
		SetBody(chatMessages).
		SetResponseDoNotParse(true).
		Post("/v1/messages")
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}

	if r.StatusCode() != 200 {
		body, readErr := io.ReadAll(r.RawResponse.Body)
		r.RawResponse.Body.Close()
		if readErr != nil {
			return fmt.Errorf("API error (%d), failed to read body: %w", r.StatusCode(), readErr)
		}
		return fmt.Errorf("API error (%d): %s", r.StatusCode(), string(body))
	}

	s.parseSSE(r.RawResponse.Body, response)
	return nil
}

// applyDefaults 灏?Config 涓殑榛樿鍊煎～鍏ヨ姹傘€?
func (s *serviceImpl) applyDefaults(m *chat.Request) {
	if m.Model == "" && s.config.Model != "" {
		m.Model = s.config.Model
	}
	if m.MaxTokens == 0 {
		m.MaxTokens = DefaultMaxTokens
	}
}

// -------- SSE 瑙ｆ瀽锛坓oroutine 涓繍琛岋級 --------

// sseEvent 琛ㄧず Anthropic 娴佸紡鍝嶅簲涓殑涓€鏉″師濮?SSE 浜嬩欢銆?
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
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Role       string          `json:"role"`
	Model      string          `json:"model"`
	Usage      chat.Usage      `json:"usage"`
	StopReason chat.StopReason `json:"stop_reason"`
}

// parseSSE 浠?HTTP 鍝嶅簲浣撲腑璇诲彇 SSE 浜嬩欢娴侊紝杞崲涓?chat.Event 骞跺啓鍏?Response銆?
// 瑙ｆ瀽瀹屾垚鍚庡叧闂?Response銆?
func (s *serviceImpl) parseSSE(body io.ReadCloser, resp chat.StreamWriter) {
	defer resp.Close()
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
				resp.Write(&chat.MessageStartEvent{
					ID:    raw.Message.ID,
					Model: raw.Message.Model,
					Role:  raw.Message.Role,
					Usage: raw.Message.Usage,
				})
			}

		case "content_block_start":
			if raw.ContentBlock != nil {
				resp.Write(&chat.ContentBlockStartEvent{
					Index:        raw.Index,
					ContentBlock: *raw.ContentBlock,
				})
			}

		case "content_block_delta":
			if raw.Delta != nil {
				resp.Write(&chat.ContentBlockDeltaEvent{
					Index: raw.Index,
					Delta: chat.ContentDelta{
						Type:        raw.Delta.Type,
						Text:        raw.Delta.Text,
						PartialJSON: raw.Delta.PartialJSON,
					},
				})
			}

		case "content_block_stop":
			resp.Write(&chat.ContentBlockStopEvent{Index: raw.Index})

		case "message_delta":
			evt := &chat.MessageDeltaEvent{}
			if raw.Delta != nil {
				evt.StopReason = chat.StopReason(raw.Delta.StopReason)
			}
			if raw.Usage != nil {
				evt.Usage = *raw.Usage
			}
			resp.Write(evt)

		case "message_stop":
			resp.Write(&chat.MessageStopEvent{})
			return // 娴佹甯哥粨鏉?
		}
	}

	if err := scanner.Err(); err != nil {
		resp.Write(&chat.ErrorEvent{Err: fmt.Errorf("SSE stream read error: %w", err)})
	}
}
