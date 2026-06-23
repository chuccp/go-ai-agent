package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chuccp/go-ai-agent/internal/agent/tool"
)

// handleFlowExecutionAction serves the run_flow agent tool.
// The context enables cancellation (e.g. user clicks stop or disconnects).
func (r *ChatRunner) handleFlowExecutionAction(ctx context.Context, action string, args map[string]any) (string, error) {
	switch action {
	case "search":
		return r.flowSearch(args)
	case "run":
		return r.flowRun(ctx, args)
	case "respond":
		return r.flowRespond(ctx, args)
	case "status":
		return r.flowStatus(args)
	case "stop":
		return r.flowStop(args)
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

func (r *ChatRunner) flowRun(ctx context.Context, args map[string]any) (string, error) {
	if r.flowRunner == nil {
		return "", fmt.Errorf("FlowRunner not initialized")
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

	// Extract session ID from context (embedded by agent.NewChat via tool.WithSessionID)
	// so flow events are enriched with the correct session_id for frontend routing.
	sessionID := tool.SessionIDFrom(ctx)

	// Built-in flow takes precedence.
	if builtin, ok := args["builtin_flow"].(string); ok && builtin != "" {
		execId, err := r.flowRunner.HandleBuiltInFlowStart(builtin, 0, sessionID, opts)
		if err != nil {
			return "", err
		}
		return r.waitForFlowCompletion(ctx, execId)
	}

	// If no flow_id provided, search by query first.
	var flowId uint
	if v, ok := args["flow_id"]; ok {
		flowId = uintArg(v)
	}
	if flowId == 0 {
		query, _ := args["query"].(string)
		if query == "" {
			return "", fmt.Errorf("flow_id, query, or builtin_flow is required")
		}
		res, err := r.flowSearch(args)
		if err != nil {
			return "", err
		}
		return res, nil
	}

	execId, err := r.flowRunner.HandleFlowStart(flowId, 0, sessionID, opts)
	if err != nil {
		return "", err
	}

	return r.waitForFlowCompletion(ctx, execId)
}

// waitForFlowCompletion blocks until the flow execution completes, errors, or
// the context is cancelled. This is the opencode-style pattern: the tool blocks
// for the entire flow lifecycle, including user-input pauses. The agent loop is
// naturally suspended during this time — no soft "relay the waiting_prompt"
// system-prompt rule is needed.
//
// When a user_input node fires, the FlowRunner emits a flow_waiting_user event
// to the frontend via sendFn (the event is broadcast independently of this
// function). The frontend shows the prompt to the user. When the user responds
// via a flow_user_response WebSocket message, HandleUserResponse unblocks the
// flow, which eventually produces the next event (another waiting prompt or
// completion). This function's select loop picks up that next event naturally.
func (r *ChatRunner) waitForFlowCompletion(ctx context.Context, executionId uint) (string, error) {
	for {
		waitCh, doneCh := r.flowRunner.GetWaitChannels(executionId)
		if waitCh == nil {
			// Execution already finished or not found; return current status.
			exec, err := r.flowRunner.GetExecutionStatus(executionId)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf(`{"execution_id":%d,"status":"%s","message":"flow finished"}`, executionId, exec.Status), nil
		}

		select {
		case <-waitCh:
			// Flow is waiting for user input. The flow_waiting_user event has
			// already been emitted to the frontend by sendFn. Just keep waiting
			// — the flow will continue when the user responds.
			continue
		case <-doneCh:
			exec, err := r.flowRunner.GetExecutionStatus(executionId)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf(`{"execution_id":%d,"status":"%s","message":"flow completed"}`, executionId, exec.Status), nil
		case <-ctx.Done():
			// Cancelled (user clicked stop or disconnected).
			r.flowRunner.HandleFlowStop(executionId)
			return "", ctx.Err()
		case <-time.After(5 * time.Minute):
			// Safety timeout: if no event for 5 minutes, return current status.
			exec, _ := r.flowRunner.GetExecutionStatus(executionId)
			return fmt.Sprintf(`{"execution_id":%d,"status":"%s","message":"timeout waiting for flow event"}`, executionId, exec.Status), nil
		}
	}
}

func (r *ChatRunner) flowRespond(ctx context.Context, args map[string]any) (string, error) {
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
	return r.waitForFlowCompletion(ctx, executionId)
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
		"execution_id":   executionId,
		"flow_id":        exec.FlowId,
		"status":         exec.Status,
		"context":        exec.Context,
		"waiting_prompt": exec.WaitingPrompt,
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
