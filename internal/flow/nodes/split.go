package nodes

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chuccp/go-ai-agent/internal/flow/engine"
)

// SplitNodeConfig Text split node config
type SplitNodeConfig struct {
	SourceKey string `json:"source_key"` // Upstream data key, e.g. "generate_story"
	Delimiter string `json:"delimiter"`  // Delimiter: \n\n(paragraph), \n(by line), or custom regex
}

// SplitNode Split upstream text into JSON array for ForEach
type SplitNode struct{}

func NewSplitNode() *SplitNode { return &SplitNode{} }

func (n *SplitNode) Type() string { return "split" }

func (n *SplitNode) Execute(ctx *engine.ExecutionContext, config string) (*engine.NodeOutput, error) {
	cfg, err := engine.GetNodeConfig[SplitNodeConfig](config)
	if err != nil {
		return nil, err
	}

	// Auto-complete key
	key := cfg.SourceKey
	if !strings.Contains(key, ".") {
		key = key + "." + KeyOutput
	}

	raw, ok := ctx.Get(key)
	if !ok {
		// fallback: iterate NodeOutputs
		for label, output := range ctx.AllNodeOutputs() {
			if label+"."+KeyOutput == key {
				raw = output.Data[KeyOutput]
				ok = true
				break
			}
		}
	}
	if !ok {
		return nil, fmt.Errorf("split: source key '%s' not found", cfg.SourceKey)
	}

	text, ok := raw.(string)
	if !ok {
		return nil, fmt.Errorf("split: source value is not a string")
	}

	// Select delimiter
	delim := cfg.Delimiter
	switch delim {
	case "paragraph", "":
		// Split by empty lines / double newlines
		text = strings.ReplaceAll(text, "\r\n\r\n", "\n\n")
		text = strings.ReplaceAll(text, "\r\n", "\n")
		parts := strings.Split(text, "\n\n")
		items := make([]any, 0)
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				items = append(items, p)
			}
		}
		if len(items) == 0 {
			// fallback: entire text as single item
			items = append(items, text)
		}
		result, _ := json.Marshal(items)
		return &engine.NodeOutput{
			Data:   map[string]any{KeyOutput: string(result), "count": len(items)},
			Status: engine.StatusSuccess,
		}, nil

	case "line":
		// Split by line (skip empty lines)
		lines := strings.Split(text, "\n")
		items := make([]any, 0)
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				items = append(items, line)
			}
		}
		result, _ := json.Marshal(items)
		return &engine.NodeOutput{
			Data:   map[string]any{KeyOutput: string(result), "count": len(items)},
			Status: engine.StatusSuccess,
		}, nil

	default:
		// Custom delimiter
		parts := strings.Split(text, delim)
		items := make([]any, 0)
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				items = append(items, p)
			}
		}
		result, _ := json.Marshal(items)
		return &engine.NodeOutput{
			Data:   map[string]any{KeyOutput: string(result), "count": len(items)},
			Status: engine.StatusSuccess,
		}, nil
	}
}
