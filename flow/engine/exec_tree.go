package engine

import (
	"fmt"

	"github.com/chuccp/go-ai-agent/entity"
)

// ExecLayer 执行层（同层节点可并行）
type ExecLayer struct {
	NodeIDs []uint
}

// BuildExecutionLayers 从节点和边构建分层执行计划
// 使用正向 BFS：从 start 节点出发，按依赖关系分层
func BuildExecutionLayers(nodes []*entity.FlowNode, edges []*entity.FlowEdge) ([]ExecLayer, error) {
	// 入度表：nodeID -> 入边来源节点 ID 集合
	inDegree := make(map[uint]map[uint]bool)
	// 出边表：nodeID -> 目标节点 ID 列表
	outEdges := make(map[uint][]uint)

	for _, n := range nodes {
		inDegree[n.Id] = make(map[uint]bool)
	}

	for _, e := range edges {
		outEdges[e.SourceNodeId] = append(outEdges[e.SourceNodeId], e.TargetNodeId)
		inDegree[e.TargetNodeId][e.SourceNodeId] = true
	}

	// 找到 start 节点
	var startID uint
	for _, n := range nodes {
		if n.Type == "start" {
			startID = n.Id
			break
		}
	}
	if startID == 0 {
		return nil, fmt.Errorf("start node not found")
	}

	var layers []ExecLayer
	visited := make(map[uint]bool)
	currentLayer := []uint{startID}

	for len(currentLayer) > 0 {
		layers = append(layers, ExecLayer{NodeIDs: currentLayer})
		for _, id := range currentLayer {
			visited[id] = true
		}

		var nextLayer []uint
		for _, id := range currentLayer {
			for _, targetID := range outEdges[id] {
				if visited[targetID] {
					continue
				}
				// 检查 target 的所有入边来源是否都已访问
				allVisited := true
				for srcID := range inDegree[targetID] {
					if !visited[srcID] {
						allVisited = false
						break
					}
				}
				if allVisited {
					nextLayer = append(nextLayer, targetID)
				}
			}
		}

		// 去重
		seen := make(map[uint]bool)
		deduped := make([]uint, 0, len(nextLayer))
		for _, id := range nextLayer {
			if !seen[id] {
				seen[id] = true
				deduped = append(deduped, id)
			}
		}
		currentLayer = deduped

		// 检测循环：如果还有未访问节点但 nextLayer 为空
		if len(currentLayer) == 0 {
			unvisited := 0
			for _, n := range nodes {
				if !visited[n.Id] {
					unvisited++
				}
			}
			if unvisited > 0 {
				return nil, fmt.Errorf("circular dependency detected, %d nodes unreachable", unvisited)
			}
		}
	}

	return layers, nil
}
