package volcengine

import (
	"bufio"
	"context"
	"encoding/json"
	"github.com/bytedance/sonic"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/chuccp/go-ai-agent/chat/common"
	"github.com/chuccp/go-ai-agent/agent/tool"
	"github.com/chuccp/go-web-frame/config"
	"github.com/chuccp/go-web-frame/log"
	"go.uber.org/zap"
	"resty.dev/v3"
)

const ServiceName = "volcengine"

const (
	DefaultTemperature = 1.0
	DefaultTopP        = 0.95
	DefaultMaxTokens   = 32768

	// BaseURL Responses API 基础地址
	BaseURL = "https://ark.cn-beijing.volces.com/api/v3"
)

// ProviderDefaults maps provider name → {baseURL, model}.
var ProviderDefaults = map[string][2]string{
	ServiceName: {BaseURL, "doubao-seed-2.0-pro"},
}

// ==================== Responses API JSON 结构体 ====================

// responsesRequest Responses API 请求体
type responsesRequest struct {
	Model              string             `json:"model"`
	Input              any                `json:"input"`
	Temperature        *float64           `json:"temperature,omitempty"`
	TopP               *float64           `json:"top_p,omitempty"`
	MaxOutputTokens    *int               `json:"max_output_tokens,omitempty"`
	Thinking           *responsesThinking `json:"thinking,omitempty"`
	Text               *responsesText     `json:"text,omitempty"`
	Stream             bool               `json:"stream,omitempty"`
	PreviousResponseID *string            `json:"previous_response_id,omitempty"`
	Instructions       *string            `json:"instructions,omitempty"`
	Caching            *responsesCaching  `json:"caching,omitempty"`
	Tools              any                `json:"tools,omitempty"`
	ToolChoice         *responsesToolChoice `json:"tool_choice,omitempty"`
}

type responsesToolChoice struct {
	Type string `json:"type"`
}

type responsesText struct {
	Format responsesTextFormat `json:"format"`
}

type responsesTextFormat struct {
	Type string `json:"type"`
}

type responsesThinking struct {
	Type string `json:"type"`
}

type responsesCaching struct {
	Type string `json:"type"`
}

// responsesObject Responses API 响应体
type responsesObject struct {
	ID                string             `json:"id"`
	Model             string             `json:"model"`
	Status            string             `json:"status"`
	Output            []outputItem       `json:"output"`
	Usage             *responsesUsage    `json:"usage"`
	Error             *responsesError    `json:"error"`
	IncompleteDetails *incompleteDetails `json:"incomplete_details"`
}

type responsesUsage struct {
	InputTokens         int                  `json:"input_tokens"`
	OutputTokens        int                  `json:"output_tokens"`
	TotalTokens         int                  `json:"total_tokens"`
	OutputTokensDetails *outputTokensDetails `json:"output_tokens_details"`
}

type outputTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"`
}

type responsesError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type incompleteDetails struct {
	Reason string `json:"reason"`
}

