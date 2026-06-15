package engine

import (
	"fmt"
	"sync"

	"github.com/chuccp/go-ai-agent/internal/flow/cache"
)

// ExecutionContext Flow execution context, passes data between nodes
type ExecutionContext struct {
	mu              sync.RWMutex
	FlowId          uint
	ExecutionId     uint
	SessionId       uint
	Data            map[string]any         // Global data (with flat label.field keys)
	NodeOutputs     map[string]*NodeOutput // key = node label
	Config          map[string]string      // Package-level runtime config
	FormValues      map[string]any         // Flow form values (app mode)
	UserInput       chan string            // User input channel
	Emitter         EventEmitter           // Event emitter
	Aborted         bool
	Functions       *FunctionRegistry      // Function registry
	Cache           *cache.CacheManager    // LLM result cache
}

func NewExecutionContext(flowId, executionId, sessionId uint, emitter EventEmitter) *ExecutionContext {
	return &ExecutionContext{
		FlowId:      flowId,
		ExecutionId: executionId,
		SessionId:   sessionId,
		Data:        make(map[string]any),
		NodeOutputs: make(map[string]*NodeOutput),
		Config:      make(map[string]string),
		FormValues:  make(map[string]any),
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

// AllNodeOutputs Returns snapshot of all node outputs (used by renderer)
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

// SetConfig replaces the runtime config map.
func (c *ExecutionContext) SetConfig(cfg map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Config = cfg
	c.Data["config"] = cfg
	for k, v := range cfg {
		c.Data["config."+k] = v
	}
}

// SetFormValues replaces the form values and exposes them as Data.
func (c *ExecutionContext) SetFormValues(values map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.FormValues = values
	c.Data["form"] = values
	for k, v := range values {
		c.Data["form."+k] = v
	}
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

// InvokeFunction Invoke registered function
func (c *ExecutionContext) InvokeFunction(name string, args map[string]any) (map[string]any, error) {
	if c.Functions == nil {
		return nil, fmt.Errorf("function registry not initialized")
	}
	return c.Functions.Invoke(c, name, args)
}
