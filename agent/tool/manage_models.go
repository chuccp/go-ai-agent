package tool

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/bytedance/sonic"
)

// ModelActionHandler is the handler for AI model CRUD operations (injected by runner).
type ModelActionHandler func(action string, params map[string]any) (string, error)

var modelHandler ModelActionHandler

// SetModelHandler registers the model operation handler.
func SetModelHandler(handler ModelActionHandler) {
	modelHandler = handler
}

func init() {
	Register(&ManageModels{})
}

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
type ManageModels struct{}

func (t *ManageModels) Definition() Definition {
	return Definition{
		Name: "manage_models",
		Description: `Manage AI model configurations through conversation. Use this to list, get, create, update, or delete AI models.

IMPORTANT — Confirmation required for modify operations (create / update / delete):
1. When you call create/update/delete, the tool returns a confirmation prompt with a HIDDEN confirm_key. DO NOT display the raw tool response or the confirm_key to the user. DO NOT show [tool_call ...] or any technical details. Simply tell the user in plain language: "涉及敏感操作，需要再次确认。" followed by a brief description of what will be done, and ask "确认执行？" or "确认删除？".
2. Wait for the user to reply. If they say "确认" / "yes" / "ok" / "好" / "可以" / "执行", call action="confirm" with the confirm_key (NOT the original create/update/delete action).
3. If the user says "取消" / "no" / "不" / "算了", call action="cancel" with the confirm_key to discard the pending operation.
4. list and get execute immediately without confirmation.

When listing or getting models, API keys are always masked as "****" for security. Never reveal them.

For creating a model, you need: name (display name), provider (e.g. openai, deepseek-openai, anthropic, gemini, volcengine, openai_compat, claude_compat), model (model identifier like gpt-4o), api_key, base_url, and optionally category (defaults to llm).`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type":        "string",
					"enum":        []string{"list", "get", "create", "update", "delete", "confirm", "cancel"},
					"description": "操作类型：list(列出所有), get(查看详情), create(创建), update(更新), delete(删除), confirm(确认执行), cancel(取消操作)",
				},
				"id": map[string]any{
					"type":        "integer",
					"description": "模型ID（get/update/delete 操作需要）",
				},
				"name": map[string]any{
					"type":        "string",
					"description": "显示名称（create 时需要）",
				},
				"provider": map[string]any{
					"type":        "string",
					"description": "提供商，如 openai, deepseek-openai, anthropic, gemini, volcengine, openai_compat, claude_compat",
				},
				"model": map[string]any{
					"type":        "string",
					"description": "模型标识，如 gpt-4o, claude-sonnet-4-6",
				},
				"api_key": map[string]any{
					"type":        "string",
					"description": "API 密钥",
				},
				"base_url": map[string]any{
					"type":        "string",
					"description": "API 基础地址",
				},
				"category": map[string]any{
					"type":        "string",
					"description": "模型分类（llm/image/voice/video），默认 llm",
				},
				"input_types": map[string]any{
					"type":        "string",
					"description": "支持的输入类型，逗号分隔（text,image,audio,video）",
				},
				"output_types": map[string]any{
					"type":        "string",
					"description": "支持的输出类型，逗号分隔（text,image,audio,video）",
				},
				"supports_multimodal": map[string]any{
					"type":        "boolean",
					"description": "是否支持多模态（图片输入），仅 LLM 模型有效",
				},
				"confirm_key": map[string]any{
					"type":        "string",
					"description": "确认码（内部使用，不要展示给用户）。create/update/delete 首次调用返回此码，用户确认后用 action=confirm 提交",
				},
			},
			"required": []string{"action"},
		},
	}
}

func (t *ManageModels) Execute(call Call) (string, error) {
	if modelHandler == nil {
		return "", fmt.Errorf("model handler not initialized")
	}

	var params map[string]any
	if err := sonic.Unmarshal([]byte(call.Arguments), &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	action, ok := params["action"].(string)
	if !ok || action == "" {
		return "", fmt.Errorf("action is required")
	}

	switch action {
	case "list", "get":
		return modelHandler(action, params)

	case "create", "update", "delete":
		if confirmKey, _ := params["confirm_key"].(string); confirmKey != "" {
			op, found := loadPendingOp(confirmKey)
			if !found {
				return "确认码无效或已过期，请重新发起操作。", nil
			}
			return modelHandler(op.action, op.params)
		}
		key := storePendingOp(action, params)
		desc := describePendingOp(action, params)
		return fmt.Sprintf("[confirm_key:%s]\n涉及敏感操作，需要再次确认。\n\n%s\n\n请回复\"确认\"执行，或\"取消\"放弃。", key, desc), nil

	case "confirm":
		confirmKey, _ := params["confirm_key"].(string)
		if confirmKey == "" {
			return "缺少 confirm_key。", nil
		}
		op, found := loadPendingOp(confirmKey)
		if !found {
			return "确认码无效或已过期，请重新发起操作。", nil
		}
		return modelHandler(op.action, op.params)

	case "cancel":
		confirmKey, _ := params["confirm_key"].(string)
		if confirmKey != "" {
			dropPendingOp(confirmKey)
		}
		return "操作已取消。", nil

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
		return fmt.Sprintf("删除模型 #%d", uint(id))
	case "create":
		return fmt.Sprintf("创建模型「%s」(%s.%s)", name, provider, modelName)
	case "update":
		return fmt.Sprintf("更新模型 #%d", uint(id))
	}
	return fmt.Sprintf("执行 %s 操作", action)
}
