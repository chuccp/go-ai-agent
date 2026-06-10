package nodes

import (
	"github.com/bytedance/sonic"
	"fmt"
	"strings"

	"github.com/chuccp/go-ai-agent/flow/engine"
)

type ForEachNodeConfig struct {
	Model     string `json:"model"`
	ItemsKey  string `json:"items_key"` // 上游数据键，如 "分段" 或 "分段.output"
	Prompt    string `json:"prompt"`
	System    string `json:"system"`
	MaxTokens int    `json:"max_tokens"`
	JSONMode  bool   `json:"json_mode"`
}

type ForEachNode struct{}

func NewForEachNode() *ForEachNode { return &ForEachNode{} }

func (n *ForEachNode) Type() string { return TypeForEach }

func (n *ForEachNode) Execute(ctx *engine.ExecutionContext, config string) (*engine.NodeOutput, error) {
	cfg, err := engine.GetNodeConfig[ForEachNodeConfig](config)
	if err != nil {
		return nil, err
	}
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 4096
	}
	if cfg.Prompt == "" {
		cfg.Prompt = "处理以下内容：\n{{item.output}}"
	}

	// 自动补全：items_key 不含 "." 则补 ".output"
	key := cfg.ItemsKey
	if !strings.Contains(key, ".") {
		key = key + "." + KeyOutput
	}

	raw, ok := ctx.Get(key)
	if !ok {
		for label, output := range ctx.AllNodeOutputs() {
			if label+"."+KeyOutput == key {
				raw = output.Data[KeyOutput]
				ok = true
				break
			}
		}
	}
	if !ok {
		return nil, fmt.Errorf("for_each: key '%s' not found (resolved to '%s')", cfg.ItemsKey, key)
	}

	items, err := parseJSONArray(raw)
	if err != nil {
		return nil, fmt.Errorf("for_each: failed to parse items as JSON array: %w", err)
	}

	results := make([]map[string]any, 0, len(items))
	for i, item := range items {
		if ctx.IsAborted() {
			break
		}

		if ctx.Emitter != nil {
			ctx.Emitter.Emit(engine.FlowEvent{
				Type:        engine.EventNodeChunk,
				ExecutionId: ctx.ExecutionId,
				Content:     fmt.Sprintf("处理第 %d/%d 项...\n", i+1, len(items)),
			})
		}

		itemCtx := cloneForItem(ctx, item)

		prompt := renderPrompt(cfg.Prompt, itemCtx)
		system := renderPrompt(cfg.System, itemCtx)

		args := map[string]any{
			"model":      cfg.Model,
			"prompt":     prompt,
			"system":     system,
			"max_tokens": cfg.MaxTokens,
			"json_mode":  cfg.JSONMode,
			"stream":     false,
		}

		result, err := ctx.InvokeFunction("llm", args)
		if err != nil {
			return nil, fmt.Errorf("for_each: LLM call failed for item %d: %w", i, err)
		}

		output, _ := result[KeyOutput].(string)
		results = append(results, map[string]any{
			"index":  i,
			"item":   item,
			"output": output,
		})
	}

	resultJSON, _ := sonic.Marshal(results)
	return &engine.NodeOutput{
		Data: map[string]any{
			KeyOutput: string(resultJSON),
			KeyPrompt: cfg.Prompt,
			"count":   len(results),
		},
		Status: engine.StatusSuccess,
	}, nil
}

func parseJSONArray(v any) ([]any, error) {
	switch arr := v.(type) {
	case []any:
		return arr, nil
	case string:
		var parsed []any
		if err := sonic.Unmarshal([]byte(arr), &parsed); err != nil {
			return nil, err
		}
		return parsed, nil
	default:
		b, err := sonic.Marshal(v)
		if err != nil {
			return nil, err
		}
		var parsed []any
		if err := sonic.Unmarshal(b, &parsed); err != nil {
			return nil, err
		}
		return parsed, nil
	}
}

func cloneForItem(ctx *engine.ExecutionContext, item any) *engine.ExecutionContext {
	clone := engine.NewExecutionContext(ctx.FlowId, ctx.ExecutionId, ctx.SessionId, ctx.Emitter)
	clone.Functions = ctx.Functions
	clone.Cache = ctx.Cache
	for k, v := range ctx.AllNodeOutputs() {
		clone.SetNodeOutput(k, v)
	}

	data := map[string]any{}
	switch v := item.(type) {
	case string:
		data[KeyOutput] = v
	case map[string]any:
		for k, val := range v {
			data[k] = val
		}
	default:
		data[KeyOutput] = fmt.Sprintf("%v", v)
	}

	itemJSON, _ := sonic.Marshal(item)
	data["_json"] = string(itemJSON)

	clone.SetNodeOutput("item", &engine.NodeOutput{
		Data:   data,
		Status: engine.StatusSuccess,
	})
	return clone
}
