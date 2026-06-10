package openai

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/chuccp/go-ai-agent/chat/common"
	"github.com/chuccp/go-web-frame/config"
	"github.com/chuccp/go-web-frame/log"
	"go.uber.org/zap"
	"resty.dev/v3"
)

const (
	DefaultTemperature = 0.7
	DefaultTopP        = 0.9
	DefaultMaxTokens   = 4096
)

// ProviderDefaults maps provider name → {baseURL, model}.
var ProviderDefaults = map[string][2]string{
	"openai":          {"https://api.openai.com/v1", "gpt-4o"},
	"deepseek-openai": {"https://api.deepseek.com/v1", "deepseek-v4-flash"},
	"qwen-openai":     {"https://dashscope.aliyuncs.com/compatible-mode/v1", "qwen-plus"},
	"zhipu-openai":    {"https://open.bigmodel.cn/api/paas/v4", "glm-5.1"},
	"baidu-openai":    {"https://qianfan.baidubce.com/v2", "ernie-4.0-turbo-8k-latest"},
	"xiaomi-openai":   {"https://api.xiaomimimo.com/v1", "mimo-v2.5-pro"},
	"qiniu-openai":    {"https://api.qnaigc.com/v1", "claude-sonnet-4-6"},
}

// ==================== Chat Completions API JSON ====================

type chatRequest struct {
	Model         string         `json:"model"`
	Messages      []messageParam `json:"messages"`
	Stream        bool           `json:"stream,omitempty"`
	Temperature   *float64       `json:"temperature,omitempty"`
	TopP          *float64       `json:"top_p,omitempty"`
	MaxTokens     *int           `json:"max_tokens,omitempty"`
	StreamOptions *streamOptions `json:"stream_options,omitempty"`
	Tools         any            `json:"tools,omitempty"`
	ToolChoice    any            `json:"tool_choice,omitempty"`
}

type streamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type messageParam struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content          string     `json:"content"`
			ReasoningContent string     `json:"reasoning_content"`
			ToolCalls        []toolCall `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
}

type toolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type streamResponse struct {
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
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

func (s *Provider) SetConfigPrefix(prefix string) { s.configPrefix = prefix }

func NewService(name string) *Provider { return &Provider{name: name} }

func (s *Provider) Name() string { return s.name }

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
	}
	if oc.BaseURL == "" {
		return fmt.Errorf("chat.%s.baseUrl is required", s.name)
	}
	if oc.Model == "" {
		return fmt.Errorf("chat.%s.model is required", s.name)
	}
	s.config = oc
	s.restyClient = resty.New().SetBaseURL(oc.GetBaseURL())
	s.initialized = true
	log.Info("OpenAI provider initialized",
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

func (s *Provider) GetModels() []string { return []string{"default"} }
func (s *Provider) GetProviderInfo() common.ProviderInfo {
	if def, ok := ProviderDefaults[s.name]; ok {
		return common.ProviderInfo{Model: def[1], BaseURL: def[0]}
	}
	return common.ProviderInfo{}
}
func (s *Provider) checkInitialized() error {
	if !s.initialized {
		return fmt.Errorf("OpenAI provider %s not initialized", s.name)
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
	req := c.buildRequest([]messageParam{{Role: "user", Content: text}}, opts, false)
	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return "", err
	}
	return getResponseContent(resp), nil
}

func (c *ChatService) ChatWithHistory(history []common.ChatMessage, text string, opts *common.LLMOptions) (string, error) {
	return c.ChatWithHistoryWithContext(context.Background(), history, text, opts)
}

func (c *ChatService) ChatWithHistoryWithContext(ctx context.Context, history []common.ChatMessage, text string, opts *common.LLMOptions) (string, error) {
	req := c.buildRequest(buildMessages(history, text), opts, false)
	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return "", err
	}
	return getResponseContent(resp), nil
}

func (c *ChatService) ChatStream(text string, handler *common.StreamHandler, opts *common.LLMOptions) error {
	return c.ChatStreamWithContext(context.Background(), nil, text, handler, opts)
}

func (c *ChatService) ChatStreamWithContext(ctx context.Context, history []common.ChatMessage, text string, handler *common.StreamHandler, opts *common.LLMOptions) error {
	req := c.buildRequest(buildMessages(history, text), opts, true)

	r, err := c.restyClient.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+c.apiKey).
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetResponseDoNotParse(true).
		Post("/chat/completions")
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer r.RawResponse.Body.Close()

	if r.StatusCode() != 200 {
		body, _ := io.ReadAll(r.RawResponse.Body)
		return fmt.Errorf("API error (%d): %s", r.StatusCode(), string(body))
	}

	var fullContent, fullReasoning strings.Builder
	scanner := bufio.NewScanner(r.RawResponse.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var sr streamResponse
		if sonic.Unmarshal([]byte(data), &sr) != nil {
			continue
		}
		for _, choice := range sr.Choices {
			if choice.Delta.ReasoningContent != "" {
				fullReasoning.WriteString(choice.Delta.ReasoningContent)
				if handler.OnContentFunc != nil {
					handler.OnContentFunc(choice.Delta.ReasoningContent, true)
				}
			}
			if choice.Delta.Content != "" {
				fullContent.WriteString(choice.Delta.Content)
				if handler.OnContentFunc != nil {
					handler.OnContentFunc(choice.Delta.Content, false)
				}
			}
		}
	}
	if handler.OnCompleteFunc != nil {
		handler.OnCompleteFunc(fullContent.String(), fullReasoning.String())
	}
	return nil
}

func (c *ChatService) ChatWithTools(ctx context.Context, history []common.ChatMessage, text string, opts *common.LLMOptions) (*common.ChatResponse, error) {
	req := c.buildRequest(buildMessages(history, text), opts, false)
	if tools := opts.GetTools(); tools != nil {
		req.Tools = tools
		req.ToolChoice = "auto"
	}
	var crResp chatResponse
	r, err := c.restyClient.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+c.apiKey).
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetResult(&crResp).
		Post("/chat/completions")
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	if r.StatusCode() != 200 {
		return nil, fmt.Errorf("API error (%d): %s", r.StatusCode(), r.String())
	}
	cr := &common.ChatResponse{}
	if len(crResp.Choices) > 0 {
		msg := crResp.Choices[0].Message
		cr.Text = msg.Content
		for _, tc := range msg.ToolCalls {
			cr.ToolCalls = append(cr.ToolCalls, common.ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}
	}
	return cr, nil
}

func (c *ChatService) GetModel() string  { return c.model }
func (c *ChatService) SetModel(m string) { c.model = m }

func (c *ChatService) buildRequest(messages []messageParam, opts *common.LLMOptions, stream bool) chatRequest {
	req := chatRequest{Model: c.model, Messages: messages, Stream: stream}
	t := float64(DefaultTemperature)
	p := float64(DefaultTopP)
	mt := DefaultMaxTokens
	if opts != nil {
		if temp := opts.GetTemperature(); temp > 0 {
			t = temp
		}
		if tp := opts.GetTopP(); tp > 0 {
			p = tp
		}
		if tokens := opts.GetMaxTokens(); tokens > 0 {
			mt = tokens
		}
	}
	req.Temperature = &t
	req.TopP = &p
	req.MaxTokens = &mt
	if stream {
		req.StreamOptions = &streamOptions{IncludeUsage: true}
	}
	return req
}

func (c *ChatService) doRequest(ctx context.Context, req chatRequest) (*chatResponse, error) {
	var cr chatResponse
	r, err := c.restyClient.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+c.apiKey).
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetResult(&cr).
		Post("/chat/completions")
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	if r.StatusCode() != 200 {
		return nil, fmt.Errorf("API error (%d): %s", r.StatusCode(), r.String())
	}
	return &cr, nil
}

func getResponseContent(resp *chatResponse) string {
	if resp == nil || len(resp.Choices) == 0 {
		return ""
	}
	return resp.Choices[0].Message.Content
}

func buildMessages(history []common.ChatMessage, text string) []messageParam {
	msgs := make([]messageParam, 0, len(history)+1)
	for _, m := range history {
		msgs = append(msgs, messageParam{Role: m.Role, Content: m.Content})
	}
	if text != "" {
		msgs = append(msgs, messageParam{Role: "user", Content: text})
	}
	return msgs
}
