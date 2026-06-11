package nodes

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chuccp/go-ai-agent/flow/engine"
)

// SplitNodeConfig 文本拆分节点配置
type SplitNodeConfig struct {
	SourceKey string `json:"source_key"` // 上游数据键，如 "生成故事"
	Delimiter string `json:"delimiter"`  // 分隔符：\n\n(段落), \n(按行), 或自定义正则
}

// SplitNode 将上游文本拆分为 JSON 数组，供 ForEach 使用
type SplitNode struct{}

func NewSplitNode() *SplitNode { return &SplitNode{} }

func (n *SplitNode) Type() string { return "split" }

func (n *SplitNode) Execute(ctx *engine.ExecutionContext, config string) (*engine.NodeOutput, error) {
	cfg, err := engine.GetNodeConfig[SplitNodeConfig](config)
	if err != nil {
		return nil, err
	}

	// 自动补全 key
	key := cfg.SourceKey
	if !strings.Contains(key, ".") {
		key = key + "." + KeyOutput
	}

	raw, ok := ctx.Get(key)
	if !ok {
		// fallback: 遍历 NodeOutputs
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

	// 选择分隔符
	delim := cfg.Delimiter
	switch delim {
	case "paragraph", "":
		// 按空行 / 双换行拆分
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
			// fallback: 全文作为一个 item
			items = append(items, text)
		}
		result, _ := json.Marshal(items)
		return &engine.NodeOutput{
			Data:   map[string]any{KeyOutput: string(result), "count": len(items)},
			Status: engine.StatusSuccess,
		}, nil

	case "line":
		// 按行拆分（去掉空行）
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
		// 自定义分隔符
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
