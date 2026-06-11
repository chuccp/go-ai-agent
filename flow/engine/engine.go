package engine

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/chuccp/go-ai-agent/entity"
	"github.com/chuccp/go-web-frame/log"
	"go.uber.org/zap"
)

type Engine struct {
	registry     *Registry
	emitter      EventEmitter
	nodes        map[uint]*entity.FlowNode
	edges        []*entity.FlowEdge
	adjacencyOut map[uint][]*entity.FlowEdge
	adjacencyIn  map[uint][]*entity.FlowEdge // 入边表
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

// isModelNode 判断节点是否为模型相关类型
func isModelNode(nodeType string) bool {
	switch nodeType {
	case "llm", "image_gen", "audio_gen", "video_gen":
		return true
	}
	return false
}

func (e *Engine) Run(ctx *ExecutionContext, startNodeId uint) error {
	layers, err := BuildExecutionLayers(e.nodeList(), e.edges)
	if err != nil {
		return err
	}

	// 条件路由结果：conditionNodeID → NextNode value
	conditionRoutes := make(map[uint]string)
	// 被跳过的节点
	skippedNodes := make(map[uint]bool)

	for _, layer := range layers {
		if ctx.IsAborted() {
			return fmt.Errorf("execution aborted")
		}

		// 判断当前层每个节点是否应该执行
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

		// 记录条件节点的路由结果
		for _, id := range activeNodes {
			node := e.nodes[id]
			if node.Type == "condition" {
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

// shouldSkipNode 判断节点是否应该跳过
// 跳过条件：至少有一条入边来自活跃路径，则不跳过
// 活跃路径 = 入边来源节点未被跳过 且（来源不是条件节点 或 来源条件节点的路由匹配该边的 SourceHandle）
func (e *Engine) shouldSkipNode(nodeID uint, conditionRoutes map[uint]string, skippedNodes map[uint]bool) bool {
	if skippedNodes[nodeID] {
		return true
	}

	inEdges := e.adjacencyIn[nodeID]
	// 无入边（start 节点）：不跳过
	if len(inEdges) == 0 {
		return false
	}

	// 检查是否至少有一条入边来自活跃路径
	for _, edge := range inEdges {
		srcID := edge.SourceNodeId
		// 来源被跳过 → 这条路径无效
		if skippedNodes[srcID] {
			continue
		}
		// 来源是条件节点 → 检查路由是否匹配
		if nextNode, isCond := conditionRoutes[srcID]; isCond {
			if edge.SourceHandle == nextNode {
				return false // 匹配的分支 → 活跃路径 → 不跳过
			}
			continue // 不匹配的分支 → 这条路径无效
		}
		// 非条件来源且未被跳过 → 活跃路径
		return false
	}

	// 所有入边都来自无效路径 → 跳过
	return true
}

// executeLayer 按节点类型分发执行：模型节点走 TaskManager，其余直接执行
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

	// 先执行非模型节点（轻量，直接调用）
	for _, id := range directNodes {
		if err := e.executeNode(ctx, id); err != nil {
			return err
		}
	}

	// 模型节点：通过 TaskManager 逐个提交执行
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

// NodeTypeStart 引擎需要的入口节点类型
const NodeTypeStart = "start"

// FindStartNode 查找起始节点
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
