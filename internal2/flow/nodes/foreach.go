package nodes

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/chuccp/go-ai-agent/internal2/flow/engine"
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

	results := make([]map[string]any, len(items))
	var wg sync.WaitGroup
	var firstErr error
	var errMu sync.Mutex

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

		// item is of type any; assert to map[string]any for the goroutine.
		it, ok := item.(map[string]any)
		if !ok {
			// Fallback: wrap scalar items so placeholders still work.
			it = map[string]any{"value": item}
		}

		wg.Add(1)
		go func(idx int, it map[string]any) {
			defer wg.Done()
			itemCtx := cloneForItem(ctx, it)
			renderedArgs := renderArgs(cfg.Args, itemCtx)

			result, err := ctx.InvokeFunction(cfg.Function, renderedArgs)
			if err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("for_each: %s failed for item %d: %w", cfg.Function, idx, err)
				}
				errMu.Unlock()
				return
			}

			output, _ := result[KeyOutput].(string)
			results[idx] = map[string]any{
				"index":  idx,
				"item":   it,
				"output": output,
			}
		}(i, it)
	}

	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	// Filter out nil results (from aborted items)
	validResults := make([]map[string]any, 0, len(results))
	for _, r := range results {
		if r != nil {
			validResults = append(validResults, r)
		}
	}

	resultJSON, _ := json.Marshal(validResults)
	return &engine.NodeOutput{
		Data: map[string]any{
			KeyOutput: string(resultJSON),
			"count":   len(validResults),
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
