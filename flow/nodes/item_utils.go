package nodes

import (
	"encoding/json"
	"fmt"

	"github.com/chuccp/go-ai-agent/flow/engine"
)

// parseJSONArray converts a value (string, []any, or other) into a []any slice.
func parseJSONArray(v any) ([]any, error) {
	switch arr := v.(type) {
	case []any:
		return arr, nil
	case string:
		var parsed []any
		if err := json.Unmarshal([]byte(arr), &parsed); err != nil {
			return nil, err
		}
		return parsed, nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		var parsed []any
		if err := json.Unmarshal(b, &parsed); err != nil {
			return nil, err
		}
		return parsed, nil
	}
}

// cloneForItem creates a shallow clone of the execution context with the current item
// injected as the "item" node output (with .output and ._json fields).
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

	itemJSON, _ := json.Marshal(item)
	data["_json"] = string(itemJSON)

	clone.SetNodeOutput("item", &engine.NodeOutput{
		Data:   data,
		Status: engine.StatusSuccess,
	})
	return clone
}
