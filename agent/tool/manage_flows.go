package tool

import (
	"encoding/json"
	"fmt"
)

// FlowActionHandler handles flow operations (injected by runner)
type FlowActionHandler func(action string, args map[string]any) (string, error)

var flowHandler FlowActionHandler

// SetFlowHandler registers a flow operation handler
func SetFlowHandler(handler FlowActionHandler) {
	flowHandler = handler
}

func init() {
	Register(&ManageFlows{})
}

// ManageFlows is a flow management tool - create/query/update/delete flows via conversation
type ManageFlows struct{}

func (t *ManageFlows) Definition() Definition {
	return Definition{
		Name: "manage_flows",
		Description: `Manage workflow pipelines (flows) through conversation. Use this to create, list, search, get, update, or delete flows.

WORKFLOW for CREATING a flow — CRITICAL: NEVER auto-create! Follow these steps:

0. DISCOVER models: Before designing any flow that uses LLM nodes, call manage_models with action="list" to discover available models and their identifiers. Note the "Model" column values (e.g. gpt-4o, claude-sonnet-4-6) — you will need these for the model field in llm node configs.

1. UNDERSTAND: Ask the user what the flow should do. What's the input? What steps are needed? What should the output be? If the user's request is vague (e.g. "create a flow for me"), ask specific questions: purpose, data source, processing steps, output format. Also ask: "Which model should I use?" and "What prompt should the LLM use for each step?"

2. DESIGN: Based on the user's answers, propose a concrete node structure that includes model and prompt for every LLM node. For example:
   "I suggest the following flow structure:
   - Start node
   - LLM node: model=gpt-4o, prompt='Summarize the key points from: {{user_input.output}}'
   - User confirmation node to let the user review results
   - End node
   Nodes connected as: Start → LLM → User Confirm → End. Does this look good? Any adjustments needed?"

3. CONFIRM: Wait for the user to explicitly confirm or request changes. Do NOT call create until the user says yes/ok/confirm.

4. CREATE: Only after confirmation, call action="create" with the agreed-upon name, description, category, nodes and edges.

If a user just says "create a flow called X" without describing what it does, ask: "What should this flow do? What processing steps are needed?"

WORKFLOW for editing/deleting by name:
1. First use action="search" with a query string to find the flow
2. If exactly 1 match → use that flow_id; if multiple → ask user to pick; if 0 → report not found
3. For updates: describe the change and confirm before calling update

Node types (REQUIRED fields are marked):
- start: Entry point, no config
- end: Exit point, no config
- llm: LLM call. REQUIRED: {prompt, model}. Optional: {system, temperature(0-2, default 0.7), top_p(0-1, default 0.9), max_tokens, thinking_level(off|low|medium|high|max), output_format_type(empty|json_auto|json_object|json_array|custom)}. Prompt supports {{NodeLabel.output}} templates. NEVER create an llm node without prompt and model.
- user_input: Pause and wait for user input. REQUIRED: {prompt (the question or instruction shown to the user)}. Optional: {confirm_only(bool)}.
- split: Split text into items. REQUIRED: {source_key (which upstream output to split), delimiter: one of paragraph|line|，|。}.
- condition: Conditional branch (yes/no). REQUIRED: {script (Starlark expression, MUST assign a bool to the variable 'result')}. Access upstream data via ctx["node_label"]["field"]. Built-ins: json_parse(s), split(s, sep). Route from "yes"/"no" output handles.
- switch: Multi-branch switch. REQUIRED: {script (Starlark expression, MUST assign a string to the variable 'result')}. Each outgoing edge's source_handle matches a case value. Falls back to "default" if empty.
- transform: Data transform / string formatting. REQUIRED: {template (the transformation template, supports {{NodeLabel.field}} placeholders)}.
- for_each: Concurrent batch processing. REQUIRED: {items_key (name of the upstream output containing the list)}. Optional: {function(default "llm"), args(map, supports {{item.field}} placeholders)}.
- iterator: Sequential iteration (skips failures). REQUIRED: {items_key}. Optional: {function(default "llm"), args(map, supports {{item.field}} placeholders)}.
- loop: Loop until condition is met. REQUIRED: {max_iterations (int, safety limit)}. Optional: {break_field, break_operator (==|!=|>|<|>=|<=|contains), break_value}.
- script: Execute Python/Starlark code. REQUIRED: {script (the code to execute)}. Access upstream data via ctx["node_label"]["field"].
- execute: Run a local shell command. REQUIRED: {command (shell command, supports {{NodeLabel.output}} placeholders)}. Optional: {timeout(int, seconds, 0=no timeout, default 30)}. Failures don't block the flow.
- image_gen: Image generation. REQUIRED: {prompt (image description)}. Optional: {model}.
- audio_gen: Audio synthesis. REQUIRED: {text (text to speak), model}. Optional: {voice}.
- video_gen: Video generation. REQUIRED: {prompt (video description)}. Optional: {model, duration}.

Creation rules: nodes must include start and end nodes. edges use source_index/target_index (0-based). Every REQUIRED config field listed above MUST be filled in — never create a node with empty required fields. In DESIGN step, specify the key config values for every node. Discuss plan → get user confirmation → then create.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type": "string",
					"enum": []string{"create", "list", "search", "get", "update", "delete"},
					"description": "Action type: create(create flow), list(list all), search(fuzzy search by name), get(view details), update(update), delete(delete)",
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
				"nodes": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"type":   map[string]any{"type": "string", "description": "Node type"},
							"label":  map[string]any{"type": "string", "description": "Node display label"},
							"config": map[string]any{"type": "object", "description": "Node config — see node type descriptions above for REQUIRED fields. Every node type has specific required config fields that MUST be filled."},
						},
					},
					"description": "Array of nodes. Must include start and end nodes. For llm nodes, config MUST contain prompt and model. Only call after user confirmation",
				},
				"edges": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"source_index":  map[string]any{"type": "integer", "description": "Source node index in nodes array (0-based)"},
							"target_index":  map[string]any{"type": "integer", "description": "Target node index in nodes array (0-based)"},
							"source_handle": map[string]any{"type": "string", "description": "Source handle: output(default), yes(condition met), no(condition not met)"},
							"label":         map[string]any{"type": "string", "description": "Edge label"},
						},
					},
					"description": "Array of edges, connecting nodes by index",
				},
			},
			"required": []string{"action"},
		},
	}
}

func (t *ManageFlows) Execute(call Call) (string, error) {
	if flowHandler == nil {
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

	return flowHandler(action, params)
}
