package nodes

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/chuccp/go-ai-agent/internal/ai/chat/common"
	"github.com/chuccp/go-ai-agent/internal/flow/cache"
	"github.com/chuccp/go-ai-agent/internal/flow/engine"
	"github.com/chuccp/go-ai-agent/internal/flow/out"
)

type LLMNodeConfig struct {
	Model         string  `json:"model"`
	Prompt        string  `json:"prompt"`
	System        string  `json:"system"`
	History       string  `json:"history"`
	Temperature   float64 `json:"temperature"`
	TopP          float64 `json:"top_p"`
	MaxTokens     int     `json:"max_tokens"`
	ThinkingLevel string  `json:"thinking_level"`
	OutputFormat  string  `json:"output_format"`
	CacheEnabled  *bool   `json:"cache_enabled"`
}

func (c *LLMNodeConfig) UnmarshalJSON(data []byte) error {
	type Alias LLMNodeConfig
	aux := &struct{ *Alias }{Alias: (*Alias)(c)}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if c.Prompt == "" {
		if v, ok := raw["prompt_template"].(string); ok {
			c.Prompt = v
		}
	}
	if c.System == "" {
		if v, ok := raw["system_prompt"].(string); ok {
			c.System = v
		}
	}
	return nil
}

type LLMNode struct{}

func NewLLMNode() *LLMNode { return &LLMNode{} }

func (n *LLMNode) Type() string { return TypeLLM }

func (n *LLMNode) Execute(ctx *engine.ExecutionContext, config string) (*engine.NodeOutput, error) {
	cfg, err := engine.GetNodeConfig[LLMNodeConfig](config)
	if err != nil {
		return nil, err
	}
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 4096
	}

	prompt := renderPrompt(cfg.Prompt, ctx)
	system := renderPrompt(cfg.System, ctx)

	// Parse history: render template, then unmarshal JSON string into []ChatMessage
	var chatHistory []common.ChatMessage
	historyJSON := renderPrompt(cfg.History, ctx)
	if historyJSON != "" {
		if err := json.Unmarshal([]byte(historyJSON), &chatHistory); err != nil {
			return nil, fmt.Errorf("llm: failed to parse history JSON: %w", err)
		}
	}

	// Structured output
	if cfg.OutputFormat != "" {
		var of out.OutFormat
		if json.Unmarshal([]byte(cfg.OutputFormat), &of) == nil {
			if inst := of.BuildPromptInstruction(); inst != "" {
				if system != "" {
					system += "\n\n"
				}
				system += inst
			}
		}
	}

	// Cache check
	cacheEnabled := isCacheEnabled(ctx, cfg.CacheEnabled)
	if cacheEnabled && ctx.Cache != nil {
		historyKey := ""
		if len(chatHistory) > 0 {
			if b, err := json.Marshal(chatHistory); err == nil {
				historyKey = string(b)
			}
		}
		cacheKey := cache.GenerateKey(cfg.Model, system, prompt, historyKey, strconv.Itoa(cfg.MaxTokens),
			fmt.Sprintf("%f", cfg.Temperature), fmt.Sprintf("%f", cfg.TopP), cfg.ThinkingLevel)
		if cached, ok := ctx.Cache.Get(cacheKey); ok {
			if ctx.Emitter != nil {
				ctx.Emitter.Emit(engine.FlowEvent{
					Type:        engine.EventNodeChunk,
					ExecutionId: ctx.ExecutionId,
					Content:     cached,
				})
			}
			return &engine.NodeOutput{
				Data:   map[string]any{KeyOutput: cached, KeyPrompt: prompt},
				Status: engine.StatusSuccess,
			}, nil
		}
	}

	// Call LLM via function registry
	args := map[string]any{
		"model":          cfg.Model,
		"prompt":         prompt,
		"system":         system,
		"history":        chatHistory,
		"max_tokens":     cfg.MaxTokens,
		"json_mode":      cfg.OutputFormat != "",
		"stream":         true,
		"temperature":    cfg.Temperature,
		"top_p":          cfg.TopP,
		"thinking_level": cfg.ThinkingLevel,
	}

	result, err := ctx.InvokeFunction("llm", args)
	if err != nil {
		return nil, err
	}

	output, _ := result[KeyOutput].(string)

	// Write cache
	if cacheEnabled && ctx.Cache != nil {
		historyKey := ""
		if len(chatHistory) > 0 {
			if b, err := json.Marshal(chatHistory); err == nil {
				historyKey = string(b)
			}
		}
		cacheKey := cache.GenerateKey(cfg.Model, system, prompt, historyKey, strconv.Itoa(cfg.MaxTokens),
			fmt.Sprintf("%f", cfg.Temperature), fmt.Sprintf("%f", cfg.TopP), cfg.ThinkingLevel)
		_ = ctx.Cache.Set(cacheKey, output)
	}

	return &engine.NodeOutput{
		Data:   map[string]any{KeyOutput: output, KeyPrompt: prompt},
		Status: engine.StatusSuccess,
	}, nil
}

func isCacheEnabled(ctx *engine.ExecutionContext, cfgVal *bool) bool {
	if cfgVal != nil {
		return *cfgVal
	}
	return ctx.Cache != nil && ctx.Cache.IsEnabled()
}
