package engine

import "fmt"

// Registry Node type registry
type Registry struct {
	executors map[string]NodeExecutor
}

// NewRegistry Create registry
func NewRegistry() *Registry {
	return &Registry{
		executors: make(map[string]NodeExecutor),
	}
}

// Register Register node executor
func (r *Registry) Register(executor NodeExecutor) {
	r.executors[executor.Type()] = executor
}

// Get Get node executor
func (r *Registry) Get(nodeType string) (NodeExecutor, error) {
	e, ok := r.executors[nodeType]
	if !ok {
		return nil, fmt.Errorf("unknown node type: %s", nodeType)
	}
	return e, nil
}

// Types List all registered node types
func (r *Registry) Types() []string {
	types := make([]string, 0, len(r.executors))
	for t := range r.executors {
		types = append(types, t)
	}
	return types
}
