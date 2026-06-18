package claude

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"encoding/json"
	"github.com/chuccp/go-ai-agent/internal/ai/chat/common"
	"github.com/chuccp/go-web-frame/config"
	"github.com/chuccp/go-web-frame/log"
	"go.uber.org/zap"
	"resty.dev/v3"
)

const AnthropicVersion = "2023-06-01"
const DefaultMaxTokens = 4096

// ProviderDefaults maps provider name → {baseURL, model}
var ProviderDefaults = map[string][2]string{
	"anthropic":         {"https://api.anthropic.com", "claude-sonnet-4-6"},
	"deepseek":          {"https://api.deepseek.com/anthropic", "deepseek-v4-flash"},
	"qwen":              {"https://dashscope.aliyuncs.com/apps/anthropic", "qwen-plus"},
	"zhipu":             {"https://open.bigmodel.cn/api/anthropic", "glm-5.1"},
	"volcengine-claude": {"https://ark.cn-beijing.volces.com/api/anthropic", "doubao-seed-2.0-pro"},
	"baidu":             {"https://qianfan.baidubce.com/anthropic", "ernie-4.0-turbo-8k-latest"},
	"xiaomi":            {"https://api.xiaomimimo.com/anthropic", "mimo-v2.5-pro"},
	"qiniu":             {"https://api.qnaigc.com", "claude-sonnet-4-6"},
	"bedrock":           {"https://bedrock-runtime.us-east-1.amazonaws.com", "claude-sonnet-4-6"},
	"vertex":            {"https://us-central1-aiplatform.googleapis.com", "claude-sonnet-4-6"},
}

// ==================== Anthropic Messages API JSON ====================

type anthropicRequest struct {
	Model     string         `json:"model"`
	MaxTokens int            `json:"max_tokens"`
	Messages  []anthropicMsg `json:"messages"`
	System    string         `json:"system,omitempty"`
	Tools     any            `json:"tools,omitempty"`
	Stream    bool           `json:"stream,omitempty"`
}

type anthropicMsg struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type anthropicImageSource struct {
	Type      string `json:"type"`       // "base64"
	MediaType string `json:"media_type"` // "image/png"
	Data      string `json:"data"`       // raw base64
}

type anthropicContent struct {
	Type      string                `json:"type"`
	Text      string                `json:"text,omitempty"`
	Source    *anthropicImageSource `json:"source,omitempty"`
	ID        string                `json:"id,omitempty"`
	Name      string                `json:"name,omitempty"`
	Input     any                   `json:"input,omitempty"`
	ToolUseID string                `json:"tool_use_id,omitempty"`
}

type anthropicResponse struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Role       string             `json:"role"`
	Content    []anthropicContent `json:"content"`
	StopReason string             `json:"stop_reason"`
}

type anthropicSSE struct {
	Type         string            `json:"type"`
	Delta        *anthropicDelta   `json:"delta"`
	ContentBlock *anthropicContent `json:"content_block"`
}

type anthropicDelta struct {
	Type        string `json:"type"`
	Text        string `json:"text"`
	PartialJSON string `json:"partial_json"`
}

// ==================== Provider ====================

type Provider struct {
	name         string
	configPrefix string
	config       Config
	mu           sync.RWMutex
	initialized  bool
	restyClient  *resty.Client
}

func NewService(name string) *Provider { return &Provider{name: name} }

func (s *Provider) Name() string { return s.name }

func (s *Provider) SetConfigPrefix(prefix string) { s.configPrefix = prefix }

