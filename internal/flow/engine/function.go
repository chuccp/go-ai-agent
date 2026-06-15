package engine

import (
	"fmt"
	"sync"
)

// FunctionHandler Function handler
type FunctionHandler func(ctx *ExecutionContext, name string, args map[string]any) (map[string]any, error)

// FunctionRegistry Function registry
type FunctionRegistry struct {
	mu        sync.RWMutex
	functions map[string]FunctionHandler
}

// NewFunctionRegistry creates a function registry
func NewFunctionRegistry() *FunctionRegistry {
	return &FunctionRegistry{
		functions: make(map[string]FunctionHandler),
	}
}

// Register Register function
func (r *FunctionRegistry) Register(name string, handler FunctionHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.functions[name] = handler
}

// Invoke Invoke function
func (r *FunctionRegistry) Invoke(ctx *ExecutionContext, name string, args map[string]any) (map[string]any, error) {
	r.mu.RLock()
	handler, ok := r.functions[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("function not found: %s", name)
	}
	return handler(ctx, name, args)
}

// Has Check if function exists
func (r *FunctionRegistry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.functions[name]
	return ok
}

// ListFunctions lists all registered functions
func (r *FunctionRegistry) ListFunctions() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.functions))
	for name := range r.functions {
		names = append(names, name)
	}
	return names
}
