package runner

import (
	"fmt"

	"encoding/json"
	"github.com/chuccp/go-ai-agent/entity"
	"github.com/gorilla/websocket"
)

// ==================== Flow message handling ====================

func (r *ChatRunner) handleFlowStart(conn *websocket.Conn, req WSRequest) {
	if r.flowRunner == nil {
		r.sendJSON(conn, WSResponse{Type: "error", Message: "FlowRunner not initialized"})
		return
	}

	var flowId, executionId uint
	if req.Options != nil {
		if v, ok := req.Options["flow_id"]; ok {
			if vv, ok := v.(float64); ok {
				flowId = uint(vv)
			}
		}
		if v, ok := req.Options["execution_id"]; ok {
			if vv, ok := v.(float64); ok {
				executionId = uint(vv)
			}
		}
	}

	initialInput := ""
	if len(req.Messages) > 0 {
		initialInput = req.Messages[len(req.Messages)-1].Content
	}

	if err := r.flowRunner.HandleFlowStart(flowId, executionId, req.SessionID, initialInput); err != nil {
		r.sendJSON(conn, WSResponse{Type: "error", Message: err.Error()})
	}
}

func (r *ChatRunner) handleFlowUserResponse(conn *websocket.Conn, req WSRequest) {
	if r.flowRunner == nil {
		r.sendJSON(conn, WSResponse{Type: "error", Message: "FlowRunner not initialized"})
		return
	}

	var executionId uint
	response := ""
	if req.Options != nil {
		if v, ok := req.Options["execution_id"]; ok {
			if vv, ok := v.(float64); ok {
				executionId = uint(vv)
			}
		}
		if v, ok := req.Options["response"]; ok {
			response = fmt.Sprintf("%v", v)
		}
	}

	if err := r.flowRunner.HandleUserResponse(executionId, response); err != nil {
		r.sendJSON(conn, WSResponse{Type: "error", Message: err.Error()})
	}
}

func (r *ChatRunner) handleFlowStop(conn *websocket.Conn, req WSRequest) {
	if r.flowRunner == nil {
		r.sendJSON(conn, WSResponse{Type: "error", Message: "FlowRunner not initialized"})
		return
	}

	var executionId uint
	if req.Options != nil {
		if v, ok := req.Options["execution_id"]; ok {
			if vv, ok := v.(float64); ok {
				executionId = uint(vv)
			}
		}
	}

	if err := r.flowRunner.HandleFlowStop(executionId); err != nil {
		r.sendJSON(conn, WSResponse{Type: "error", Message: err.Error()})
	}
}

// ==================== Flow management tool handling ====================

func (r *ChatRunner) handleFlowAction(action string, args map[string]any) (string, error) {
	switch action {
	case "list":
		return r.flowList()
	case "search":
		return r.flowSearch(args)
	case "get":
		return r.flowGet(args)
	case "create":
		return r.flowCreate(args)
	case "update":
		return r.flowUpdate(args)
	case "delete":
		return r.flowDelete(args)
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

// flowNodesEdgesResult stores node and edge results
type flowNodesEdgesResult struct {
	nodeIDs  []uint // Ordered list of node IDs
	nodeCnt  int
	edgeCnt  int
	hasNodes bool // Whether nodes were provided
}

// saveFlowNodesEdges parses and saves nodes + edges (shared by create/update)
func (r *ChatRunner) saveFlowNodesEdges(flowId uint, args map[string]any) flowNodesEdgesResult {
	var res flowNodesEdgesResult
	nodesRaw, ok := args["nodes"]
	if !ok {
		return res
	}
	res.hasNodes = true
	nodesBytes, _ := json.Marshal(nodesRaw)
	var nodeInputs []struct {
		Type   string         `json:"type"`
		Label  string         `json:"label"`
		Config map[string]any `json:"config"`
	}
	if json.Unmarshal(nodesBytes, &nodeInputs) != nil {
		return res
	}
	if len(nodeInputs) == 0 {
		return res
	}

	res.nodeIDs = make([]uint, len(nodeInputs))
	for i, ni := range nodeInputs {
		cfgBytes, _ := json.Marshal(ni.Config)
		node := &entity.FlowNode{
			FlowId:    flowId,
			Type:      ni.Type,
			Label:     ni.Label,
			Config:    string(cfgBytes),
			PositionX: float64(100 + (i%4)*300),
			PositionY: float64(100 + (i/4)*200),
		}
		if err := r.nodeModel.Create(node); err == nil {
			res.nodeIDs[i] = node.Id
			res.nodeCnt++
		}
	}

	if edgesRaw, ok := args["edges"]; ok {
		edgesBytes, _ := json.Marshal(edgesRaw)
		var edgeInputs []struct {
			SourceIndex  int    `json:"source_index"`
			TargetIndex  int    `json:"target_index"`
			SourceHandle string `json:"source_handle"`
			Label        string `json:"label"`
		}
		if json.Unmarshal(edgesBytes, &edgeInputs) == nil {
			for _, ei := range edgeInputs {
				if ei.SourceIndex < 0 || ei.SourceIndex >= len(res.nodeIDs) ||
					ei.TargetIndex < 0 || ei.TargetIndex >= len(res.nodeIDs) {
					continue
				}
				handle := ei.SourceHandle
				if handle == "" {
					handle = "output"
				}
				edge := &entity.FlowEdge{
					FlowId:       flowId,
					SourceNodeId: res.nodeIDs[ei.SourceIndex],
					TargetNodeId: res.nodeIDs[ei.TargetIndex],
					SourceHandle: handle,
					TargetHandle: "input",
					Label:        ei.Label,
				}
				if err := r.edgeModel.Create(edge); err == nil {
					res.edgeCnt++
				}
			}
		}
	}
	return res
}