func (s *Provider) Init(_ context.Context, cfg config.IConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.initialized {
		return nil
	}
	key := s.configPrefix
	if key == "" {
		key = "chat." + s.name
	}
	var oc Config
	if err := cfg.UnmarshalKey(key, &oc); err != nil {
		return fmt.Errorf("load chat.%s config failed: %w", s.name, err)
	}
	if def, ok := ProviderDefaults[s.name]; ok {
		if oc.BaseURL == "" {
			oc.BaseURL = def[0]
		}
		if oc.Model == "" {
			oc.Model = def[1]
		}
	} else {
		if oc.BaseURL == "" {
			return fmt.Errorf("chat.%s.baseUrl is required", s.name)
		}
		if oc.Model == "" {
			return fmt.Errorf("chat.%s.model is required", s.name)
		}
	}
	if oc.APIKey == "" {
		return fmt.Errorf("chat.%s.apiKey is required", s.name)
	}
	s.config = oc
	s.restyClient = resty.New().SetBaseURL(oc.GetBaseURL())
	s.initialized = true
	log.Info("Claude provider initialized",
		zap.String("name", s.name),
		zap.String("baseUrl", oc.GetBaseURL()),
		zap.String("model", oc.GetModel()))
	return nil
}

func (s *Provider) GetChat(model string) (common.ChatService, error) {
	if err := s.checkInitialized(); err != nil {
		return nil, err
	}
	m := s.config.GetModel()
	if model != "" && model != "default" {
		m = model
	}
	return &ChatService{
		restyClient: s.restyClient,
		apiKey:      s.config.APIKey,
		model:       m,
	}, nil
}

func (s *Provider) GetModels() []string          { return []string{"default"} }
func (s *Provider) GetProviderInfo() common.ProviderInfo {
	if def, ok := ProviderDefaults[s.name]; ok {
		return common.ProviderInfo{Model: def[1], BaseURL: def[0]}
	}
	return common.ProviderInfo{}
}
func (s *Provider) checkInitialized() error {
	if !s.initialized {
		return fmt.Errorf("Claude provider %s not initialized", s.name)
	}
	return nil
}

// ==================== ChatService ====================

type ChatService struct {
	restyClient *resty.Client
	apiKey      string
	model       string
}

func (c *ChatService) Chat(text string, opts *common.LLMOptions) (string, error) {
	return c.ChatWithContext(context.Background(), text, opts)
}

func (c *ChatService) ChatWithContext(ctx context.Context, text string, opts *common.LLMOptions) (string, error) {
	req := c.buildRequest([]common.ChatMessage{{Role: "user", Content: text}}, opts, false)
	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return "", err
	}
	return getTextContent(resp), nil
}

func (c *ChatService) ChatWithHistory(history []common.ChatMessage, text string, opts *common.LLMOptions) (string, error) {
	return c.ChatWithHistoryWithContext(context.Background(), history, text, opts)
}

func (c *ChatService) ChatWithHistoryWithContext(ctx context.Context, history []common.ChatMessage, text string, opts *common.LLMOptions) (string, error) {
	msgs := history
	if text != "" {
		msgs = append(msgs, common.ChatMessage{Role: "user", Content: text})
	}
	req := c.buildRequest(msgs, opts, false)
	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return "", err
	}
	return getTextContent(resp), nil
}

func (c *ChatService) ChatStream(text string, handler *common.StreamHandler, opts *common.LLMOptions) error {
	return c.ChatStreamWithContext(context.Background(), nil, text, handler, opts)
}

func (c *ChatService) ChatStreamWithContext(ctx context.Context, history []common.ChatMessage, text string, handler *common.StreamHandler, opts *common.LLMOptions) error {
	msgs := history
	if text != "" {
		msgs = append(msgs, common.ChatMessage{Role: "user", Content: text})
	}
	req := c.buildRequest(msgs, opts, true)

	r, err := c.restyClient.R().
		SetContext(ctx).
		SetHeader("x-api-key", c.apiKey).
		SetHeader("anthropic-version", AnthropicVersion).
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetResponseDoNotParse(true).
		Post("/v1/messages")
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer r.RawResponse.Body.Close()

	if r.StatusCode() != 200 {
		body, _ := io.ReadAll(r.RawResponse.Body)
		return fmt.Errorf("API error (%d): %s", r.StatusCode(), string(body))
	}

	var fullText, fullReasoning strings.Builder
	scanner := bufio.NewScanner(r.RawResponse.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		var evt anthropicSSE
		if json.Unmarshal([]byte(data), &evt) != nil {
			continue
		}
		if evt.Type == "content_block_delta" && evt.Delta != nil && evt.Delta.Type == "text_delta" {
			fullText.WriteString(evt.Delta.Text)
			if handler.OnContentFunc != nil {
				handler.OnContentFunc(evt.Delta.Text, false)
			}
		}
		if evt.Type == "message_stop" {
			if handler.OnCompleteFunc != nil {
				handler.OnCompleteFunc(fullText.String(), fullReasoning.String())
			}
			return nil
		}
	}
	if handler.OnCompleteFunc != nil {
		handler.OnCompleteFunc(fullText.String(), fullReasoning.String())
	}
	return nil
}

