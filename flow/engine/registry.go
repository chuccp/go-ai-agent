package engine

import "fmt"

// Registry 节点类型注册表
type Registry struct {
	executors map[string]NodeExecutor
}

// NewRegistry 创建注册表
func NewRegistry() *Registry {
	return &Registry{
		executors: make(map[string]NodeExecutor),
	}
}

// Register 注册节点执行器
func (r *Registry) Register(executor NodeExecutor) {
	r.executors[executor.Type()] = executor
}

// Get 获取节点执行器
func (r *Registry) Get(nodeType string) (NodeExecutor, error) {
	e, ok := r.executors[nodeType]
	if !ok {
		return nil, fmt.Errorf("unknown node type: %s", nodeType)
	}
	return e, nil
}

// Types 列出所有已注册的节点类型
func (r *Registry) Types() []string {
	types := make([]string, 0, len(r.executors))
	for t := range r.executors {
		types = append(types, t)
	}
	return types
}
