package tool

import (
	"fmt"

	"github.com/chuccp/go-web-frame/core"
)

// Registry is an IService that holds tool executors and handler references.
// It replaces the package-level singleton pattern, allowing multiple isolated
// tool registries (e.g., per context or per application instance).
type Registry struct {
	registry     map[string]Executor
	FlowHandler  FlowActionHandler
	ModelHandler ModelActionHandler
}

// NewRegistry creates a new tool registry.
func NewRegistry() *Registry {
	return &Registry{registry: make(map[string]Executor)}
}

func (r *Registry) Init(ctx *core.Context) error {
	r.Register(&ExecuteCommand{})
	r.Register(&ReadDocument{})
	r.Register(&WebSearch{})
	r.Register(&ManageFlows{reg: r})
	r.Register(&ManageModels{reg: r})
	return nil
}

// Register adds an executor to the registry.
func (r *Registry) Register(e Executor) {
	def := e.Definition()
	r.registry[def.Name] = e
}

// Get returns an executor by name.
func (r *Registry) Get(name string) (Executor, error) {
	e, ok := r.registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
	return e, nil
}

// List returns all tool definitions (Anthropic format).
func (r *Registry) List() []Definition {
	defs := make([]Definition, 0, len(r.registry))
	for _, e := range r.registry {
		defs = append(defs, e.Definition())
	}
	return defs
}

// Execute dispatches a tool call and returns the result.
func (r *Registry) Execute(call Call) Result {
	e, err := r.Get(call.Name)
	if err != nil {
		return Result{CallID: call.ID, Output: fmt.Sprintf("error: %v", err)}
	}
	output, err := e.Execute(call)
	if err != nil {
		return Result{CallID: call.ID, Output: fmt.Sprintf("execute failed: %v", err)}
	}
	return Result{CallID: call.ID, Output: output}
}
