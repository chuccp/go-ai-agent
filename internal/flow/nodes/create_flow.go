package nodes

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/chuccp/go-ai-agent/internal/entity"
	"github.com/chuccp/go-ai-agent/internal/flow/engine"
	"github.com/chuccp/go-ai-agent/internal/service"
)

// CreateFlowNodeConfig config for the internal create_flow node.
type CreateFlowNodeConfig struct {
	Source string `json:"source"` // Upstream node label whose "output" contains the flow JSON
}

// CreateFlowNode internal node that persists a flow definition to the database.
// It is intended for built-in/meta flows only and is not exposed in the frontend palette.
type CreateFlowNode struct {
	flowService *service.FlowService
}

func NewCreateFlowNode(svc *service.FlowService) *CreateFlowNode {
	return &CreateFlowNode{flowService: svc}
}

func (n *CreateFlowNode) Type() string { return TypeCreateFlow }

func (n *CreateFlowNode) Execute(ctx *engine.ExecutionContext, config string) (*engine.NodeOutput, error) {
	if n.flowService == nil {
		return nil, fmt.Errorf("create_flow: flow service not initialized")
	}

	cfg, err := engine.GetNodeConfig[CreateFlowNodeConfig](config)
	if err != nil {
		return nil, err
	}
	if cfg.Source == "" {
		return nil, fmt.Errorf("create_flow: source field is required")
	}

	key := cfg.Source
	if !containsDot(key) {
		key = key + "." + KeyOutput
	}
	raw, ok := ctx.Get(key)
	if !ok {
		return nil, fmt.Errorf("create_flow: source %s not found in context", key)
	}

	payload, err := parseFlowPayload(fmt.Sprintf("%v", raw))
	if err != nil {
		return nil, fmt.Errorf("create_flow: invalid flow json: %w", err)
	}

	f, err := n.flowService.CreateFlow(
		payload.Name,
		payload.Description,
		payload.Category,
		payload.Config,
		payload.FormSchema,
		payload.Settings,
		payload.Icon,
		payload.Nodes,
		payload.Edges,
	)
	if err != nil {
		return nil, fmt.Errorf("create_flow: failed to create flow: %w", err)
	}

	return &engine.NodeOutput{
		Data: map[string]any{
			KeyOutput: fmt.Sprintf("Flow created: %s (id=%d)", f.Name, f.Id),
			"flow_id": f.Id,
			"name":    f.Name,
		},
		Status: engine.StatusSuccess,
	}, nil
}

func containsDot(s string) bool {
	for _, c := range s {
		if c == '.' {
			return true
		}
	}
	return false
}

func stripMarkdownCodeBlock(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		if idx := strings.Index(s[3:], "\n"); idx >= 0 {
			s = s[idx+4:]
		} else {
			s = s[3:]
		}
	}
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSpace(s[:len(s)-3])
	}
	return s
}

