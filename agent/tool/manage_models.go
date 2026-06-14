package tool

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"encoding/json"
)

// ModelActionHandler is the handler for AI model CRUD operations (injected by runner).
type ModelActionHandler func(action string, params map[string]any) (string, error)

// pendingModelOp stores a model operation awaiting confirmation.
type pendingModelOp struct {
	action  string
	params  map[string]any
	expires time.Time
}

var pendingModelOps sync.Map // map[string]pendingModelOp

func storePendingOp(action string, params map[string]any) string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%08d", time.Now().UnixNano()%100000000)
	}
	key := hex.EncodeToString(b)

	pendingModelOps.Store(key, pendingModelOp{
		action:  action,
		params:  params,
		expires: time.Now().Add(5 * time.Minute),
	})
	return key
}

func loadPendingOp(key string) (pendingModelOp, bool) {
	v, ok := pendingModelOps.Load(key)
	if !ok {
		return pendingModelOp{}, false
	}
	op := v.(pendingModelOp)
	if time.Now().After(op.expires) {
		pendingModelOps.Delete(key)
		return pendingModelOp{}, false
	}
	pendingModelOps.Delete(key)
	return op, true
}

func dropPendingOp(key string) {
	pendingModelOps.Delete(key)
}

// ManageModels lets the user manage AI models through conversation.
type ManageModels struct {
	reg *Registry
}

func (t *ManageModels) Definition() Definition {
	return Definition{
		Name: "manage_models",
		Description: `Manage AI model configurations through conversation. Use this to list, get, create, update, or delete AI models.

IMPORTANT — Confirmation required for modify operations (create / update / delete):
1. When you call create/update/delete, the tool returns a confirmation prompt with a HIDDEN confirm_key. DO NOT display the raw tool response or the confirm_key to the user. DO NOT show [tool_call ...] or any technical details. Simply tell the user in plain language: "This is a sensitive operation and requires confirmation." followed by a brief description of what will be done, and ask "Confirm execution?" or "Confirm deletion?".
2. Wait for the user to reply. If they say "confirm" / "yes" / "ok" / "yep" / "sure" / "go ahead", call action="confirm" with the confirm_key (NOT the original create/update/delete action).
3. If the user says "cancel" / "no" / "nope" / "never mind", call action="cancel" with the confirm_key to discard the pending operation.
4. list and get execute immediately without confirmation.

When listing or getting models, API keys are always masked as "****" for security. Never reveal them.

For creating a model, you need: name (display name), provider (e.g. openai, deepseek-openai, anthropic, gemini, volcengine, openai_compat, claude_compat), model (model identifier like gpt-4o), api_key, base_url, and optionally category (defaults to llm).`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type":        "string",
					"enum":        []string{"list", "get", "create", "update", "delete", "confirm", "cancel"},
					"description": "Action type: list(list all), get(view details), create(create), update(update), delete(delete), confirm(confirm execution), cancel(cancel operation)",
				},
				"id": map[string]any{
					"type":        "integer",
					"description": "Model ID (required for get/update/delete operations)",
				},
				"name": map[string]any{
					"type":        "string",
					"description": "Display name (required for create)",
				},
				"provider": map[string]any{
					"type":        "string",
					"description": "Provider, e.g. openai, deepseek-openai, anthropic, gemini, volcengine, openai_compat, claude_compat",
				},
				"model": map[string]any{
					"type":        "string",
					"description": "Model identifier, e.g. gpt-4o, claude-sonnet-4-6",
				},
				"api_key": map[string]any{
					"type":        "string",
					"description": "API key",
				},
				"base_url": map[string]any{
					"type":        "string",
					"description": "API base URL",
				},
				"category": map[string]any{
					"type":        "string",
					"description": "Model category (llm/image/voice/video), default: llm",
				},
				"input_types": map[string]any{
					"type":        "string",
					"description": "Supported input types, comma-separated (text,image,audio,video)",
				},
				"output_types": map[string]any{
					"type":        "string",
					"description": "Supported output types, comma-separated (text,image,audio,video)",
				},
				"supports_multimodal": map[string]any{
					"type":        "boolean",
					"description": "Whether multimodal input (image) is supported, only applies to LLM models",
				},
				"confirm_key": map[string]any{
					"type":        "string",
					"description": "Confirmation key (internal use, do NOT show to user). Returned by first create/update/delete call; submit with action=confirm after user approval",
				},
			},
			"required": []string{"action"},
		},
	}
}

func (t *ManageModels) Execute(call Call) (string, error) {
	if t.reg == nil || t.reg.ModelHandler == nil {
		return "", fmt.Errorf("model handler not initialized")
	}
	mh := t.reg.ModelHandler

	var params map[string]any
	if err := json.Unmarshal([]byte(call.Arguments), &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	action, ok := params["action"].(string)
	if !ok || action == "" {
		return "", fmt.Errorf("action is required")
	}

	switch action {
	case "list", "get":
		return mh(action, params)

	case "create", "update", "delete":
		if confirmKey, _ := params["confirm_key"].(string); confirmKey != "" {
			op, found := loadPendingOp(confirmKey)
			if !found {
				return "Confirmation key is invalid or expired. Please retry the operation.", nil
			}
			return mh(op.action, op.params)
		}
		key := storePendingOp(action, params)
		desc := describePendingOp(action, params)
		return fmt.Sprintf("[confirm_key:%s]\nThis is a sensitive operation and requires confirmation.\n\n%s\n\nReply \"confirm\" to execute, or \"cancel\" to abort.", key, desc), nil

	case "confirm":
		confirmKey, _ := params["confirm_key"].(string)
		if confirmKey == "" {
			return "Missing confirm_key.", nil
		}
		op, found := loadPendingOp(confirmKey)
		if !found {
			return "Confirmation key is invalid or expired. Please retry the operation.", nil
		}
		return mh(op.action, op.params)

	case "cancel":
		confirmKey, _ := params["confirm_key"].(string)
		if confirmKey != "" {
			dropPendingOp(confirmKey)
		}
		return "Operation cancelled.", nil

	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

func describePendingOp(action string, params map[string]any) string {
	id, _ := params["id"].(float64)
	name, _ := params["name"].(string)
	provider, _ := params["provider"].(string)
	modelName, _ := params["model"].(string)

	switch action {
	case "delete":
		return fmt.Sprintf("Delete model #%d", uint(id))
	case "create":
		return fmt.Sprintf("Create model \"%s\" (%s.%s)", name, provider, modelName)
	case "update":
		return fmt.Sprintf("Update model #%d", uint(id))
	}
	return fmt.Sprintf("Execute %s operation", action)
}
