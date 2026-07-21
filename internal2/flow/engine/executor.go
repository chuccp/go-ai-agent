package engine

// NodeExecutor Node executor interface, each node type implements this
type NodeExecutor interface {
	// Type Returns node type name (corresponds to FlowNode.Type)
	Type() string
	// Execute Execute node logic
	// ctx: execution context with upstream node outputs
	// config: node configuration JSON string
	Execute(ctx *ExecutionContext, config string) (*NodeOutput, error)
}

// NodeOutput Node execution output
type NodeOutput struct {
	Data     map[string]any `json:"data"`      // Node output data (stored in context)
	NextNode string         `json:"next_node"` // Specify next node label (empty means follow edges)
	Status   string         `json:"status"`    // "success", "error", "waiting_user"
	Message  string         `json:"message"`   // Prompt message for the user
}
