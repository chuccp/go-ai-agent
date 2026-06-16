package nodes

import (
	"fmt"
	"strings"

	"github.com/chuccp/go-ai-agent/internal/entity"
	"github.com/chuccp/go-ai-agent/internal/flow/engine"
)

// LoopNodeConfig Loop node config
type LoopNodeConfig struct {
	MaxIterations int    `json:"max_iterations"` // Max iterations, 0=unlimited (max 100)
	BreakField    string `json:"break_field"`    // Break condition check field, e.g. "child.output"
	BreakOperator string `json:"break_operator"` // contains/equals/not_empty
	BreakValue    string `json:"break_value"`    // Break compare value
}

// LoopNode Loop over child nodes until break condition is met or max iterations reached
type LoopNode struct{}

func NewLoopNode() *LoopNode { return &LoopNode{} }

func (n *LoopNode) Type() string { return TypeLoop }

func (n *LoopNode) Execute(ctx *engine.ExecutionContext, config string) (*engine.NodeOutput, error) {
	cfg, err := engine.GetNodeConfig[LoopNodeConfig](config)
	if err != nil {
		return nil, err
	}
	if cfg.MaxIterations <= 0 {
		cfg.MaxIterations = 10 // Default max 10 iterations
	}
	if cfg.MaxIterations > 100 {
		cfg.MaxIterations = 100
	}

	loopId := ctx.CurrentNodeId
	if loopId == 0 {
		return nil, fmt.Errorf("loop: current node id not set in context")
	}

	// Find child nodes by GroupId
	var children []*entity.FlowNode
	for _, node := range ctx.Nodes {
		if node.GroupId != nil && *node.GroupId == loopId {
			children = append(children, node)
		}
	}
	if len(children) == 0 {
		return &engine.NodeOutput{
			Data: map[string]any{
				KeyOutput:    "",
				"iterations": 0,
			},
			Status: engine.StatusSuccess,
		}, nil
	}

	// Find edges wholly inside the loop body
	childIDs := make(map[uint]bool)
	for _, n := range children {
		childIDs[n.Id] = true
	}
	var childEdges []*entity.FlowEdge
	for _, edge := range ctx.Edges {
		if childIDs[edge.SourceNodeId] && childIDs[edge.TargetNodeId] {
			childEdges = append(childEdges, edge)
		}
	}

	// Build sub-engine using the same node registry
	if ctx.Registry == nil {
		return nil, fmt.Errorf("loop: node registry not available in context")
	}

	// Suppress flow-complete events from the sub-engine so the parent flow stays alive.
	subEmitter := &loopEmitter{parent: ctx.Emitter}
	subEngine := engine.NewEngine(ctx.Registry, subEmitter)
	subEngine.LoadFlow(children, childEdges)

	startId, err := findLoopStartNode(children, childEdges)
	if err != nil {
		return nil, fmt.Errorf("loop: %w", err)
	}

	loopCtx := &LoopContext{
		MaxIterations: cfg.MaxIterations,
		BreakField:    cfg.BreakField,
		BreakOperator: cfg.BreakOperator,
		BreakValue:    cfg.BreakValue,
	}

	iterations := 0
	for {
		if ctx.IsAborted() {
			return nil, fmt.Errorf("execution aborted")
		}
		iterations++

		if err := subEngine.Run(ctx, startId); err != nil {
			return nil, fmt.Errorf("loop iteration %d failed: %w", iterations, err)
		}

		if loopCtx.ShouldBreak(ctx) {
			break
		}
	}

	return &engine.NodeOutput{
		Data: map[string]any{
			KeyOutput:    fmt.Sprintf("completed %d iterations", iterations),
			"iterations": iterations,
		},
		Status: engine.StatusSuccess,
	}, nil
}

// findLoopStartNode finds the entry node inside the loop body.
// It prefers an explicit "start" node; otherwise uses the node with no in-body incoming edges.
func findLoopStartNode(children []*entity.FlowNode, edges []*entity.FlowEdge) (uint, error) {
	inDegree := make(map[uint]int)
	for _, n := range children {
		inDegree[n.Id] = 0
	}
	for _, e := range edges {
		inDegree[e.TargetNodeId]++
	}

	for _, n := range children {
		if n.Type == engine.NodeTypeStart {
			return n.Id, nil
		}
	}

	for _, n := range children {
		if inDegree[n.Id] == 0 {
			return n.Id, nil
		}
	}

	if len(children) > 0 {
		return children[0].Id, nil
	}
	return 0, fmt.Errorf("no loop body nodes")
}

// loopEmitter wraps an emitter and suppresses flow-complete events from sub-engines.
type loopEmitter struct {
	parent engine.EventEmitter
}

func (e *loopEmitter) Emit(event engine.FlowEvent) {
	if event.Type == engine.EventFlowComplete {
		return
	}
	if e.parent != nil {
		e.parent.Emit(event)
	}
}

// LoopContext Loop execution context (used by Engine)
type LoopContext struct {
	MaxIterations int
	BreakField    string
	BreakOperator string
	BreakValue    string
	CurrentIter   int
}

// ShouldBreak Check whether loop should break
func (lc *LoopContext) ShouldBreak(ctx *engine.ExecutionContext) bool {
	lc.CurrentIter++
	if lc.CurrentIter >= lc.MaxIterations {
		return true
	}
	if lc.BreakField == "" {
		return false
	}

	field := lc.BreakField
	if !strings.Contains(field, ".") {
		field = field + "." + KeyOutput
	}
	raw, ok := ctx.Get(field)
	if !ok {
		for label, output := range ctx.AllNodeOutputs() {
			if label+"."+KeyOutput == field {
				raw = output.Data[KeyOutput]
				ok = true
				break
			}
		}
	}
	if !ok {
		return false
	}

	fieldVal := fmt.Sprintf("%v", raw)
	return Evaluate(lc.BreakOperator, fieldVal, lc.BreakValue)
}
