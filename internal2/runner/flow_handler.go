package runner

import (
	"fmt"

	"encoding/json"
	"github.com/chuccp/go-ai-agent/internal2/entity"
	"github.com/chuccp/go-web-frame/web"
)

// ==================== Flow message handling ====================

func (r *ChatRunner) handleFlowStart(stream *web.WebSocketStream, req WSRequest) {
	if r.flowRunner == nil {
		r.sendJSON(stream, WSResponse{Type: "error", Message: "FlowRunner not initialized"})
		return
	}

	var flowId, executionId uint
	var builtInName string
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
		if v, ok := req.Options["builtin_flow"]; ok {
			builtInName = fmt.Sprintf("%v", v)
		}
	}

	initialInput := ""
	if len(req.Messages) > 0 {
		initialInput = req.Messages[len(req.Messages)-1].Content
	}

	opts := FlowStartOptions{InitialInput: initialInput}
	if req.Options != nil {
		if fv, ok := req.Options["form_values"].(map[string]any); ok {
			opts.FormValues = fv
		}
		if co, ok := req.Options["config_overrides"].(map[string]string); ok {
			opts.ConfigOverrides = co
		}
	}

	var newExecId uint
	var err error
	if builtInName != "" {
		newExecId, err = r.flowRunner.HandleBuiltInFlowStart(builtInName, executionId, req.SessionID, opts)
	} else {
		newExecId, err = r.flowRunner.HandleFlowStart(flowId, executionId, req.SessionID, opts)
	}
	if err != nil {
		r.sendJSON(stream, WSResponse{Type: "error", Message: err.Error()})
	} else if newExecId > 0 {
		r.sendJSON(stream, WSResponse{Type: "flow_started", Message: fmt.Sprintf("flow started: %d", newExecId)})
	}
}

func (r *ChatRunner) handleFlowUserResponse(stream *web.WebSocketStream, req WSRequest) {
	if r.flowRunner == nil {
		r.sendJSON(stream, WSResponse{Type: "error", Message: "FlowRunner not initialized"})
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
		r.sendJSON(stream, WSResponse{Type: "error", Message: err.Error()})
	}
}

func (r *ChatRunner) handleFlowStop(stream *web.WebSocketStream, req WSRequest) {
	if r.flowRunner == nil {
		r.sendJSON(stream, WSResponse{Type: "error", Message: "FlowRunner not initialized"})
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
		r.sendJSON(stream, WSResponse{Type: "error", Message: err.Error()})
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

// flowNodesEdgesResult stores parsed node and edge results
type flowNodesEdgesResult struct {
	nodes    []*entity.FlowNode
	edges    []*entity.FlowEdge
	nodeCnt  int
	edgeCnt  int
	hasNodes bool  // Whether nodes were provided
	err      error // Validation error (returns early, nodes/edges not saved)
}

// parseFlowNodesEdges parses, validates, and returns nodes + edges from args.
// It does NOT save to disk — the caller passes the result to FlowService.
func (r *ChatRunner) parseFlowNodesEdges(flowId uint, args map[string]any) flowNodesEdgesResult {
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

	// Validate required config fields for each node type
	for _, ni := range nodeInputs {
		label := ni.Label
		if label == "" {
			label = ni.Type
		}
		switch ni.Type {
		case "llm":
			prompt, _ := ni.Config["prompt"].(string)
			model, _ := ni.Config["model"].(string)
			if prompt == "" || model == "" {
				res.err = fmt.Errorf("LLM node '%s': prompt and model are required. Use manage_models list to find available models.", label)
				return res
			}
		case "user_input":
			prompt, _ := ni.Config["prompt"].(string)
			if prompt == "" {
				res.err = fmt.Errorf("User_input node '%s': prompt is required (the question shown to the user).", label)
				return res
			}
		case "condition":
			script, _ := ni.Config["script"].(string)
			if script == "" {
				res.err = fmt.Errorf("Condition node '%s': script is required (Starlark expression, assign bool to 'result').", label)
				return res
			}
		case "switch":
			script, _ := ni.Config["script"].(string)
			if script == "" {
				res.err = fmt.Errorf("Switch node '%s': script is required (Starlark expression, assign string to 'result').", label)
				return res
			}
		case "for_each", "iterator":
			itemsKey, _ := ni.Config["items_key"].(string)
			if itemsKey == "" {
				res.err = fmt.Errorf("%s node '%s': items_key is required (which upstream output contains the list).", ni.Type, label)
				return res
			}
		case "loop":
			maxIter, _ := ni.Config["max_iterations"].(float64)
			if maxIter <= 0 {
				res.err = fmt.Errorf("Loop node '%s': max_iterations is required (safety limit).", label)
				return res
			}
		case "script":
			script, _ := ni.Config["script"].(string)
			if script == "" {
				res.err = fmt.Errorf("Script node '%s': script (code) is required.", label)
				return res
			}
		case "execute":
			command, _ := ni.Config["command"].(string)
			if command == "" {
				res.err = fmt.Errorf("Execute node '%s': command is required.", label)
				return res
			}
		case "image_gen", "video_gen":
			prompt, _ := ni.Config["prompt"].(string)
			if prompt == "" {
				res.err = fmt.Errorf("%s node '%s': prompt is required.", ni.Type, label)
				return res
			}
		case "audio_gen":
			text, _ := ni.Config["text"].(string)
			model, _ := ni.Config["model"].(string)
			if text == "" || model == "" {
				res.err = fmt.Errorf("Audio_gen node '%s': text and model are required.", label)
				return res
			}
		}
	}

	// Build nodes with sequential IDs (not saved to DB)
	res.nodes = make([]*entity.FlowNode, len(nodeInputs))
	for i, ni := range nodeInputs {
		cfgBytes, _ := json.Marshal(ni.Config)
		res.nodes[i] = &entity.FlowNode{
			Id:        uint(i + 1),
			FlowId:    flowId,
			Type:      ni.Type,
			Label:     ni.Label,
			Config:    string(cfgBytes),
			PositionX: float64(100 + (i%4)*300),
			PositionY: float64(100 + (i/4)*200),
		}
		res.nodeCnt++
	}

	// Build edges, mapping source/target indices to sequential node IDs
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
				if ei.SourceIndex < 0 || ei.SourceIndex >= len(res.nodes) ||
					ei.TargetIndex < 0 || ei.TargetIndex >= len(res.nodes) {
					continue
				}
				handle := ei.SourceHandle
				if handle == "" {
					handle = "output"
				}
				res.edges = append(res.edges, &entity.FlowEdge{
					FlowId:       flowId,
					SourceNodeId: res.nodes[ei.SourceIndex].Id,
					TargetNodeId: res.nodes[ei.TargetIndex].Id,
					SourceHandle: handle,
					TargetHandle: "input",
					Label:        ei.Label,
				})
				res.edgeCnt++
			}
		}
	}
	return res
}
