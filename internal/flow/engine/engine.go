package engine

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/chuccp/go-ai-agent/internal/entity"
	"github.com/chuccp/go-web-frame/log"
	"go.uber.org/zap"
)

type Engine struct {
	registry     *Registry
	emitter      EventEmitter
	nodes        map[uint]*entity.FlowNode
	edges        []*entity.FlowEdge
	adjacencyOut map[uint][]*entity.FlowEdge
	adjacencyIn  map[uint][]*entity.FlowEdge // In-edge table
	taskMgr      *TaskManager
}

func NewEngine(registry *Registry, emitter EventEmitter) *Engine {
	return &Engine{
		registry:     registry,
		emitter:      emitter,
		nodes:        make(map[uint]*entity.FlowNode),
		edges:        make([]*entity.FlowEdge, 0),
		adjacencyOut: make(map[uint][]*entity.FlowEdge),
		adjacencyIn:  make(map[uint][]*entity.FlowEdge),
	}
}

func (e *Engine) SetTaskManager(tm *TaskManager) {
	e.taskMgr = tm
}

func (e *Engine) LoadFlow(nodes []*entity.FlowNode, edges []*entity.FlowEdge) {
	for _, n := range nodes {
		e.nodes[n.Id] = n
	}
	e.edges = edges
	for _, edge := range edges {
		e.adjacencyOut[edge.SourceNodeId] = append(e.adjacencyOut[edge.SourceNodeId], edge)
		e.adjacencyIn[edge.TargetNodeId] = append(e.adjacencyIn[edge.TargetNodeId], edge)
	}
}

// isModelNode Check if node is model-related type
func isModelNode(nodeType string) bool {
	switch nodeType {
	case "llm", "image_gen", "audio_gen", "video_gen":
		return true
	}
	return false
}

func (e *Engine) Run(ctx *ExecutionContext, startNodeId uint) error {
	layers, err := BuildExecutionLayers(e.nodeList(), e.edges, startNodeId)
	if err != nil {
		return err
	}

	// Conditional routing results：conditionNodeID → NextNode value
	conditionRoutes := make(map[uint]string)
	// Skipped nodes
	skippedNodes := make(map[uint]bool)

	for _, layer := range layers {
		if ctx.IsAborted() {
			return fmt.Errorf("execution aborted")
		}

		// Determine whether each node in current layer should execute
		var activeNodes []uint
		for _, id := range layer.NodeIDs {
			if e.shouldSkipNode(id, conditionRoutes, skippedNodes) {
				skippedNodes[id] = true
				continue
			}
			activeNodes = append(activeNodes, id)
		}

		if len(activeNodes) == 0 {
			continue
		}

		if err := e.executeLayer(ctx, activeNodes); err != nil {
			return err
		}

		// Record router node (condition/switch) routing results
		for _, id := range activeNodes {
			node := e.nodes[id]
			if node.Type == "condition" || node.Type == "switch" {
				if output, ok := ctx.GetNodeOutput(node.Label); ok && output.NextNode != "" {
					conditionRoutes[id] = output.NextNode
				}
			}
		}
	}

	e.emitter.Emit(FlowEvent{
		Type:        EventFlowComplete,
		ExecutionId: ctx.ExecutionId,
	})
	return nil
}

// shouldSkipNode Check if node should be skipped
// Skip condition: at least one in-edge from an active path, do not skip
// Active path = source node not skipped AND (source is not a condition node OR source condition route matches this edge's SourceHandle)
func (e *Engine) shouldSkipNode(nodeID uint, conditionRoutes map[uint]string, skippedNodes map[uint]bool) bool {
	if skippedNodes[nodeID] {
		return true
	}

	inEdges := e.adjacencyIn[nodeID]
	// No in-edges (start node): do not skip
	if len(inEdges) == 0 {
		return false
	}

	// Check if at least one in-edge comes from active path
	for _, edge := range inEdges {
		srcID := edge.SourceNodeId
		// Source was skipped → this path is invalid
		if skippedNodes[srcID] {
			continue
		}
		// Source is a condition node → check route match
		if nextNode, isCond := conditionRoutes[srcID]; isCond {
			if edge.SourceHandle == nextNode {
				return false // Matching branch → active path → do not skip
			}
			continue // Non-matching branch → this path is invalid
		}
		// Non-condition source and not skipped → active path
		return false
	}

	// All in-edges from invalid paths → skip
	return true
}

