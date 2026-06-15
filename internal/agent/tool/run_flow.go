package tool

import (
	"encoding/json"
	"fmt"
)

// RunFlow lets the agent search for and execute flows via conversation.
type RunFlow struct {
	flowExecutionHandler FlowExecutionHandler
}

func (t *RunFlow) SetFlowExecutionHandler(h FlowExecutionHandler) {
	t.flowExecutionHandler = h
}

func (t *RunFlow) Definition() Definition {
	return Definition{
		Name: "run_flow",
		Description: `Execute a flow (workflow) through conversation.

Use this tool when the user wants to run, execute, or use a flow.

Workflow:
1. If the user mentions a flow by name but you don't have the ID, call action="search" with the query.
2. If exactly one flow matches, call action="run" with flow_id and any initial_input or form_values.
3. If the flow requires user input (tool result says waiting_user), ask the user the question and then call action="respond" with execution_id and response.
4. To check progress, call action="status" with execution_id.
5. To stop a running flow, call action="stop" with execution_id.

Actions:
- search: find flows by name/description. Required: query.
- run: start a flow. Required: flow_id. Optional: initial_input, form_values (object).
- respond: continue a paused flow. Required: execution_id, response.
- status: get execution status. Required: execution_id.
- stop: stop execution. Required: execution_id.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type":        "string",
					"enum":        []string{"search", "run", "respond", "status", "stop"},
					"description": "Action type",
				},
				"query": map[string]any{
					"type":        "string",
					"description": "Search keyword (for search action)",
				},
				"flow_id": map[string]any{
					"type":        "integer",
					"description": "Flow ID (for run action)",
				},
				"execution_id": map[string]any{
					"type":        "integer",
					"description": "Execution ID (for respond/status/stop)",
				},
				"initial_input": map[string]any{
					"type":        "string",
					"description": "Initial input text passed to the flow",
				},
				"form_values": map[string]any{
					"type":        "object",
					"description": "Form field values if the flow has a form_schema",
				},
				"response": map[string]any{
					"type":        "string",
					"description": "User response (for respond action)",
				},
			},
			"required": []string{"action"},
		},
	}
}

func (t *RunFlow) Execute(call Call) (string, error) {
	if t.flowExecutionHandler == nil {
		return "", fmt.Errorf("flow execution handler not initialized")
	}

	var params map[string]any
	if err := json.Unmarshal([]byte(call.Arguments), &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	action, ok := params["action"].(string)
	if !ok || action == "" {
		return "", fmt.Errorf("action is required")
	}

	return t.flowExecutionHandler(action, params)
}
