package tool

import "fmt"

// Definition 工具定义（Anthropic 标准格式）
type Definition struct {
	Type        string `json:"type,omitempty"`        // 仅内置工具需要，如 "web_search_20250305"
	Name        string `json:"name"`                  // 函数名
	Description string `json:"description,omitempty"` // 描述
	InputSchema any    `json:"input_schema,omitempty"` // JSON Schema（自定义工具）
}

// Call 工具调用请求
type Call struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string of input
}

// Result 工具执行结果
type Result struct {
	CallID string `json:"call_id"`
	Output string `json:"output"`
}

// Executor 工具执行器接口
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

// List 列出所有工具定义（Anthropic 格式）
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