// parseFlowPayload parses the LLM-generated flow JSON and normalizes node configs to JSON strings.
func parseFlowPayload(raw string) (*flowPayload, error) {
	jsonStr := stripMarkdownCodeBlock(raw)

	var flexible struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Category    string `json:"category"`
		Config      string `json:"config"`
		FormSchema  string `json:"form_schema"`
		Settings    string `json:"settings"`
		Icon        string `json:"icon"`
		Nodes       []struct {
			Type      string  `json:"type"`
			Label     string  `json:"label"`
			Config    any     `json:"config"`
			GroupId   *uint   `json:"group_id"`
			PositionX float64 `json:"position_x"`
			PositionY float64 `json:"position_y"`
		} `json:"nodes"`
		Edges []struct {
			SourceIndex  int    `json:"source"`
			TargetIndex  int    `json:"target"`
			FromIndex    int    `json:"from"`
			ToIndex      int    `json:"to"`
			SourceHandle string `json:"source_handle"`
			TargetHandle string `json:"target_handle"`
			Label        string `json:"label"`
		} `json:"edges"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &flexible); err != nil {
		return nil, err
	}

	payload := &flowPayload{
		Name:        flexible.Name,
		Description: flexible.Description,
		Category:    flexible.Category,
		Config:      normalizeString(flexible.Config, "{}"),
		FormSchema:  flexible.FormSchema,
		Settings:    flexible.Settings,
		Icon:        flexible.Icon,
	}

	for i, n := range flexible.Nodes {
		cfgStr := "{}"
		if n.Config != nil {
			switch v := n.Config.(type) {
			case string:
				cfgStr = v
			default:
				b, err := json.Marshal(v)
				if err != nil {
					return nil, fmt.Errorf("node %s config marshal error: %w", n.Label, err)
				}
				cfgStr = string(b)
			}
		}
		// Ensure LLM/gen model paths use the required "id.model" format.
		if n.Type == TypeLLM || n.Type == TypeImageGen || n.Type == TypeAudioGen || n.Type == TypeVideoGen {
			cfgStr = normalizeModelPathInConfig(cfgStr)
		}
		// LLMs often emit {{user_input}} instead of {{user_input.output}}; fix it.
		if n.Type == TypeLLM {
			cfgStr = normalizePromptPlaceholders(cfgStr)
		}

		payload.Nodes = append(payload.Nodes, &entity.FlowNode{
			Id:        uint(i + 1), // assign sequential IDs for edge mapping
			Type:      n.Type,
			Label:     n.Label,
			Config:    cfgStr,
			GroupId:   n.GroupId,
			PositionX: n.PositionX,
			PositionY: n.PositionY,
		})
	}

	// Fix common LLM placeholder mistakes using actual node labels.
	hasUserInput := false
	for _, n := range payload.Nodes {
		if n.Type == TypeUserInput && n.Label == "user_input" {
			hasUserInput = true
			break
		}
	}
	for _, n := range payload.Nodes {
		if n.Type == TypeLLM {
			n.Config = normalizePromptPlaceholdersWithNodes(n.Config, payload.Nodes)
			if hasUserInput {
				n.Config = ensureUserInputPlaceholder(n.Config)
			}
		}
	}

	for _, e := range flexible.Edges {
		// LLM outputs 0-based indices; convert to 1-based IDs matching the node IDs above.
		// Some LLMs use "source"/"target", others use "from"/"to".
		srcIdx := e.SourceIndex
		tgtIdx := e.TargetIndex
		if srcIdx == 0 && e.FromIndex != 0 {
			srcIdx = e.FromIndex
		}
		if tgtIdx == 0 && e.ToIndex != 0 {
			tgtIdx = e.ToIndex
		}
		payload.Edges = append(payload.Edges, &entity.FlowEdge{
			SourceNodeId: uint(srcIdx + 1),
			TargetNodeId: uint(tgtIdx + 1),
			SourceHandle: defaultHandle(e.SourceHandle, "output"),
			TargetHandle: defaultHandle(e.TargetHandle, "input"),
			Label:        e.Label,
		})
	}

	return payload, nil
}

func normalizeString(v any, def string) string {
	if v == nil {
		return def
	}
	if s, ok := v.(string); ok {
		if s == "" {
			return def
		}
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return def
	}
	return string(b)
}

func defaultHandle(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// normalizeModelPathInConfig ensures the "model" field inside a node config uses "id.model" format.
func normalizeModelPathInConfig(cfgStr string) string {
	var cfg map[string]any
	if err := json.Unmarshal([]byte(cfgStr), &cfg); err != nil {
		return cfgStr
	}
	m, ok := cfg["model"].(string)
	if !ok || m == "" {
		return cfgStr
	}
	if !strings.Contains(m, ".") {
		cfg["model"] = "1." + m
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		return cfgStr
	}
	return string(b)
}

// normalizePromptPlaceholders rewrites bare {{user_input}} to {{user_input.output}}
// so that LLM prompts correctly reference the user input node's output field.
func normalizePromptPlaceholders(cfgStr string) string {
	var cfg map[string]any
	if err := json.Unmarshal([]byte(cfgStr), &cfg); err != nil {
		return cfgStr
	}
	prompt, ok := cfg["prompt"].(string)
	if !ok || prompt == "" {
		return cfgStr
	}
	// Replace {{user_input}} when it is not already suffixed with .something.
	re := regexp.MustCompile(`\{\{user_input\}\}`)
	cfg["prompt"] = re.ReplaceAllString(prompt, "{{user_input.output}}")
	b, err := json.Marshal(cfg)
	if err != nil {
		return cfgStr
	}
	return string(b)
}

// normalizePromptPlaceholdersWithNodes rewrites placeholders that reference node
// indices (e.g. {{node_0.output}}) to the actual label of the node at that index.
// It also fixes bare {{user_input}} to {{user_input.output}}.
func normalizePromptPlaceholdersWithNodes(cfgStr string, nodes []*entity.FlowNode) string {
	var cfg map[string]any
	if err := json.Unmarshal([]byte(cfgStr), &cfg); err != nil {
		return cfgStr
	}
	prompt, ok := cfg["prompt"].(string)
	if !ok || prompt == "" {
		return cfgStr
	}

	// Build index -> label map.
	idxToLabel := make(map[int]string)
	for i, n := range nodes {
		idxToLabel[i] = n.Label
	}

	// Replace {{node_i}} and {{node_i.field}} with {{label.field}}.
	re := regexp.MustCompile(`\{\{node_(\d+)(\.[^}]+)?\}\}`)
	prompt = re.ReplaceAllStringFunc(prompt, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		idx, err := strconv.Atoi(parts[1])
		if err != nil {
			return match
		}
		label, ok := idxToLabel[idx]
		if !ok || label == "" {
			return match
		}
		field := ".output"
		if len(parts) >= 3 && parts[2] != "" {
			field = parts[2]
		}
		return fmt.Sprintf("{{%s%s}}", label, field)
	})

	// Replace bare {{user_input}} with {{user_input.output}}.
	userRe := regexp.MustCompile(`\{\{user_input\}\}`)
	prompt = userRe.ReplaceAllString(prompt, "{{user_input.output}}")

	cfg["prompt"] = prompt
	b, err := json.Marshal(cfg)
	if err != nil {
		return cfgStr
	}
	return string(b)
}

// ensureUserInputPlaceholder makes sure an LLM prompt that references user input
// actually contains a {{user_input.output}} placeholder. LLMs often emit a prompt
// like "请根据以下句子写故事：用户描述的一句话" instead of the required template.
func ensureUserInputPlaceholder(cfgStr string) string {
	var cfg map[string]any
	if err := json.Unmarshal([]byte(cfgStr), &cfg); err != nil {
		return cfgStr
	}
	prompt, ok := cfg["prompt"].(string)
	if !ok || prompt == "" {
		return cfgStr
	}
	// Already has a placeholder, leave it alone.
	if strings.Contains(prompt, "{{") && strings.Contains(prompt, "}}") {
		return cfgStr
	}

	placeholder := "{{user_input.output}}"
	// Find the last sentence separator that introduces user input.
	seps := []string{":", "：", "?", "？", ".", "。", "!", "！"}
	lastIdx := -1
	lastSepLen := 0
	for _, sep := range seps {
		if idx := strings.LastIndex(prompt, sep); idx > lastIdx {
			lastIdx = idx
			lastSepLen = len(sep)
		}
	}
	if lastIdx >= 0 {
		prompt = strings.TrimSpace(prompt[:lastIdx+lastSepLen]) + placeholder
	} else {
		prompt = prompt + placeholder
	}

	cfg["prompt"] = prompt
	b, err := json.Marshal(cfg)
	if err != nil {
		return cfgStr
	}
	return string(b)
}

// flowPayload mirrors the REST create-flow payload.
type flowPayload struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Category    string             `json:"category"`
	Config      string             `json:"config"`
	FormSchema  string             `json:"form_schema"`
	Settings    string             `json:"settings"`
	Icon        string             `json:"icon"`
	Nodes       []*entity.FlowNode `json:"nodes"`
	Edges       []*entity.FlowEdge `json:"edges"`
}