// outputItem 输出项
type outputItem struct {
	Type      string          `json:"type"`
	Role      string          `json:"role"`
	Content   json.RawMessage `json:"content"`
	Status    string          `json:"status"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	CallID    string          `json:"call_id"`
	Arguments string          `json:"arguments"`
}

// contentItem 内容块（output 数组中的 content）
type contentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// inputMessage 用于构建 Input 数组中的消息
type inputMessage struct {
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ==================== SSE 流式事件结构体 ====================

type sseEvent struct {
	Type           string           `json:"type"`
	Delta          string           `json:"delta,omitempty"`
	Text           string           `json:"text,omitempty"`
	ItemID         string           `json:"item_id,omitempty"`
	OutputIndex    int              `json:"output_index,omitempty"`
	SummaryIndex   int              `json:"summary_index,omitempty"`
	SequenceNumber int64            `json:"sequence_number,omitempty"`
	Response       *responsesObject `json:"response,omitempty"`
	Message        string           `json:"message,omitempty"`
	Code           string           `json:"code,omitempty"`
}

// ==================== VolcengineProvider ====================

// VolcengineProvider 火山引擎聊天提供商
type VolcengineProvider struct {
	configPrefix string
	config       VolcengineConfig
	mu           sync.RWMutex
	initialized  bool
	restyClient  *resty.Client
}

func NewVolcengineService() *VolcengineProvider {
	return &VolcengineProvider{
		restyClient: resty.New().SetBaseURL(BaseURL),
	}
}

func (s *VolcengineProvider) Name() string {
	return ServiceName
}

func (s *VolcengineProvider) SetConfigPrefix(prefix string) { s.configPrefix = prefix }

func (s *VolcengineProvider) Init(ctx context.Context, cfg config.IConfig) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialized {
		return nil
	}

	key := s.configPrefix
	if key == "" {
		key = "chat.volcengine"
	}
	var vc VolcengineConfig
	if err := cfg.UnmarshalKey(key, &vc); err != nil {
		return fmt.Errorf("加载火山引擎配置失败: %w", err)
	}

	if vc.APIKey == "" {
		return fmt.Errorf("火山引擎 API Key 不能为空")
	}

	s.config = vc
	s.initialized = true

	log.Info("火山引擎 Responses API 初始化成功",
		zap.String("baseUrl", BaseURL),
		zap.String("model", vc.GetModel()))
	return nil
}

func (s *VolcengineProvider) GetChat(model string) (common.ChatService, error) {
	if err := s.checkInitialized(); err != nil {
		return nil, err
	}

	modelID := s.config.GetModel()
	if model != "" && model != "default" {
		modelID = model
	}
	return &VolcengineChat{
		restyClient: s.restyClient,
		apiKey:      s.config.APIKey,
		model:       modelID,
	}, nil
}

func (s *VolcengineProvider) GetModels() []string {
	return []string{"default"}
}

func (s *VolcengineProvider) GetProviderInfo() common.ProviderInfo {
	if def, ok := ProviderDefaults[ServiceName]; ok {
		return common.ProviderInfo{Model: def[1], BaseURL: def[0]}
	}
	return common.ProviderInfo{}
}

func (s *VolcengineProvider) checkInitialized() error {
	if !s.initialized {
		return fmt.Errorf("火山引擎服务未初始化")
	}
	return nil
}

// ==================== VolcengineChat ====================

// VolcengineChat 火山引擎 Responses API 聊天实现
type VolcengineChat struct {
	restyClient *resty.Client
	apiKey      string
	model       string
}

func (c *VolcengineChat) Chat(text string, options *common.LLMOptions) (string, error) {
	return c.ChatWithContext(context.Background(), text, options)
}

func (c *VolcengineChat) ChatWithContext(ctx context.Context, text string, options *common.LLMOptions) (string, error) {
	req := c.buildRequest(text, options, false)

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return "", err
	}

	return getOutputText(resp), nil
}

func (c *VolcengineChat) ChatWithHistory(history []common.ChatMessage, text string, options *common.LLMOptions) (string, error) {
	return c.ChatWithHistoryWithContext(context.Background(), history, text, options)
}

func (c *VolcengineChat) ChatWithHistoryWithContext(ctx context.Context, history []common.ChatMessage, text string, options *common.LLMOptions) (string, error) {
	input := buildInputWithHistory(history, text)
	req := c.buildRequest(input, options, false)

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return "", err
	}

	return getOutputText(resp), nil
}

func (c *VolcengineChat) ChatStream(text string, handler *common.StreamHandler, options *common.LLMOptions) error {
	return c.ChatStreamWithContext(context.Background(), nil, text, handler, options)
}

func (c *VolcengineChat) ChatStreamWithContext(ctx context.Context, history []common.ChatMessage, text string, handler *common.StreamHandler, options *common.LLMOptions) error {
	var input any
	if len(history) > 0 {
		input = buildInputWithHistory(history, text)
	} else {
		input = text
	}

	req := c.buildRequest(input, options, true)

	reqBody, _ := sonic.Marshal(req)
	log.Debug("Responses API 流式请求", zap.String("body", string(reqBody)))

	resp, err := c.restyClient.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+c.apiKey).
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetResponseDoNotParse(true).
		Post("/responses")
	if err != nil {
		return fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.RawResponse.Body.Close()

	if resp.StatusCode() != 200 {
		errBody, _ := io.ReadAll(resp.RawResponse.Body)
		return fmt.Errorf("API Error (%d): %s", resp.StatusCode(), string(errBody))
	}

	var fullContent strings.Builder
	var fullReasoning strings.Builder

	scanner := bufio.NewScanner(resp.RawResponse.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var event sseEvent
		if err := sonic.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "response.output_text.delta":
			fullContent.WriteString(event.Delta)
			if handler.OnContentFunc != nil {
				handler.OnContentFunc(event.Delta, false)
			}
		case "response.reasoning_summary_text.delta":
			fullReasoning.WriteString(event.Delta)
			if handler.OnContentFunc != nil {
				handler.OnContentFunc(event.Delta, true)
			}
		case "response.completed", "response.failed", "response.incomplete":
			if event.Type == "response.failed" && handler.OnErrorFunc != nil {
				errMsg := "请求失败"
				if event.Response != nil && event.Response.Error != nil {
					errMsg = event.Response.Error.Message
				}
				handler.OnErrorFunc(fmt.Errorf("%s: %s", event.Type, errMsg))
				return nil
			}
		}
	}

	if handler.OnCompleteFunc != nil {
		handler.OnCompleteFunc(fullContent.String(), fullReasoning.String())
	}

	return nil
}

// ChatWithTools 带工具调用的聊天，解析 function_call
func (c *VolcengineChat) ChatWithTools(ctx context.Context, history []common.ChatMessage, text string, opts *common.LLMOptions) (*common.ChatResponse, error) {
	var input any
	if len(history) > 0 {
		input = buildInputWithHistory(history, text)
	} else if text != "" {
		input = text
	} else {
		return nil, fmt.Errorf("input 和 history 不能同时为空")
	}

	req := c.buildRequest(input, opts, false)

	// 添加工具定义
	if tools := opts.GetTools(); tools != nil {
		req.Tools = convertToolsForVolcengine(tools)
		req.ToolChoice = &responsesToolChoice{Type: "required"}
	}

	reqBody, _ := sonic.Marshal(req)
	log.Info("ChatWithTools 请求", zap.String("body", string(reqBody)))

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	cr := &common.ChatResponse{}

	// 解析 output，区分 text 和 function_call
	for _, item := range resp.Output {
		switch item.Type {
		case "message":
			var contentItems []contentItem
			if err := sonic.Unmarshal(item.Content, &contentItems); err == nil {
				for _, ci := range contentItems {
					if ci.Type == "output_text" {
						cr.Text += ci.Text
					}
				}
			}
		case "function_call":
			tc := common.ToolCall{
				ID:   item.ID,
				Name: item.Name,
			}
			// arguments 在 item 中是 string
			if item.Arguments != "" {
				tc.Arguments = item.Arguments
			}
			cr.ToolCalls = append(cr.ToolCalls, tc)
		}
	}

	return cr, nil
}

func (c *VolcengineChat) GetModel() string {
	return c.model
}

func (c *VolcengineChat) SetModel(model string) {
	c.model = model
}

// ==================== 请求构建 ====================

func (c *VolcengineChat) buildRequest(input any, opts *common.LLMOptions, stream bool) responsesRequest {
	req := responsesRequest{
		Model:  c.model,
		Input:  input,
		Stream: stream,
	}

	temperature := float64(DefaultTemperature)
	topP := float64(DefaultTopP)
	maxTokens := DefaultMaxTokens
	enableThinking := true

	if opts != nil {
		if temp := opts.GetTemperature(); temp > 0 {
			temperature = temp
		}
		if p := opts.GetTopP(); p > 0 {
			topP = p
		}
		if tokens := opts.GetMaxTokens(); tokens > 0 {
			maxTokens = tokens
		}
		enableThinking = opts.GetThinking()
	}

	req.Temperature = &temperature
	req.TopP = &topP
	req.MaxOutputTokens = &maxTokens
	req.Text = &responsesText{Format: responsesTextFormat{Type: "text"}}

	if enableThinking {
		req.Thinking = &responsesThinking{Type: "enabled"}
	} else {
		req.Thinking = &responsesThinking{Type: "disabled"}
	}

	return req
}

// ==================== HTTP 请求 ====================

func (c *VolcengineChat) doRequest(ctx context.Context, req responsesRequest) (*responsesObject, error) {
	var obj responsesObject

	reqBody, _ := sonic.Marshal(req)
	log.Debug("Responses API 请求", zap.String("body", string(reqBody)))

	resp, err := c.restyClient.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+c.apiKey).
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetResult(&obj).
		Post("/responses")
	if err != nil {
		return nil, fmt.Errorf("HTTP 请求失败: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("API 返回错误 (%d): %s", resp.StatusCode(), resp.String())
	}

	if obj.Error != nil {
		return nil, fmt.Errorf("API 错误 (%s): %s", obj.Error.Code, obj.Error.Message)
	}

	return &obj, nil
}

// ==================== 响应解析 ====================

func getOutputText(resp *responsesObject) string {
	if resp == nil || len(resp.Output) == 0 {
		return ""
	}

	var builder strings.Builder
	for _, item := range resp.Output {
		if item.Type != "message" {
			continue
		}

		var contentItems []contentItem
		if err := sonic.Unmarshal(item.Content, &contentItems); err != nil {
			continue
		}

		for _, ci := range contentItems {
			if ci.Type == "output_text" {
				builder.WriteString(ci.Text)
			}
		}
	}
	return builder.String()
}

// ==================== Input 构建 ====================

func buildInputWithHistory(history []common.ChatMessage, text string) []inputMessage {
	messages := make([]inputMessage, 0, len(history)+1)
	for _, msg := range history {
		messages = append(messages, inputMessage{Type: "message", Role: msg.Role, Content: msg.Content})
	}
	if text != "" {
		messages = append(messages, inputMessage{Type: "message", Role: "user", Content: text})
	}
	return messages
}

// convertToolsForVolcengine converts OpenAI nested tool format to Volcengine flat format
// convertToolsForVolcengine converts Anthropic-format tools to Volcengine flat format
func convertToolsForVolcengine(tools any) []map[string]any {
	// tools from agent is []tool.Definition (Anthropic format)
	src, ok := tools.([]tool.Definition)
	if ok {
		result := make([]map[string]any, 0, len(src))
		for _, t := range src {
			if t.Type == "web_search_20250305" || t.Type == "web_search_20260209" {
				// Convert Anthropic built-in web search to regular function tool
				result = append(result, map[string]any{
					"name":        t.Name,
					"description": t.Description,
					"parameters":  t.InputSchema,
				})
				continue
			}
			result = append(result, map[string]any{
				"type":        t.Type,
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.InputSchema,
			})
		}
		return result
	}
	return nil
}

