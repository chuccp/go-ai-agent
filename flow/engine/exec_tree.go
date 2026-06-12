package engine

import (
	"fmt"

	"github.com/chuccp/go-ai-agent/entity"
)

// ExecLayer Execution layer (nodes in same layer can run in parallel)
type ExecLayer struct {
	NodeIDs []uint
}

// BuildExecutionLayers Build layered execution plan from nodes and edges
// Use forward BFS from start node, layered by dependencies
func BuildExecutionLayers(nodes []*entity.FlowNode, edges []*entity.FlowEdge) ([]ExecLayer, error) {
	// In-degree table: nodeID -> set of source node IDs
	inDegree := make(map[uint]map[uint]bool)
	// Out-edge table: nodeID -> list of target node IDs
	outEdges := make(map[uint][]uint)

	for _, n := range nodes {
		inDegree[n.Id] = make(map[uint]bool)
	}

	for _, e := range edges {
		outEdges[e.SourceNodeId] = append(outEdges[e.SourceNodeId], e.TargetNodeId)
		inDegree[e.TargetNodeId][e.SourceNodeId] = true
	}

	// Find start node
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
				// Check if all in-edge sources of target have been visited
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

		// Deduplicate
		seen := make(map[uint]bool)
		deduped := make([]uint, 0, len(nextLayer))
		for _, id := range nextLayer {
			if !seen[id] {
				seen[id] = true
				deduped = append(deduped, id)
			}
		}
		currentLayer = deduped

		// Detect cycles: if nodes remain unvisited but nextLayer is empty
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
