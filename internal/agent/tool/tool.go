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

// FlowActionHandler handles flow CRUD operations (injected by runner)
type FlowActionHandler func(action string, args map[string]any) (string, error)

// FlowExecutionHandler handles flow execution operations (injected by runner)
type FlowExecutionHandler func(action string, args map[string]any) (string, error)

// ModelActionHandler handles AI model CRUD operations (injected by runner)
type ModelActionHandler func(action string, params map[string]any) (string, error)

// FlowHandlerSetter is implemented by tools that need a flow action handler.
// Registry.Register auto-injects the handler via this interface.
type FlowHandlerSetter interface {
	SetFlowHandler(handler FlowActionHandler)
}

// FlowExecutionHandlerSetter is implemented by tools that need a flow execution handler.
type FlowExecutionHandlerSetter interface {
	SetFlowExecutionHandler(handler FlowExecutionHandler)
}

// SkillHandlerSetter is implemented by tools that need the skill service.
type SkillHandlerSetter interface {
	SetSkillService(svc any)
}

// ModelHandlerSetter is implemented by tools that need a model action handler.
// Registry.Register auto-injects the handler via this interface.
type ModelHandlerSetter interface {
	SetModelHandler(handler ModelActionHandler)
}
