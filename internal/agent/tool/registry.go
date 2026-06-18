package tool

import (
	"fmt"

	"github.com/chuccp/go-web-frame/core"
)

// Registry is an IService that holds tool executors and handler references.
// It replaces the package-level singleton pattern, allowing multiple isolated
// tool registries (e.g., per context or per application instance).
type Registry struct {
	registry             map[string]Executor
	flowHandler          FlowActionHandler
	flowExecutionHandler FlowExecutionHandler
	modelHandler         ModelActionHandler
}

// NewRegistry creates a new tool registry.
func NewRegistry() *Registry {
	return &Registry{registry: make(map[string]Executor)}
}

// SetFlowHandler sets the flow action handler and pushes it to any tool
// that implements FlowHandlerSetter.
func (r *Registry) SetFlowHandler(h FlowActionHandler) {
	r.flowHandler = h
	for _, e := range r.registry {
		if s, ok := e.(FlowHandlerSetter); ok {
			s.SetFlowHandler(h)
		}
	}
}

// SetModelHandler sets the model action handler and pushes it to any tool
// that implements ModelHandlerSetter.
func (r *Registry) SetModelHandler(h ModelActionHandler) {
	r.modelHandler = h
	for _, e := range r.registry {
		if s, ok := e.(ModelHandlerSetter); ok {
			s.SetModelHandler(h)
		}
	}
}

// SetFlowExecutionHandler sets the flow execution handler and pushes it to any tool
// that implements FlowExecutionHandlerSetter.
func (r *Registry) SetFlowExecutionHandler(h FlowExecutionHandler) {
	r.flowExecutionHandler = h
	for _, e := range r.registry {
		if s, ok := e.(FlowExecutionHandlerSetter); ok {
			s.SetFlowExecutionHandler(h)
		}
	}
}

func (r *Registry) Init(ctx *core.Context) error {
	r.Register(&ExecuteCommand{})
	r.Register(&ReadDocument{})
	r.Register(&WebSearch{})
	r.Register(&ManageFlows{})
	r.Register(&RunFlow{})
	r.Register(&CreateFlowConversation{})
	r.Register(&ManageModels{})
	return nil
}

// Register adds an executor to the registry and auto-injects handlers
// if the executor implements FlowHandlerSetter / ModelHandlerSetter / FlowExecutionHandlerSetter.
func (r *Registry) Register(e Executor) {
	def := e.Definition()
	r.registry[def.Name] = e
	if r.flowHandler != nil {
		if s, ok := e.(FlowHandlerSetter); ok {
			s.SetFlowHandler(r.flowHandler)
		}
	}
	if r.flowExecutionHandler != nil {
		if s, ok := e.(FlowExecutionHandlerSetter); ok {
			s.SetFlowExecutionHandler(r.flowExecutionHandler)
		}
	}
	if r.modelHandler != nil {
		if s, ok := e.(ModelHandlerSetter); ok {
			s.SetModelHandler(r.modelHandler)
		}
	}
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
		return Result{CallID: call.ID, Output: fmt.Sprintf("error: %v", err), Success: false, Error: err.Error()}
	}
	output, err := e.Execute(call)
	if err != nil {
		return Result{CallID: call.ID, Output: fmt.Sprintf("execute failed: %v", err), Success: false, Error: err.Error()}
	}
	return Result{CallID: call.ID, Output: output, Success: true}
}
