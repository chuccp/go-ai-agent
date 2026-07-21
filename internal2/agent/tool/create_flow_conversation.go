package tool

import (
	"context"
	"encoding/json"
	"fmt"
)

// CreateFlowConversation triggers the built-in conversational flow-creation assistant.
type CreateFlowConversation struct {
	flowExecutionHandler FlowExecutionHandler
}

func (t *CreateFlowConversation) SetFlowExecutionHandler(h FlowExecutionHandler) {
	t.flowExecutionHandler = h
}

func (t *CreateFlowConversation) Definition() Definition {
	return Definition{
		Name: "create_flow_conversation",
		Description: `Start the built-in conversational flow-creation assistant.

Use this tool whenever the user wants to create a new flow (workflow/app), especially when they describe what they want in natural language. Examples:
- "帮我创建一个流程"
- "create a flow that generates stories"
- "我想做一个输入一句话输出故事的流程"
- "用对话方式生成一个flow"

This assistant will ask the user follow-up questions and automatically generate the flow. Do NOT use manage_flows create unless the user explicitly asks for manual node-level control.

Parameters:
- initial_input (required): a description of what the user wants the flow to do. Include purpose, input, and output if known.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"initial_input": map[string]any{
					"type":        "string",
					"description": "Description of the flow the user wants to create",
				},
			},
			"required": []string{"initial_input"},
		},
	}
}

func (t *CreateFlowConversation) Execute(ctx context.Context, call Call) (string, error) {
	if t.flowExecutionHandler == nil {
		return "", fmt.Errorf("flow execution handler not initialized")
	}

	var params map[string]any
	if err := json.Unmarshal([]byte(call.Arguments), &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	initialInput, _ := params["initial_input"].(string)
	if initialInput == "" {
		return "", fmt.Errorf("initial_input is required")
	}

	return t.flowExecutionHandler(ctx, "run", map[string]any{
		"builtin_flow":  "create_flow",
		"initial_input": initialInput,
	})
}
