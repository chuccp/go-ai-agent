package runner

import (
	"fmt"
	"strings"

	"encoding/json"
	"github.com/chuccp/go-ai-agent/internal/entity"
)

func (r *ChatRunner) flowList() (string, error) {
	flows, err := r.flowModel.List()
	if err != nil {
		return "", err
	}
	if len(flows) == 0 {
		return "No flows yet", nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Total %d flows:\n", len(flows)))
	for _, f := range flows {
		sb.WriteString(fmt.Sprintf("- [ID:%d] %s (%s)\n", f.Id, f.Name, f.Category))
	}
	return sb.String(), nil
}

func (r *ChatRunner) flowSearch(args map[string]any) (string, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return "", fmt.Errorf("search query cannot be empty")
	}

	flows, err := r.flowModel.List()
	if err != nil {
		return "", err
	}

	queryLower := strings.ToLower(query)

	// Round 1: exact match (case-insensitive)
	var exact []*entity.FlowDefinition
	for _, f := range flows {
		if strings.ToLower(f.Name) == queryLower {
			exact = append(exact, f)
		}
	}
	if len(exact) == 1 {
		f := exact[0]
		return fmt.Sprintf("Exact match found:\n[ID:%d] %s (%s)\nYou can proceed with this flow.", f.Id, f.Name, f.Category), nil
	}
	if len(exact) > 1 {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d flows with the same name, ask the user to pick one:\n", len(exact)))
		for _, f := range exact {
			sb.WriteString(fmt.Sprintf("- [ID:%d] %s (%s) Description: %s\n", f.Id, f.Name, f.Category, f.Description))
		}
		sb.WriteString("Ask the user to reply with the number or ID to select")
		return sb.String(), nil
	}

	// Round 2: partial match
	var partial []*entity.FlowDefinition
	for _, f := range flows {
		if strings.Contains(strings.ToLower(f.Name), queryLower) {
			partial = append(partial, f)
		}
	}
	if len(partial) == 1 {
		f := partial[0]
		return fmt.Sprintf("Matching flow found:\n[ID:%d] %s (%s)\nDescription: %s\nYou can proceed with this flow.", f.Id, f.Name, f.Category, f.Description), nil
	}
	if len(partial) > 1 {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d matching '%s' flows, ask the user to select:\n", len(partial), query))
		for _, f := range partial {
			sb.WriteString(fmt.Sprintf("- [ID:%d] %s (%s)\n", f.Id, f.Name, f.Category))
		}
		sb.WriteString("Ask the user to reply with ID or name to select")
		return sb.String(), nil
	}

	// Not found
	return fmt.Sprintf("No flow found with name containing '%s' Use list to view all flows.", query), nil
}

func (r *ChatRunner) flowGet(args map[string]any) (string, error) {
	id, err := getUintArg(args, "flow_id")
	if err != nil {
		return "", err
	}
	f, err := r.flowModel.FindById(id)
	if err != nil {
		return "", fmt.Errorf("flow not found: ID=%d", id)
	}
	nodes, _ := r.nodeModel.FindByFlowId(id)
	edges, _ := r.edgeModel.FindByFlowId(id)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Flow [ID:%d] %s | %s | %d nodes %d edges\n",
		f.Id, f.Name, f.Category, len(nodes), len(edges)))
	for _, n := range nodes {
		extra := nodeConfigSummary(n)
		sb.WriteString(fmt.Sprintf("  [%d] %s(%s)%s\n", n.Id, n.Label, n.Type, extra))
	}
	for _, e := range edges {
		sb.WriteString(fmt.Sprintf("  %d->%d", e.SourceNodeId, e.TargetNodeId))
		if e.SourceHandle != "" && e.SourceHandle != "output" {
			sb.WriteString(fmt.Sprintf("(%s)", e.SourceHandle))
		}
		sb.WriteString(" ")
	}
	return sb.String(), nil
}

func (r *ChatRunner) flowCreate(args map[string]any) (string, error) {
	name, _ := args["name"].(string)
	if name == "" {
		return "", fmt.Errorf("flow name cannot be empty")
	}
	desc, _ := args["description"].(string)
	cat, _ := args["category"].(string)

	f := &entity.FlowDefinition{
		Name:        name,
		Description: desc,
		Category:    cat,
		Config:      getStringArg(args, "config"),
		FormSchema:  getStringArg(args, "form_schema"),
		Settings:    getStringArg(args, "settings"),
		Icon:        getStringArg(args, "icon"),
	}
	if err := r.flowModel.Create(f); err != nil {
		return "", fmt.Errorf("create flow failed: %w", err)
	}

	res := r.saveFlowNodesEdges(f.Id, args)
	if res.err != nil {
		// Clean up the created flow on validation error
		r.flowModel.Delete(f.Id)
		return "", res.err
	}
	nodeCount := res.nodeCnt
	edgeCount := res.edgeCnt

	// No nodes provided -- auto-adding start + end
	if nodeCount == 0 {
		startNode := &entity.FlowNode{
			FlowId:    f.Id,
			Type:      "start",
			Label:     "Start",
			PositionX: 100,
			PositionY: 100,
		}
		r.nodeModel.Create(startNode)
		endNode := &entity.FlowNode{
			FlowId:    f.Id,
			Type:      "end",
			Label:     "End",
			PositionX: 400,
			PositionY: 100,
		}
		r.nodeModel.Create(endNode)
		r.edgeModel.Create(&entity.FlowEdge{
			FlowId:       f.Id,
			SourceNodeId: startNode.Id,
			TargetNodeId: endNode.Id,
			SourceHandle: "output",
			TargetHandle: "input",
		})
		nodeCount = 2
		edgeCount = 1
	}

	return fmt.Sprintf("Flow created successfully!\nID: %d\nName: %s\nCategory: %s\nNodes: %d\nEdges: %d",
		f.Id, f.Name, f.Category, nodeCount, edgeCount), nil
}

