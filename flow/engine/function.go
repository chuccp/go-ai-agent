package engine

import (
	"fmt"
	"sync"
)

// FunctionHandler 函数处理器
type FunctionHandler func(ctx *ExecutionContext, name string, args map[string]any) (map[string]any, error)

// FunctionRegistry 函数注册表
type FunctionRegistry struct {
	mu        sync.RWMutex
	functions map[string]FunctionHandler
}

// NewFunctionRegistry 创建函数注册表
func NewFunctionRegistry() *FunctionRegistry {
	return &FunctionRegistry{
		functions: make(map[string]FunctionHandler),
	}
}

// Register 注册函数
func (r *FunctionRegistry) Register(name string, handler FunctionHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.functions[name] = handler
}

// Invoke 调用函数
func (r *FunctionRegistry) Invoke(ctx *ExecutionContext, name string, args map[string]any) (map[string]any, error) {
	r.mu.RLock()
	handler, ok := r.functions[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("function not found: %s", name)
	}
	return handler(ctx, name, args)
}

// Has 检查函数是否存在
func (r *FunctionRegistry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.functions[name]
	return ok
}

// ListFunctions 列出所有已注册函数
func (r *FunctionRegistry) ListFunctions() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.functions))
	for name := range r.functions {
		names = append(names, name)
	}
	return names
}
