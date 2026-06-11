package gemini

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/chuccp/go-ai-agent/agent/tool"
	"github.com/chuccp/go-ai-agent/ai/chat/common"
	"github.com/chuccp/go-web-frame/config"
	"github.com/chuccp/go-web-frame/log"
	"go.uber.org/zap"
	"resty.dev/v3"
)

const ServiceName = "gemini"
const DefaultMaxTokens = 8192
const DefaultBaseURL = "https://generativelanguage.googleapis.com"

// ProviderDefaults maps provider name → {baseURL, model}.
var ProviderDefaults = map[string][2]string{
	ServiceName: {DefaultBaseURL, "gemini-3.5-flash"},
}

// ==================== Gemini API JSON ====================

type geminiRequest struct {
	Contents         []geminiContent    `json:"contents"`
	SystemInstruction *geminiSystemInst `json:"systemInstruction,omitempty"`
	GenerationConfig *geminiGenConfig   `json:"generationConfig,omitempty"`
	Tools            any                `json:"tools,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type geminiPart struct {
	Text         string               `json:"text,omitempty"`
	InlineData   *geminiInlineData    `json:"inlineData,omitempty"`
	FunctionCall *geminiFunctionCall  `json:"functionCall,omitempty"`
	FunctionResp *geminiFunctionResp  `json:"functionResponse,omitempty"`
}

type geminiFunctionCall struct {
	Name string `json:"name"`
	Args any    `json:"args,omitempty"`
}

type geminiFunctionResp struct {
	Name  string `json:"name"`
	Response any `json:"response"`
}

type geminiSystemInst struct {
	Parts []geminiPart `json:"parts"`
}

type geminiGenConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	TopP            *float64 `json:"topP,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
	ResponseMimeType string  `json:"responseMimeType,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text         string            `json:"text,omitempty"`
				FunctionCall *geminiFuncCallResp `json:"functionCall,omitempty"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	Error *geminiError `json:"error,omitempty"`
}

type geminiFuncCallResp struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args,omitempty"`
}

type geminiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ==================== GeminiProvider ====================

type GeminiProvider struct {
	configPrefix string
	config       GeminiConfig
	mu           sync.RWMutex
	initialized  bool
	restyClient  *resty.Client
}

func NewGeminiService() *GeminiProvider { return &GeminiProvider{} }

func (s *GeminiProvider) Name() string { return ServiceName }

func (s *GeminiProvider) SetConfigPrefix(prefix string) { s.configPrefix = prefix }

func (s *GeminiProvider) Init(ctx context.Context, cfg config.IConfig) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.initialized {
		return nil
	}
	key := s.configPrefix
	if key == "" {
		key = "chat.gemini"
	}
	var gc GeminiConfig
	if err := cfg.UnmarshalKey(key, &gc); err != nil {
		return fmt.Errorf("load gemini config failed: %w", err)
	}
	if gc.APIKey == "" {
		return fmt.Errorf("Gemini API Key is empty")
	}
	s.config = gc
	s.restyClient = resty.New().SetBaseURL("https://generativelanguage.googleapis.com")
	s.initialized = true
	log.Info("Google Gemini initialized",
		zap.String("model", gc.GetModel()))
	return nil
}

func (s *GeminiProvider) GetChat(model string) (common.ChatService, error) {
	if err := s.checkInitialized(); err != nil {
		return nil, err
	}
	m := s.config.GetModel()
	if model != "" && model != "default" {
		m = model
	}
	return &GeminiChat{
		restyClient: s.restyClient,
		apiKey:      s.config.APIKey,
		model:       m,
	}, nil
}

func (s *GeminiProvider) GetModels() []string { return []string{"default"} }
func (s *GeminiProvider) GetProviderInfo() common.ProviderInfo {
	if def, ok := ProviderDefaults[ServiceName]; ok {
		return common.ProviderInfo{Model: def[1], BaseURL: def[0]}
	}
	return common.ProviderInfo{}
}

func (s *GeminiProvider) checkInitialized() error {
	if !s.initialized {
		return fmt.Errorf("Gemini not initialized")
	}
	return nil
}

// ==================== GeminiChat ====================