// executeLayer Dispatch by node type: model nodes via TaskManager, others direct
func (e *Engine) executeLayer(ctx *ExecutionContext, nodeIDs []uint) error {
	var modelNodes []uint
	var directNodes []uint

	for _, id := range nodeIDs {
		if n, ok := e.nodes[id]; ok && isModelNode(n.Type) {
			modelNodes = append(modelNodes, id)
		} else {
			directNodes = append(directNodes, id)
		}
	}

	// Execute non-model nodes first (lightweight, direct call)
	for _, id := range directNodes {
		if err := e.executeNode(ctx, id); err != nil {
			return err
		}
	}

	// Model nodes: submit via TaskManager one by one
	for _, id := range modelNodes {
		nodeID := id
		if e.taskMgr != nil {
			if err := e.taskMgr.Submit([]TaskFunc{func() error {
				return e.executeNode(ctx, nodeID)
			}}); err != nil {
				return err
			}
		} else {
			if err := e.executeNode(ctx, nodeID); err != nil {
				return err
			}
		}
	}

	return nil
}

func (e *Engine) executeNode(ctx *ExecutionContext, nodeID uint) error {
	node, ok := e.nodes[nodeID]
	if !ok {
		return fmt.Errorf("node %d not found", nodeID)
	}

	log.Info("Flow engine executing node",
		zap.Uint("nodeId", node.Id),
		zap.String("label", node.Label),
		zap.String("type", node.Type))

	e.emitter.Emit(FlowEvent{
		Type:        EventNodeStart,
		ExecutionId: ctx.ExecutionId,
		NodeId:      node.Id,
		NodeLabel:   node.Label,
		NodeType:    node.Type,
	})

	start := time.Now()
	executor, err := e.registry.Get(node.Type)
	if err != nil {
		e.emitter.Emit(FlowEvent{
			Type:        EventFlowError,
			ExecutionId: ctx.ExecutionId,
			NodeId:      node.Id,
			Message:     err.Error(),
		})
		return err
	}

	ctx.CurrentNodeId = node.Id
	ctx.CurrentNodeLabel = node.Label
	ctx.CurrentNodeType = node.Type

	output, err := executor.Execute(ctx, node.Config)
	duration := time.Since(start).Milliseconds()
	if err != nil {
		e.emitter.Emit(FlowEvent{
			Type:        EventFlowError,
			ExecutionId: ctx.ExecutionId,
			NodeId:      node.Id,
			Message:     err.Error(),
		})
		return err
	}

	ctx.SetNodeOutput(node.Label, output)

	e.emitter.Emit(FlowEvent{
		Type:        EventNodeDone,
		ExecutionId: ctx.ExecutionId,
		NodeId:      node.Id,
		NodeLabel:   node.Label,
		Status:      output.Status,
		Content:     fmt.Sprintf("%dms", duration),
	})

	return nil
}

func (e *Engine) nodeList() []*entity.FlowNode {
	list := make([]*entity.FlowNode, 0, len(e.nodes))
	for _, n := range e.nodes {
		list = append(list, n)
	}
	return list
}

// NodeTypeStart Entry node type needed by engine
const NodeTypeStart = "start"

// FindStartNode Find start node
func (e *Engine) FindStartNode() (uint, error) {
	for _, n := range e.nodes {
		if n.Type == NodeTypeStart {
			return n.Id, nil
		}
	}
	return 0, fmt.Errorf("no start node found in flow")
}

func GetNodeConfig[T any](config string) (T, error) {
	var cfg T
	if err := json.Unmarshal([]byte(config), &cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse node config: %w", err)
	}
	return cfg, nil
}
