package engine

// NodeExecutor 节点执行器接口，每种节点类型实现此接口
type NodeExecutor interface {
	// Type 返回节点类型名称（与 FlowNode.Type 对应）
	Type() string
	// Execute 执行节点逻辑
	// ctx: 执行上下文，包含上游节点输出
	// config: 节点配置 JSON 字符串
	Execute(ctx *ExecutionContext, config string) (*NodeOutput, error)
}

// NodeOutput 节点执行输出
type NodeOutput struct {
	Data     map[string]any `json:"data"`      // 节点输出数据（存入上下文）
	NextNode string         `json:"next_node"` // 指定下一个节点标签（为空则按边遍历）
	Status   string         `json:"status"`    // "success", "error", "waiting_user"
	Message  string         `json:"message"`   // 给用户看的提示消息
}
