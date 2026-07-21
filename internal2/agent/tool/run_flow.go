package tool

import (
	"context"
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

The tool BLOCKS until the flow completes. If the flow requires user input,
prompts are shown to the user automatically by the frontend — you do NOT need
to relay questions or end your turn early. When the flow finishes, the tool
returns the final result.

Workflow:
1. If the user mentions a flow by name but you don't have the ID, call action="search" with the query.
2. If exactly one flow matches, call action="run" with flow_id and any initial_input or form_values.
3. To start the special built-in "create flow" assistant, call action="run" with builtin_flow="create_flow" and an initial_input describing what the user wants.
4. To modify an existing flow, first get the flow JSON via manage_flows (action="get", format="json"), then call action="run" with builtin_flow="modify_flow" and initial_input containing the existing flow JSON + the user's modification request.
5. The tool blocks until completion. Summarize the result for the user when it returns.
6. To stop a running flow, call action="stop" with execution_id.

Actions:
- search: find flows by name/description. Required: query.
- run: start a flow. Required: flow_id OR builtin_flow. Optional: initial_input, form_values (object).
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
				"builtin_flow": map[string]any{
					"type":        "string",
					"enum":        []string{"create_flow", "modify_flow"},
					"description": "Built-in flow name. Use 'create_flow' to start the conversational flow-creation assistant, 'modify_flow' to modify an existing flow.",
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

func (t *RunFlow) Execute(ctx context.Context, call Call) (string, error) {
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

	return t.flowExecutionHandler(ctx, action, params)
}