func (c *ChatService) ChatWithTools(ctx context.Context, history []common.ChatMessage, text string, opts *common.LLMOptions) (*common.ChatResponse, error) {
	msgs := history
	if text != "" {
		msgs = append(msgs, common.ChatMessage{Role: "user", Content: text})
	}
	req := c.buildRequest(msgs, opts, false)
	if tools := opts.GetTools(); tools != nil {
		req.Tools = tools
	}
	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	cr := &common.ChatResponse{}
	for _, content := range resp.Content {
		switch content.Type {
		case "text":
			cr.Text += content.Text
		case "tool_use":
			inputJSON, _ := json.Marshal(content.Input)
			cr.ToolCalls = append(cr.ToolCalls, common.ToolCall{
				ID:        content.ID,
				Name:      content.Name,
				Arguments: string(inputJSON),
			})
		}
	}
	return cr, nil
}

func (c *ChatService) GetModel() string  { return c.model }
func (c *ChatService) SetModel(m string) { c.model = m }

func (c *ChatService) ChatWithToolsStream(ctx context.Context, history []common.ChatMessage, text string, opts *common.LLMOptions, handler *common.StreamHandler) (*common.ChatResponse, error) {
	msgs := history
	if text != "" {
		msgs = append(msgs, common.ChatMessage{Role: "user", Content: text})
	}
	req := c.buildRequest(msgs, opts, true)
	if tools := opts.GetTools(); tools != nil {
		req.Tools = tools
	}

	r, err := c.restyClient.R().
		SetContext(ctx).
		SetHeader("x-api-key", c.apiKey).
		SetHeader("anthropic-version", AnthropicVersion).
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetResponseDoNotParse(true).
		Post("/v1/messages")
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer r.RawResponse.Body.Close()

	if r.StatusCode() != 200 {
		body, _ := io.ReadAll(r.RawResponse.Body)
		return nil, fmt.Errorf("API error (%d): %s", r.StatusCode(), string(body))
	}

	var fullText strings.Builder
	var toolCalls []common.ToolCall
	var currentToolID, currentToolName string
	var currentToolArgs strings.Builder
	inToolBlock := false

	scanner := bufio.NewScanner(r.RawResponse.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		var evt anthropicSSE
		if json.Unmarshal([]byte(data), &evt) != nil {
			continue
		}

		switch evt.Type {
		case "content_block_start":
			if evt.ContentBlock != nil && evt.ContentBlock.Type == "tool_use" {
				inToolBlock = true
				currentToolID = evt.ContentBlock.ID
				currentToolName = evt.ContentBlock.Name
				currentToolArgs.Reset()
			}
		case "content_block_delta":
			if evt.Delta != nil {
				if evt.Delta.Type == "text_delta" && evt.Delta.Text != "" {
					fullText.WriteString(evt.Delta.Text)
					if handler != nil && handler.OnContentFunc != nil {
						handler.OnContentFunc(evt.Delta.Text, false)
					}
				}
				if evt.Delta.Type == "input_json_delta" && evt.Delta.PartialJSON != "" {
					currentToolArgs.WriteString(evt.Delta.PartialJSON)
				}
			}
		case "content_block_stop":
			if inToolBlock {
				toolCalls = append(toolCalls, common.ToolCall{
					ID:        currentToolID,
					Name:      currentToolName,
					Arguments: currentToolArgs.String(),
				})
				inToolBlock = false
			}
		case "message_stop":
			if handler != nil && handler.OnCompleteFunc != nil {
				handler.OnCompleteFunc(fullText.String(), "")
			}
			return &common.ChatResponse{
				Text:      fullText.String(),
				ToolCalls: toolCalls,
			}, nil
		}
	}

	if handler != nil && handler.OnCompleteFunc != nil {
		handler.OnCompleteFunc(fullText.String(), "")
	}
	return &common.ChatResponse{
		Text:      fullText.String(),
		ToolCalls: toolCalls,
	}, nil
}

