package runner

import (
	"encoding/json"
	"fmt"
)

// handleFlowExecutionAction serves the run_flow agent tool.
func (r *ChatRunner) handleFlowExecutionAction(action string, args map[string]any) (string, error) {
	switch action {
	case "search":
		return r.flowSearch(args)
	case "run":
		return r.flowRun(args)
	case "respond":
		return r.flowRespond(args)
	case "status":
		return r.flowStatus(args)
	case "stop":
		return r.flowStop(args)
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

func (r *ChatRunner) flowRun(args map[string]any) (string, error) {
	if r.flowRunner == nil {
		return "", fmt.Errorf("FlowRunner not initialized")
	}

	// If no flow_id provided, search by query first.
	var flowId uint
	if v, ok := args["flow_id"]; ok {
		flowId = uintArg(v)
	}
	if flowId == 0 {
		query, _ := args["query"].(string)
		if query == "" {
			return "", fmt.Errorf("flow_id or query is required")
		}
		res, err := r.flowSearch(args)
		if err != nil {
			return "", err
		}
		return res, nil
	}

	initialInput, _ := args["initial_input"].(string)
	var formValues map[string]any
	if fv, ok := args["form_values"].(map[string]any); ok {
		formValues = fv
	}

	opts := FlowStartOptions{
		InitialInput: initialInput,
		FormValues:   formValues,
	}

	execId, err := r.flowRunner.HandleFlowStart(flowId, 0, 0, opts)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Flow started. Execution ID: %d", execId), nil
}

func (r *ChatRunner) flowRespond(args map[string]any) (string, error) {
	if r.flowRunner == nil {
		return "", fmt.Errorf("FlowRunner not initialized")
	}
	executionId := uintArg(args["execution_id"])
	response := getStringArg(args, "response")
	if executionId == 0 {
		return "", fmt.Errorf("execution_id is required")
	}
	if response == "" {
		return "", fmt.Errorf("response is required")
	}
	if err := r.flowRunner.HandleUserResponse(executionId, response); err != nil {
		return "", err
	}
	return "Response sent", nil
}

func (r *ChatRunner) flowStatus(args map[string]any) (string, error) {
	if r.flowRunner == nil {
		return "", fmt.Errorf("FlowRunner not initialized")
	}
	executionId := uintArg(args["execution_id"])
	if executionId == 0 {
		return "", fmt.Errorf("execution_id is required")
	}
	exec, err := r.flowRunner.GetExecutionStatus(executionId)
	if err != nil {
		return "", fmt.Errorf("execution not found: %d", executionId)
	}
	data := map[string]any{
		"execution_id": executionId,
		"flow_id":      exec.FlowId,
		"status":       exec.Status,
		"context":      exec.Context,
	}
	b, _ := json.Marshal(data)
	return string(b), nil
}

func (r *ChatRunner) flowStop(args map[string]any) (string, error) {
	if r.flowRunner == nil {
		return "", fmt.Errorf("FlowRunner not initialized")
	}
	executionId := uintArg(args["execution_id"])
	if executionId == 0 {
		return "", fmt.Errorf("execution_id is required")
	}
	if err := r.flowRunner.HandleFlowStop(executionId); err != nil {
		return "", err
	}
	return "Execution stopped", nil
}

func uintArg(v any) uint {
	switch n := v.(type) {
	case float64:
		return uint(n)
	case int:
		return uint(n)
	case int64:
		return uint(n)
	case uint:
		return n
	}
	return 0
}