func (r *ChatRunner) flowUpdate(args map[string]any) (string, error) {
	id, err := getUintArg(args, "flow_id")
	if err != nil {
		return "", err
	}
	f, err := r.flowModel.FindById(id)
	if err != nil {
		return "", fmt.Errorf("flow not found: ID=%d", id)
	}

	if v, ok := args["name"].(string); ok && v != "" {
		f.Name = v
	}
	if v, ok := args["description"].(string); ok && v != "" {
		f.Description = v
	}
	if v, ok := args["category"].(string); ok && v != "" {
		f.Category = v
	}
	if v := getStringArg(args, "config"); v != "" {
		f.Config = v
	}
	if v := getStringArg(args, "form_schema"); v != "" {
		f.FormSchema = v
	}
	if v := getStringArg(args, "settings"); v != "" {
		f.Settings = v
	}
	if v := getStringArg(args, "icon"); v != "" {
		f.Icon = v
	}

	nodeCount := 0
	edgeCount := 0

	// New nodes provided -- validate, then replace all nodes and edges
	res := r.saveFlowNodesEdges(id, args)
	if res.err != nil {
		return "", res.err
	}
	if res.hasNodes {
		// Validation passed, now replace: delete old, then re-save
		r.nodeModel.DeleteByFlowId(id)
		r.edgeModel.DeleteByFlowId(id)
		res = r.saveFlowNodesEdges(id, args)
		if res.err != nil {
			return "", res.err
		}
		nodeCount = res.nodeCnt
		edgeCount = res.edgeCnt
	}

	if err := r.flowModel.Update(f); err != nil {
		return "", fmt.Errorf("update flow failed: %w", err)
	}
	return fmt.Sprintf("Flow updated successfully!\nID: %d\nName: %s\nNodes: %d\nEdges: %d",
		f.Id, f.Name, nodeCount, edgeCount), nil
}

func (r *ChatRunner) flowDelete(args map[string]any) (string, error) {
	id, err := getUintArg(args, "flow_id")
	if err != nil {
		return "", err
	}
	if _, err := r.flowModel.FindById(id); err != nil {
		return "", fmt.Errorf("flow not found: ID=%d", id)
	}
	r.nodeModel.DeleteByFlowId(id)
	r.edgeModel.DeleteByFlowId(id)
	if err := r.flowModel.Delete(id); err != nil {
		return "", fmt.Errorf("delete failed: %w", err)
	}
	return fmt.Sprintf("Flow [ID:%d] deleted", id), nil
}

func getUintArg(args map[string]any, key string) (uint, error) {
	v, ok := args[key]
	if !ok {
		return 0, fmt.Errorf("missing parameter: %s", key)
	}
	switch n := v.(type) {
	case float64:
		return uint(n), nil
	case int:
		return uint(n), nil
	case int64:
		return uint(n), nil
	default:
		return 0, fmt.Errorf("parameter %s type error", key)
	}
}

func getStringArg(args map[string]any, key string) string {
	v, ok := args[key]
	if !ok {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	default:
		return fmt.Sprintf("%v", s)
	}
}

func nodeConfigSummary(n *entity.FlowNode) string {
	if n.Config == "" {
		return ""
	}
	var cfg map[string]any
	if err := json.Unmarshal([]byte(n.Config), &cfg); err != nil {
		return ""
	}
	parts := make([]string, 0, 3)
	if model, ok := cfg["model"].(string); ok && model != "" {
		parts = append(parts, "Model:"+model)
	}
	if prompt, ok := cfg["prompt"].(string); ok && prompt != "" {
		p := prompt
		if len(p) > 40 { p = p[:40] + "..." }
		parts = append(parts, "Prompt:"+p)
	}
	if tl, ok := cfg["thinking_level"].(string); ok && tl != "" && tl != "off" {
		parts = append(parts, "Thinking:"+tl)
	}
	if oft, ok := cfg["output_format_type"].(string); ok && oft != "" {
		parts = append(parts, "Format:"+oft)
	}
	if confirm, ok := cfg["confirm_only"].(bool); ok && confirm {
		parts = append(parts, "Confirm")
	}
	if itemsKey, ok := cfg["items_key"].(string); ok && itemsKey != "" {
		parts = append(parts, "Source:"+itemsKey)
	}
	if len(parts) == 0 { return "" }
	return " (" + strings.Join(parts, ", ") + ")"
}