type GeminiChat struct {
	restyClient *resty.Client
	apiKey      string
	model       string
}

func (c *GeminiChat) Chat(text string, opts *common.LLMOptions) (string, error) {
	return c.ChatWithContext(context.Background(), text, opts)
}

func (c *GeminiChat) ChatWithContext(ctx context.Context, text string, opts *common.LLMOptions) (string, error) {
	req := c.buildRequest([]common.ChatMessage{{Role: "user", Content: text}}, opts)
	resp, err := c.doRequest(ctx, req, false)
	if err != nil {
		return "", err
	}
	return getResponseText(resp), nil
}

func (c *GeminiChat) ChatWithHistory(history []common.ChatMessage, text string, opts *common.LLMOptions) (string, error) {
	return c.ChatWithHistoryWithContext(context.Background(), history, text, opts)
}

func (c *GeminiChat) ChatWithHistoryWithContext(ctx context.Context, history []common.ChatMessage, text string, opts *common.LLMOptions) (string, error) {
	msgs := history
	if text != "" {
		msgs = append(msgs, common.ChatMessage{Role: "user", Content: text})
	}
	req := c.buildRequest(msgs, opts)
	resp, err := c.doRequest(ctx, req, false)
	if err != nil {
		return "", err
	}
	return getResponseText(resp), nil
}

func (c *GeminiChat) ChatStream(text string, handler *common.StreamHandler, opts *common.LLMOptions) error {
	return c.ChatStreamWithContext(context.Background(), nil, text, handler, opts)
}

func (c *GeminiChat) ChatStreamWithContext(ctx context.Context, history []common.ChatMessage, text string, handler *common.StreamHandler, opts *common.LLMOptions) error {
	msgs := history
	if text != "" {
		msgs = append(msgs, common.ChatMessage{Role: "user", Content: text})
	}
	req := c.buildRequest(msgs, opts)

	url := fmt.Sprintf("/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", c.model, c.apiKey)
	r, err := c.restyClient.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetResponseDoNotParse(true).
		Post(url)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer r.RawResponse.Body.Close()

	if r.StatusCode() != 200 {
		body, _ := io.ReadAll(r.RawResponse.Body)
		return fmt.Errorf("API error (%d): %s", r.StatusCode(), string(body))
	}

	var fullContent strings.Builder
	var fullReasoning strings.Builder

	scanner := bufio.NewScanner(r.RawResponse.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		var evt geminiResponse
		if json.Unmarshal([]byte(data), &evt) != nil {
			continue
		}
		if evt.Error != nil {
			if handler.OnErrorFunc != nil {
				handler.OnErrorFunc(fmt.Errorf("Gemini error (%d): %s", evt.Error.Code, evt.Error.Message))
			}
			return nil
		}
		for _, cand := range evt.Candidates {
			for _, part := range cand.Content.Parts {
				if part.Text != "" {
					fullContent.WriteString(part.Text)
					if handler.OnContentFunc != nil {
						handler.OnContentFunc(part.Text, false)
					}
				}
			}
		}
	}

	if handler.OnCompleteFunc != nil {
		handler.OnCompleteFunc(fullContent.String(), fullReasoning.String())
	}
	return nil
}

func (c *GeminiChat) ChatWithTools(ctx context.Context, history []common.ChatMessage, text string, opts *common.LLMOptions) (*common.ChatResponse, error) {
	msgs := history
	if text != "" {
		msgs = append(msgs, common.ChatMessage{Role: "user", Content: text})
	}
	req := c.buildRequest(msgs, opts)

	if tools := opts.GetTools(); tools != nil {
		req.Tools = convertToolsForGemini(tools)
	}

	resp, err := c.doRequest(ctx, req, false)
	if err != nil {
		return nil, err
	}

	cr := &common.ChatResponse{}
	for _, cand := range resp.Candidates {
		for _, part := range cand.Content.Parts {
			if part.Text != "" {
				cr.Text += part.Text
			}
			if part.FunctionCall != nil {
				argsJSON, _ := json.Marshal(part.FunctionCall.Args)
				cr.ToolCalls = append(cr.ToolCalls, common.ToolCall{
					Name:      part.FunctionCall.Name,
					Arguments: string(argsJSON),
				})
			}
		}
	}
	return cr, nil
}

