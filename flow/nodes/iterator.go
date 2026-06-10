package nodes

import (
	"github.com/bytedance/sonic"
	"fmt"
	"strings"

	"github.com/chuccp/go-ai-agent/flow/engine"
)

// IteratorNodeConfig 按序迭代节点配置
type IteratorNodeConfig struct {
	ItemsKey string `json:"items_key"` // 上游数组键
	Model    string `json:"model"`
	Prompt   string `json:"prompt"` // 每项 prompt，{{item.output}} 引用当前项
	System   string `json:"system"`
}

// IteratorNode 按序迭代——逐项处理，每项等上一项完成再处理下一项
type IteratorNode struct{}

func NewIteratorNode() *IteratorNode { return &IteratorNode{} }

func (n *IteratorNode) Type() string { return "iterator" }

func (n *IteratorNode) Execute(ctx *engine.ExecutionContext, config string) (*engine.NodeOutput, error) {
	cfg, err := engine.GetNodeConfig[IteratorNodeConfig](config)
	if err != nil {
		return nil, err
	}
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}
	if cfg.Prompt == "" {
		cfg.Prompt = "处理以下内容：\n{{item.output}}"
	}

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
		return nil, fmt.Errorf("iterator: key '%s' not found", cfg.ItemsKey)
	}

	items, err := parseJSONArray(raw)
	if err != nil {
		return nil, fmt.Errorf("iterator: parse items failed: %w", err)
	}

	results := make([]map[string]any, 0, len(items))
	var fullOutput strings.Builder

	for i, item := range items {
		if ctx.IsAborted() {
			break
		}

		itemCtx := cloneForItem(ctx, item)
		prompt := renderPrompt(cfg.Prompt, itemCtx)
		system := renderPrompt(cfg.System, itemCtx)

		args := map[string]any{
			"model":      cfg.Model,
			"prompt":     prompt,
			"system":     system,
			"max_tokens": 4096,
			"json_mode":  false,
			"stream":     false,
		}

		if ctx.Emitter != nil {
			ctx.Emitter.Emit(engine.FlowEvent{
				Type:        engine.EventNodeChunk,
				ExecutionId: ctx.ExecutionId,
				Content:     fmt.Sprintf("【%d/%d】处理中...\n", i+1, len(items)),
			})
		}

		result, err := ctx.InvokeFunction("llm", args)
		if err != nil {
			if ctx.Emitter != nil {
				ctx.Emitter.Emit(engine.FlowEvent{
					Type:        engine.EventNodeChunk,
					ExecutionId: ctx.ExecutionId,
					Content:     fmt.Sprintf("【%d/%d】失败: %v\n", i+1, len(items), err),
				})
			}
			continue
		}

		output, _ := result[KeyOutput].(string)
		fullOutput.WriteString(fmt.Sprintf("--- 第 %d 项 ---\n%s\n\n", i+1, output))

		results = append(results, map[string]any{
			"index":  i,
			"item":   item,
			"output": output,
		})

		if ctx.Emitter != nil {
			ctx.Emitter.Emit(engine.FlowEvent{
				Type:        engine.EventNodeChunk,
				ExecutionId: ctx.ExecutionId,
				Content:     fmt.Sprintf("【%d/%d】完成 ✓\n", i+1, len(items)),
			})
		}
	}

	resultJSON, _ := sonic.Marshal(results)
	return &engine.NodeOutput{
		Data: map[string]any{
			KeyOutput: fullOutput.String(),
			"json":    string(resultJSON),
			"count":   len(results),
			"total":   len(items),
		},
		Status: engine.StatusSuccess,
	}, nil
}
