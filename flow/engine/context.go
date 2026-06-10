package engine

import (
	"fmt"
	"sync"

	"github.com/chuccp/go-ai-agent/flow/cache"
)

// ExecutionContext 流程执行上下文，在节点间传递数据
type ExecutionContext struct {
	mu              sync.RWMutex
	FlowId          uint
	ExecutionId     uint
	SessionId       uint
	Data            map[string]any         // 全局数据（含 label.field 扁平键）
	NodeOutputs     map[string]*NodeOutput // key = node label
	UserInput       chan string            // 用户输入通道
	Emitter         EventEmitter           // 事件发射器
	Aborted         bool
	Functions       *FunctionRegistry      // 函数注册表
	Cache           *cache.CacheManager    // LLM 结果缓存
}

func NewExecutionContext(flowId, executionId, sessionId uint, emitter EventEmitter) *ExecutionContext {
	return &ExecutionContext{
		FlowId:      flowId,
		ExecutionId: executionId,
		SessionId:   sessionId,
		Data:        make(map[string]any),
		NodeOutputs: make(map[string]*NodeOutput),
		UserInput:   make(chan string, 1),
		Emitter:     emitter,
	}
}

func (c *ExecutionContext) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Data[key] = value
}

func (c *ExecutionContext) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.Data[key]
	return v, ok
}

func (c *ExecutionContext) SetNodeOutput(label string, output *NodeOutput) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.NodeOutputs[label] = output
	for k, v := range output.Data {
		c.Data[label+"."+k] = v
	}
}

func (c *ExecutionContext) GetNodeOutput(label string) (*NodeOutput, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	o, ok := c.NodeOutputs[label]
	return o, ok
}

// AllNodeOutputs 返回所有节点输出的快照（渲染器使用）
func (c *ExecutionContext) AllNodeOutputs() map[string]*NodeOutput {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]*NodeOutput, len(c.NodeOutputs))
	for k, v := range c.NodeOutputs {
		out[k] = v
	}
	return out
}

func (c *ExecutionContext) SeedInput(input string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.NodeOutputs["user_input"] = &NodeOutput{
		Data:   map[string]any{"output": input},
		Status: StatusSuccess,
	}
	c.Data["user_input.output"] = input
}

func (c *ExecutionContext) SendUserInput(input string) {
	select {
	case c.UserInput <- input:
	default:
	}
}

func (c *ExecutionContext) WaitUserInput() (string, error) {
	return <-c.UserInput, nil
}

func (c *ExecutionContext) Abort() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Aborted = true
}

func (c *ExecutionContext) IsAborted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Aborted
}

// InvokeFunction 调用注册的函数
func (c *ExecutionContext) InvokeFunction(name string, args map[string]any) (map[string]any, error) {
	if c.Functions == nil {
		return nil, fmt.Errorf("function registry not initialized")
	}
	return c.Functions.Invoke(c, name, args)
}
