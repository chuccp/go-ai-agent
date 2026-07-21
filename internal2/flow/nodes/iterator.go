package nodes

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chuccp/go-ai-agent/internal2/flow/engine"
)

// IteratorNodeConfig is the configuration for an iterator node.
// Iterates over an array from ctx[items_key] and invokes function with args for each item sequentially.
// Args support {{item.field}} and {{item.output}} placeholders.
type IteratorNodeConfig struct {
	ItemsKey string         `json:"items_key"` // upstream array key
	Function string         `json:"function"`  // function name to invoke per item, e.g. "llm"
	Args     map[string]any `json:"args"`      // args passed to the function
}

// IteratorNode processes items sequentially — each item waits for the previous one to complete.
// Failures are skipped, processing continues with the next item.
type IteratorNode struct{}

func NewIteratorNode() *IteratorNode { return &IteratorNode{} }

func (n *IteratorNode) Type() string { return "iterator" }

func (n *IteratorNode) Execute(ctx *engine.ExecutionContext, config string) (*engine.NodeOutput, error) {
	cfg, err := engine.GetNodeConfig[IteratorNodeConfig](config)
	if err != nil {
		return nil, err
	}
	if cfg.ItemsKey == "" {
		return nil, fmt.Errorf("iterator: items_key is required")
	}
	if cfg.Function == "" {
		cfg.Function = "llm"
	}

	items, err := resolveItems(ctx, cfg.ItemsKey)
	if err != nil {
		return nil, err
	}

	var results []map[string]any
	var fullOutput strings.Builder

	for i, item := range items {
		if ctx.IsAborted() {
			break
		}

		itemCtx := cloneForItem(ctx, item)
		renderedArgs := renderArgs(cfg.Args, itemCtx)

		if ctx.Emitter != nil {
			ctx.Emitter.Emit(engine.FlowEvent{
				Type:        engine.EventNodeChunk,
				ExecutionId: ctx.ExecutionId,
				Content:     fmt.Sprintf("[%d/%d] Processing...\n", i+1, len(items)),
			})
		}

		result, err := ctx.InvokeFunction(cfg.Function, renderedArgs)
		if err != nil {
			if ctx.Emitter != nil {
				ctx.Emitter.Emit(engine.FlowEvent{
					Type:        engine.EventNodeChunk,
					ExecutionId: ctx.ExecutionId,
					Content:     fmt.Sprintf("[%d/%d] Failed: %v\n", i+1, len(items), err),
				})
			}
			continue
		}

		output, _ := result[KeyOutput].(string)
		fullOutput.WriteString(fmt.Sprintf("--- Item %d ---\n%s\n\n", i+1, output))

		results = append(results, map[string]any{
			"index":  i,
			"item":   item,
			"output": output,
		})

		if ctx.Emitter != nil {
			ctx.Emitter.Emit(engine.FlowEvent{
				Type:        engine.EventNodeChunk,
				ExecutionId: ctx.ExecutionId,
				Content:     fmt.Sprintf("[%d/%d] Done\n", i+1, len(items)),
			})
		}
	}

	resultJSON, _ := json.Marshal(results)
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