func (c *ChatService) buildRequest(messages []common.ChatMessage, opts *common.LLMOptions, stream bool) anthropicRequest {
	req := anthropicRequest{Model: c.model, MaxTokens: DefaultMaxTokens, Stream: stream}
	if opts != nil {
		if tokens := opts.GetMaxTokens(); tokens > 0 {
			req.MaxTokens = tokens
		}
	}
	var msgs []anthropicMsg
	for _, m := range messages {
		if m.Role == "system" {
			req.System = m.Content
			continue
		}
		// Tool result messages: convert role=tool to user with tool_result content block
		if m.Role == "tool" && m.ToolCallID != "" {
			content := []anthropicContent{{
				Type:      "tool_result",
				ToolUseID: m.ToolCallID,
				Text:      m.Content,
			}}
			msgs = append(msgs, anthropicMsg{Role: "user", Content: content})
			continue
		}
		// Assistant messages with tool calls: convert to content blocks
		if m.Role == "assistant" && len(m.ToolCalls) > 0 {
			var content []anthropicContent
			if m.Content != "" {
				content = append(content, anthropicContent{Type: "text", Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				var input any
				if tc.Arguments != "" {
					json.Unmarshal([]byte(tc.Arguments), &input)
				}
				content = append(content, anthropicContent{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Name,
					Input: input,
				})
			}
			msgs = append(msgs, anthropicMsg{Role: "assistant", Content: content})
			continue
		}
		var content any = m.Content
		if m.HasContentParts() {
			content = buildAnthropicContent(m.ContentParts, m.Content)
		}
		msgs = append(msgs, anthropicMsg{Role: m.Role, Content: content})
	}
	req.Messages = msgs
	return req
}

func buildAnthropicContent(parts []common.ContentPart, textContent string) []anthropicContent {
	var result []anthropicContent
	if textContent != "" {
		result = append(result, anthropicContent{Type: "text", Text: textContent})
	}
	for _, p := range parts {
		switch p.Type {
		case "text":
			result = append(result, anthropicContent{Type: "text", Text: p.Text})
		case "image":
			mediaType, data := parseDataURL(p.ImageURL)
			if mediaType == "" {
				mediaType = "image/png"
			}
			result = append(result, anthropicContent{
				Type: "image",
				Source: &anthropicImageSource{
					Type:      "base64",
					MediaType: mediaType,
					Data:      data,
				},
			})
		}
	}
	return result
}

func parseDataURL(url string) (mediaType string, data string) {
	if !strings.HasPrefix(url, "data:") {
		return "", url
	}
	// data:image/png;base64,<data>
	rest := strings.TrimPrefix(url, "data:")
	idx := strings.Index(rest, ";base64,")
	if idx < 0 {
		return "", rest
	}
	mediaType = rest[:idx]
	data = rest[idx+len(";base64,"):]
	return
}

func (c *ChatService) doRequest(ctx context.Context, req anthropicRequest) (*anthropicResponse, error) {
	var resp anthropicResponse
	r, err := c.restyClient.R().
		SetContext(ctx).
		SetHeader("x-api-key", c.apiKey).
		SetHeader("anthropic-version", AnthropicVersion).
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetResult(&resp).
		Post("/v1/messages")
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	if r.StatusCode() != 200 {
		return nil, fmt.Errorf("API error (%d): %s", r.StatusCode(), r.String())
	}
	return &resp, nil
}

func getTextContent(resp *anthropicResponse) string {
	var texts []string
	for _, c := range resp.Content {
		if c.Type == "text" {
			texts = append(texts, c.Text)
		}
	}
	return strings.Join(texts, "")
}
