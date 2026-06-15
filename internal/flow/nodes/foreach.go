package nodes

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chuccp/go-ai-agent/internal/flow/engine"
)

// ForEachNodeConfig is the configuration for a for_each node.
// Iterates over an array from ctx[items_key] and invokes function with args for each item.
// Args support {{item.field}} and {{item.output}} placeholders.
type ForEachNodeConfig struct {
	ItemsKey string         `json:"items_key"` // upstream data key, e.g. "split" or "split.output"
	Function string         `json:"function"`  // function name to invoke per item, e.g. "llm"
	Args     map[string]any `json:"args"`      // args passed to the function
}

// ForEachNode processes array items in parallel, invoking a function for each.
// Each item runs independently — suitable for stateless operations.
type ForEachNode struct{}

func NewForEachNode() *ForEachNode { return &ForEachNode{} }

func (n *ForEachNode) Type() string { return TypeForEach }

func (n *ForEachNode) Execute(ctx *engine.ExecutionContext, config string) (*engine.NodeOutput, error) {
	cfg, err := engine.GetNodeConfig[ForEachNodeConfig](config)
	if err != nil {
		return nil, err
	}
	if cfg.ItemsKey == "" {
		return nil, fmt.Errorf("for_each: items_key is required")
	}
	if cfg.Function == "" {
		cfg.Function = "llm"
	}

	items, err := resolveItems(ctx, cfg.ItemsKey)
	if err != nil {
		return nil, err
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
				Content:     fmt.Sprintf("Processing item %d/%d...\n", i+1, len(items)),
			})
		}

		itemCtx := cloneForItem(ctx, item)
		renderedArgs := renderArgs(cfg.Args, itemCtx)

		result, err := ctx.InvokeFunction(cfg.Function, renderedArgs)
		if err != nil {
			return nil, fmt.Errorf("for_each: %s failed for item %d: %w", cfg.Function, i, err)
		}

		output, _ := result[KeyOutput].(string)
		results = append(results, map[string]any{
			"index":  i,
			"item":   item,
			"output": output,
		})
	}

	resultJSON, _ := json.Marshal(results)
	return &engine.NodeOutput{
		Data: map[string]any{
			KeyOutput: string(resultJSON),
			"count":   len(results),
		},
		Status: engine.StatusSuccess,
	}, nil
}

// resolveItems extracts an array from the context by key.
func resolveItems(ctx *engine.ExecutionContext, key string) ([]any, error) {
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
		return nil, fmt.Errorf("for_each: key '%s' not found", key)
	}

	items, err := parseJSONArray(raw)
	if err != nil {
		return nil, fmt.Errorf("for_each: failed to parse items as JSON array: %w", err)
	}
	return items, nil
}

// renderArgs renders {{item.field}} placeholders in arg values.
func renderArgs(args map[string]any, ctx *engine.ExecutionContext) map[string]any {
	result := make(map[string]any, len(args))
	for k, v := range args {
		switch val := v.(type) {
		case string:
			result[k] = renderPrompt(val, ctx)
		default:
			result[k] = v
		}
	}
	return result
}
