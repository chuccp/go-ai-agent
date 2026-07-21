package engine

import (
	"fmt"
	"sync"

	"github.com/chuccp/go-ai-agent/internal2/entity"
	"github.com/chuccp/go-ai-agent/internal2/flow/cache"
)

// ExecutionContext Flow execution context, passes data between nodes
type ExecutionContext struct {
	mu               sync.RWMutex
	FlowId           uint
	ExecutionId      uint
	SessionId        uint
	Data             map[string]any         // Global data (with flat label.field keys)
	NodeOutputs      map[string]*NodeOutput // key = node label
	Config           map[string]string      // Package-level runtime config
	FormValues       map[string]any         // Flow form values (app mode)
	Emitter          EventEmitter           // Event emitter
	Aborted          bool
	Functions        *FunctionRegistry   // Function registry
	Cache            *cache.CacheManager // LLM result cache
	Registry         *Registry           // Node executor registry (used by container nodes like loop)
	Nodes            []*entity.FlowNode  // All nodes in current flow (for container/loop lookup)
	Edges            []*entity.FlowEdge  // All edges in current flow
	CurrentNodeId    uint                // Node currently being executed
	CurrentNodeLabel string              // Label of current node
	CurrentNodeType  string              // Type of current node
	WaitingPrompt    string              // Prompt shown when waiting for user input

	// User input mechanism (replaces channel to prevent lost inputs)
	userInputMu    sync.Mutex
	userInputCond  *sync.Cond
	userInputValue string
	userInputReady bool
}

func NewExecutionContext(flowId, executionId, sessionId uint, emitter EventEmitter) *ExecutionContext {
	ctx := &ExecutionContext{
		FlowId:      flowId,
		ExecutionId: executionId,
		SessionId:   sessionId,
		Data:        make(map[string]any),
		NodeOutputs: make(map[string]*NodeOutput),
		Config:      make(map[string]string),
		FormValues:  make(map[string]any),
		Emitter:     emitter,
	}
	ctx.userInputCond = sync.NewCond(&ctx.userInputMu)
	return ctx
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
	c.userInputMu.Lock()
	c.userInputValue = input
	c.userInputReady = true
	c.userInputCond.Signal()
	c.userInputMu.Unlock()
}

func (c *ExecutionContext) WaitUserInput() (string, error) {
	c.userInputMu.Lock()
	for !c.userInputReady {
		c.userInputCond.Wait()
	}
	val := c.userInputValue
	c.userInputReady = false
	c.userInputMu.Unlock()
	return val, nil
}

func (c *ExecutionContext) SetCurrentNode(id uint, label, nodeType string) {
	c.mu.Lock()
	c.CurrentNodeId = id
	c.CurrentNodeLabel = label
	c.CurrentNodeType = nodeType
	c.mu.Unlock()
}

func (c *ExecutionContext) GetCurrentNode() (uint, string, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.CurrentNodeId, c.CurrentNodeLabel, c.CurrentNodeType
}

func (c *ExecutionContext) SetWaitingPrompt(prompt string) {
	c.mu.Lock()
	c.WaitingPrompt = prompt
	c.mu.Unlock()
}

func (c *ExecutionContext) GetWaitingPrompt() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.WaitingPrompt
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

// CloneForItem returns a shallow copy of the context for parallel item processing.
// Shared registries and emitter are copied by reference; Data and NodeOutputs maps
// are duplicated so concurrent per-item writes stay isolated.
func (c *ExecutionContext) CloneForItem() *ExecutionContext {
	c.mu.RLock()
	defer c.mu.RUnlock()

	clone := &ExecutionContext{
		FlowId:           c.FlowId,
		ExecutionId:      c.ExecutionId,
		SessionId:        c.SessionId,
		Data:             make(map[string]any, len(c.Data)),
		NodeOutputs:      make(map[string]*NodeOutput, len(c.NodeOutputs)),
		Config:           c.Config,
		FormValues:       c.FormValues,
		Emitter:          c.Emitter,
		Aborted:          c.Aborted,
		Functions:        c.Functions,
		Cache:            c.Cache,
		Registry:         c.Registry,
		Nodes:            c.Nodes,
		Edges:            c.Edges,
		CurrentNodeId:    c.CurrentNodeId,
		CurrentNodeLabel: c.CurrentNodeLabel,
		CurrentNodeType:  c.CurrentNodeType,
		WaitingPrompt:    c.WaitingPrompt,
	}
	for k, v := range c.Data {
		clone.Data[k] = v
	}
	for k, v := range c.NodeOutputs {
		clone.NodeOutputs[k] = v
	}
	clone.userInputCond = sync.NewCond(&clone.userInputMu)
	return clone
}
