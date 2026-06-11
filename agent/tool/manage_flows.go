package tool

import (
	"encoding/json"
	"fmt"
)

// FlowActionHandler 流程操作处理器（由 runner 注入）
type FlowActionHandler func(action string, args map[string]any) (string, error)

var flowHandler FlowActionHandler

// SetFlowHandler 注册流程操作处理器
func SetFlowHandler(handler FlowActionHandler) {
	flowHandler = handler
}

func init() {
	Register(&ManageFlows{})
}

// ManageFlows 流程管理工具 - 通过对话创建/查询/更新/删除流程
type ManageFlows struct{}

func (t *ManageFlows) Definition() Definition {
	return Definition{
		Name: "manage_flows",
		Description: `Manage workflow pipelines (flows) through conversation. Use this to create, list, search, get, update, or delete flows.

WORKFLOW for CREATING a flow — CRITICAL: NEVER auto-create! Follow these steps:

1. UNDERSTAND: Ask the user what the flow should do. What's the input? What steps are needed? What should the output be? If the user's request is vague (e.g. "create a flow for me"), ask specific questions: purpose, data source, processing steps, output format.

2. DESIGN: Based on the user's answers, propose a concrete node structure in plain language. For example:
   "我建议这个流程包含以下步骤：
   - 开始节点
   - LLM调用节点，用来...（模型：xxx，提示词：...）
   - 用户确认节点，让用户审核结果
   - 结束节点
   节点按 开始→LLM→用户确认→结束 的顺序连接。这样可以吗？需要调整吗？"

3. CONFIRM: Wait for the user to explicitly confirm or request changes. Do NOT call create until the user says yes/ok/confirm.

4. CREATE: Only after confirmation, call action="create" with the agreed-upon name, description, category, nodes and edges.

If a user just says "create a flow called X" without describing what it does, ask: "这个流程需要做什么？包含哪些处理步骤？"

WORKFLOW for editing/deleting by name:
1. First use action="search" with a query string to find the flow
2. If exactly 1 match → use that flow_id; if multiple → ask user to pick; if 0 → report not found
3. For updates: describe the change and confirm before calling update

Node types:
- start: 开始节点，无配置
- end: 结束节点，无配置
- llm: LLM调用。config: {model, prompt(支持{{NodeLabel.output}}模板), system, temperature(0-2,默认0.7), top_p(0-1,默认0.9), max_tokens, thinking_level(off|low|medium|high|max), output_format_type(空|json_auto|json_object|json_array|custom)}
- user_input: 用户输入。config: {prompt, confirm_only(bool)}
- split: 文本拆分。config: {source_key, delimiter(paragraph|line|，|。)}
- condition: 条件分支。config: {field, operator(contains|equals|not_empty), value}。从"yes"/"no"出口连线
- transform: 数据变换。config: {template}
- for_each: 并发批量处理，数组各项独立并发执行，互不依赖。config: {items_key(数据来源键)}
- iterator: 按序迭代，逐项顺序执行，下一项可使用前一项结果。config: {items_key(数据来源键)}
- loop: 循环。config: {max_iterations, break_field, break_operator, break_value}
- script: 脚本。config: {script(Python/Starlark代码)}
- image_gen: 图片生成。config: {prompt, model}
- audio_gen: 音频合成。config: {text, model, voice}
- video_gen: 视频生成。config: {prompt, model, duration}

创建规则：nodes必须包含start和end节点。edges用source_index/target_index(0-based)。先讨论方案→用户确认→再创建。`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type": "string",
					"enum": []string{"create", "list", "search", "get", "update", "delete"},
					"description": "操作类型：create(创建流程), list(列出所有流程), search(按名称模糊搜索), get(查看详情), update(更新), delete(删除)",
				},
				"query": map[string]any{
					"type":        "string",
					"description": "搜索关键词（search 操作时使用），按名称模糊匹配",
				},
				"name": map[string]any{
					"type":        "string",
					"description": "流程名称（create/update 时需要）",
				},
				"description": map[string]any{
					"type":        "string",
					"description": "流程描述",
				},
				"category": map[string]any{
					"type":        "string",
					"description": "流程分类",
				},
				"flow_id": map[string]any{
					"type":        "integer",
					"description": "流程ID（get/update/delete 时需要）",
				},
				"nodes": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"type":   map[string]any{"type": "string", "description": "节点类型"},
							"label":  map[string]any{"type": "string", "description": "节点显示标签"},
							"config": map[string]any{"type": "object", "description": "节点配置，参见节点类型说明"},
						},
					},
					"description": "节点列表。创建时必须包含start和end节点。仅用户确认后调用",
				},
				"edges": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"source_index":  map[string]any{"type": "integer", "description": "源节点在nodes数组中的索引（从0开始）"},
							"target_index":  map[string]any{"type": "integer", "description": "目标节点在nodes数组中的索引（从0开始）"},
							"source_handle": map[string]any{"type": "string", "description": "源出口：output(默认), yes(条件为真), no(条件为假)"},
							"label":         map[string]any{"type": "string", "description": "连线标签"},
						},
					},
					"description": "连线列表，用节点索引连接",
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
