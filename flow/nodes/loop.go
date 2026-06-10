package nodes

import (
	"fmt"
	"strings"

	"github.com/chuccp/go-ai-agent/flow/engine"
)

// LoopNodeConfig 循环节点配置
type LoopNodeConfig struct {
	MaxIterations int    `json:"max_iterations"` // 最大循环次数，0=不限(最多100)
	BreakField    string `json:"break_field"`    // 中断条件检查字段，如 "子节点.output"
	BreakOperator string `json:"break_operator"` // contains/equals/not_empty
	BreakValue    string `json:"break_value"`    // 中断比较值
}

// LoopNode 循环执行子节点直到满足中断条件或达到最大次数
type LoopNode struct{}

func NewLoopNode() *LoopNode { return &LoopNode{} }

func (n *LoopNode) Type() string { return "loop" }

func (n *LoopNode) Execute(ctx *engine.ExecutionContext, config string) (*engine.NodeOutput, error) {
	cfg, err := engine.GetNodeConfig[LoopNodeConfig](config)
	if err != nil {
		return nil, err
	}
	if cfg.MaxIterations <= 0 {
		cfg.MaxIterations = 10 // 默认最多10次
	}
	if cfg.MaxIterations > 100 {
		cfg.MaxIterations = 100
	}

	// 需要 Engine 引用才能运行子流程
	// Loop 节点依赖 engine.runSubFlow —— 由 Engine.Run 在检测到 children 时调用
	// 这里只返回占位输出，实际循环由 Engine 处理
	return &engine.NodeOutput{
		Data: map[string]any{
			KeyOutput:       "",
			"max_iterations": cfg.MaxIterations,
		},
		Status: engine.StatusSuccess,
	}, nil
}

// LoopContext 循环执行上下文（由 Engine 使用）
type LoopContext struct {
	MaxIterations int
	BreakField    string
	BreakOperator string
	BreakValue    string
	CurrentIter   int
}

// ShouldBreak 检查是否应该中断循环
func (lc *LoopContext) ShouldBreak(ctx *engine.ExecutionContext) bool {
	lc.CurrentIter++
	if lc.CurrentIter >= lc.MaxIterations {
		return true
	}
	if lc.BreakField == "" {
		return false
	}

	field := lc.BreakField
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
	if !ok {
		return false
	}

	fieldVal := fmt.Sprintf("%v", raw)
	return evaluate(lc.BreakOperator, fieldVal, lc.BreakValue)
}
