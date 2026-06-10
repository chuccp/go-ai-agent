package nodes

import (
	"fmt"
	"strings"

	"github.com/chuccp/go-ai-agent/flow/engine"
)

// ConditionNodeConfig 条件分支节点配置
type ConditionNodeConfig struct {
	Field    string `json:"field"`    // 要检查的上下文键，如 "确认故事.output"
	Operator string `json:"operator"` // contains / equals / not_empty / is_json
	Value    string `json:"value"`    // 比较值
}

// ConditionNode 条件分支 — 根据数据决定走 true 还是 false 出口
// 匹配时走 source_handle="true" 的边，否则走 "false"
type ConditionNode struct{}

func NewConditionNode() *ConditionNode { return &ConditionNode{} }

func (n *ConditionNode) Type() string { return "condition" }

func (n *ConditionNode) Execute(ctx *engine.ExecutionContext, config string) (*engine.NodeOutput, error) {
	cfg, err := engine.GetNodeConfig[ConditionNodeConfig](config)
	if err != nil {
		return nil, err
	}
	if cfg.Field == "" {
		return nil, fmt.Errorf("condition: field is required")
	}

	// 自动补全
	field := cfg.Field
	if !strings.Contains(field, ".") {
		field = field + "." + KeyOutput
	}

	raw, ok := ctx.Get(field)
	if !ok {
		for label, output := range ctx.AllNodeOutputs() {
			if label+"."+KeyOutput == field {
				raw = output.Data[KeyOutput]
				ok = true
				break
			}
		}
	}

	fieldVal := ""
	if ok {
		fieldVal = fmt.Sprintf("%v", raw)
	}

	result := evaluate(cfg.Operator, fieldVal, cfg.Value)

	nextNode := "false"
	if result {
		nextNode = "true"
	}

	return &engine.NodeOutput{
		Data: map[string]any{
			KeyOutput: fmt.Sprintf("%v", result),
			"field":   fieldVal,
			"result":  result,
		},
		Status:   engine.StatusSuccess,
		NextNode: nextNode,
	}, nil
}

func evaluate(op, fieldVal, compareVal string) bool {
	switch op {
	case "contains":
		return strings.Contains(fieldVal, compareVal)
	case "equals":
		return strings.TrimSpace(fieldVal) == strings.TrimSpace(compareVal)
	case "not_empty":
		return strings.TrimSpace(fieldVal) != ""
	case "is_json":
		return strings.HasPrefix(strings.TrimSpace(fieldVal), "{") || strings.HasPrefix(strings.TrimSpace(fieldVal), "[")
	case "confirmed":
		// 用户确认场景：检查是否回复了"确认"/"是"/"ok"等
		v := strings.TrimSpace(fieldVal)
		return v == "确认" || v == "是" || v == "ok" || v == "yes" || v == "y"
	default:
		return strings.Contains(fieldVal, compareVal)
	}
}