func (c *GeminiChat) GetModel() string  { return c.model }
func (c *GeminiChat) SetModel(m string) { c.model = m }

// ==================== helpers ====================

func (c *GeminiChat) buildRequest(messages []common.ChatMessage, opts *common.LLMOptions) geminiRequest {
	req := geminiRequest{
		Contents: make([]geminiContent, 0),
	}

	maxTokens := DefaultMaxTokens
	var temperature *float64
	var topP *float64
	var responseMIME string

	if opts != nil {
		if tokens := opts.GetMaxTokens(); tokens > 0 {
			maxTokens = tokens
		}
		if t := opts.GetTemperature(); t > 0 {
			temperature = &t
		}
		if p := opts.GetTopP(); p > 0 {
			topP = &p
		}
		if opts.GetJSONMode() {
			responseMIME = "application/json"
		}
	}

	req.GenerationConfig = &geminiGenConfig{
		MaxOutputTokens:  &maxTokens,
		Temperature:      temperature,
		TopP:             topP,
		ResponseMimeType: responseMIME,
	}

	for _, m := range messages {
		if m.Role == "system" {
			req.SystemInstruction = &geminiSystemInst{
				Parts: buildGeminiParts(m.Content, m.ContentParts),
			}
			continue
		}
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		req.Contents = append(req.Contents, geminiContent{
			Role:  role,
			Parts: buildGeminiParts(m.Content, m.ContentParts),
		})
	}

	return req
}

func buildGeminiParts(textContent string, contentParts []common.ContentPart) []geminiPart {
	if len(contentParts) == 0 {
		return []geminiPart{{Text: textContent}}
	}
	var parts []geminiPart
	if textContent != "" {
		parts = append(parts, geminiPart{Text: textContent})
	}
	for _, p := range contentParts {
		switch p.Type {
		case "text":
			parts = append(parts, geminiPart{Text: p.Text})
		case "image":
			mediaType, data := parseGeminiDataURL(p.ImageURL)
			parts = append(parts, geminiPart{
				InlineData: &geminiInlineData{MimeType: mediaType, Data: data},
			})
		}
	}
	return parts
}

func parseGeminiDataURL(url string) (mediaType string, data string) {
	if !strings.HasPrefix(url, "data:") {
		return "image/png", url
	}
	rest := strings.TrimPrefix(url, "data:")
	idx := strings.Index(rest, ";base64,")
	if idx < 0 {
		return "image/png", rest
	}
	return rest[:idx], rest[idx+len(";base64,"):]
}

func (c *GeminiChat) doRequest(ctx context.Context, req geminiRequest, stream bool) (*geminiResponse, error) {
	url := fmt.Sprintf("/v1beta/models/%s:generateContent?key=%s", c.model, c.apiKey)

	var resp geminiResponse
	r, err := c.restyClient.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetResult(&resp).
		Post(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	if r.StatusCode() != 200 {
		return nil, fmt.Errorf("API error (%d): %s", r.StatusCode(), r.String())
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("Gemini error (%d): %s", resp.Error.Code, resp.Error.Message)
	}
	return &resp, nil
}

func getResponseText(resp *geminiResponse) string {
	var texts []string
	for _, cand := range resp.Candidates {
		for _, part := range cand.Content.Parts {
			if part.Text != "" {
				texts = append(texts, part.Text)
			}
		}
	}
	return strings.Join(texts, "")
}

func convertToolsForGemini(tools any) []map[string]any {
	src, ok := tools.([]tool.Definition)
	if !ok {
		return nil
	}
	funcDecls := make([]map[string]any, 0, len(src))
	for _, t := range src {
		if t.Type == "web_search_20250305" || t.Type == "web_search_20260209" {
			// Convert Anthropic built-in web search to regular function tool
			funcDecls = append(funcDecls, map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.InputSchema,
			})
			continue
		}
		funcDecls = append(funcDecls, map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"parameters":  t.InputSchema,
		})
	}
	return []map[string]any{{
		"functionDeclarations": funcDecls,
	}}
}
