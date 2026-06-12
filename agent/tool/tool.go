package tool

import "fmt"

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

var registry = make(map[string]Executor)

func Register(e Executor) {
	def := e.Definition()
	registry[def.Name] = e
}

func Get(name string) (Executor, error) {
	e, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
	return e, nil
}

// List returns all tool definitions (Anthropic format)
func List() []Definition {
	defs := make([]Definition, 0, len(registry))
	for _, e := range registry {
		defs = append(defs, e.Definition())
	}
	return defs
}

func Execute(call Call) Result {
	e, err := Get(call.Name)
	if err != nil {
		return Result{CallID: call.ID, Output: fmt.Sprintf("error: %v", err)}
	}
	output, err := e.Execute(call)
	if err != nil {
		return Result{CallID: call.ID, Output: fmt.Sprintf("execute failed: %v", err)}
	}
	return Result{CallID: call.ID, Output: output}
}
