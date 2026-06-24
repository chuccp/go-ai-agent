package tool

import (
	"context"
	"encoding/json"
	"fmt"
)

// ManageFlows is a flow management tool - create/query/update/delete flows via conversation
type ManageFlows struct {
	flowHandler FlowActionHandler
}

func (t *ManageFlows) SetFlowHandler(h FlowActionHandler) {
	t.flowHandler = h
}

func (t *ManageFlows) Definition() Definition {
	return Definition{
		Name: "manage_flows",
		Description: `Manage workflow pipelines (flows) — list, search, get, and delete.

Creating or modifying flows is handled by dedicated built-in flows:
- To CREATE a flow: use the create_flow_conversation tool.
- To MODIFY a flow: use manage_flows action="get" with format="json" to fetch the flow JSON, then use run_flow with builtin_flow="modify_flow" and initial_input containing the existing flow JSON + the modification request.

This tool is for CRUD operations only:
- list: list all flows
- search: fuzzy search by name
- get: view flow details (use format="json" to get the full JSON definition for modify_flow)
- delete: delete a flow by ID`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type": "string",
					"enum": []string{"list", "search", "get", "delete"},
					"description": "Action type: list(list all), search(fuzzy search by name), get(view details, use format=json for full JSON), delete(delete)",
				},
				"query": map[string]any{
					"type":        "string",
					"description": "Search keyword (for search action), fuzzy match by name",
				},
				"name": map[string]any{
					"type":        "string",
					"description": "Flow name (required for create/update)",
				},
				"description": map[string]any{
					"type":        "string",
					"description": "Flow description",
				},
				"category": map[string]any{
					"type":        "string",
					"description": "Flow category",
				},
				"flow_id": map[string]any{
					"type":        "integer",
					"description": "Flow ID (required for get/update/delete)",
				},
				"format": map[string]any{
					"type":        "string",
					"description": "For get action: use \"json\" to return the full flow definition as JSON (needed for modify_flow)",
				},
			},
			"required": []string{"action"},
		},
	}
}

func (t *ManageFlows) Execute(ctx context.Context, call Call) (string, error) {
	if t.flowHandler == nil {
		return "", fmt.Errorf("flow handler not initialized")
	}

	var params map[string]any
	if err := json.Unmarshal([]byte(call.Arguments), &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	action, ok := params["action"].(string)
	if !ok || action == "" {
		return "", fmt.Errorf("action is required")
	}

	return t.flowHandler(action, params)
}
