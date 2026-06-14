package tool

// Definition is a tool definition in Anthropic standard format
type Definition struct {
	Type        string `json:"type,omitempty"`        // Only needed for built-in tools, e.g. "web_search_20250305"
	Name        string `json:"name"`                  // Function name
	Description string `json:"description,omitempty"` // Description
	InputSchema any    `json:"input_schema,omitempty"` // JSON Schema (custom tools)
}

// Call is a tool call request
type Call struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string of input
}

// Result is the result of executing a tool
type Result struct {
	CallID string `json:"call_id"`
	Output string `json:"output"`
}

// Executor is the interface for tool executors
type Executor interface {
	Execute(call Call) (string, error)
	Definition() Definition
}
