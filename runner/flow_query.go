package runner

import (
	"fmt"
	"strings"

	"encoding/json"
	"github.com/chuccp/go-ai-agent/entity"
)

func (r *ChatRunner) flowList() (string, error) {
	flows, err := r.flowModel.List()
	if err != nil {
		return "", err
	}
	if len(flows) == 0 {
		return "暂无流程", nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("共有 %d 个流程：\n", len(flows)))
	for _, f := range flows {
		sb.WriteString(fmt.Sprintf("- [ID:%d] %s (%s)\n", f.Id, f.Name, f.Category))
	}
	return sb.String(), nil
}

func (r *ChatRunner) flowSearch(args map[string]any) (string, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return "", fmt.Errorf("搜索关键词不能为空")
	}

	flows, err := r.flowModel.List()
	if err != nil {
		return "", err
	}

	queryLower := strings.ToLower(query)

	// 第一轮：精确匹配（不区分大小写）
	var exact []*entity.FlowDefinition
	for _, f := range flows {
		if strings.ToLower(f.Name) == queryLower {
			exact = append(exact, f)
		}
	}
	if len(exact) == 1 {
		f := exact[0]
		return fmt.Sprintf("找到精确匹配：\n[ID:%d] %s (%s)\n请继续操作此流程。", f.Id, f.Name, f.Category), nil
	}
	if len(exact) > 1 {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("找到 %d 个同名流程，请用户确认是哪一个：\n", len(exact)))
		for _, f := range exact {
			sb.WriteString(fmt.Sprintf("- [ID:%d] %s (%s) 描述: %s\n", f.Id, f.Name, f.Category, f.Description))
		}
		sb.WriteString("请用户回复数字序号或ID来选择")
		return sb.String(), nil
	}

	// 第二轮：包含匹配
	var partial []*entity.FlowDefinition
	for _, f := range flows {
		if strings.Contains(strings.ToLower(f.Name), queryLower) {
			partial = append(partial, f)
		}
	}
	if len(partial) == 1 {
		f := partial[0]
		return fmt.Sprintf("找到匹配流程：\n[ID:%d] %s (%s)\n描述: %s\n请继续操作此流程。", f.Id, f.Name, f.Category, f.Description), nil
	}
	if len(partial) > 1 {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("找到 %d 个匹配 '%s' 的流程，请用户选择：\n", len(partial), query))
		for _, f := range partial {
			sb.WriteString(fmt.Sprintf("- [ID:%d] %s (%s)\n", f.Id, f.Name, f.Category))
		}
		sb.WriteString("请用户回复ID或名称来选择")
		return sb.String(), nil
	}

	// 未找到
	return fmt.Sprintf("未找到名称包含 '%s' 的流程。可用 list 查看所有流程。", query), nil
}

func (r *ChatRunner) flowGet(args map[string]any) (string, error) {
	id, err := getUintArg(args, "flow_id")
	if err != nil {
		return "", err
	}
	f, err := r.flowModel.FindById(id)
	if err != nil {
		return "", fmt.Errorf("未找到 ID=%d 的流程", id)
	}
	nodes, _ := r.nodeModel.FindByFlowId(id)
	edges, _ := r.edgeModel.FindByFlowId(id)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("流程 [ID:%d] %s | %s | %d节点 %d连线\n",
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
		return "", fmt.Errorf("流程名称不能为空")
	}
	desc, _ := args["description"].(string)
	cat, _ := args["category"].(string)

	f := &entity.FlowDefinition{
		Name:        name,
		Description: desc,
		Category:    cat,
	}
	if err := r.flowModel.Create(f); err != nil {
		return "", fmt.Errorf("创建流程失败: %w", err)
	}

	res := r.saveFlowNodesEdges(f.Id, args)
	nodeCount := res.nodeCnt
	edgeCount := res.edgeCnt

	// 没给节点或没给有效节点 → 自动补 start + end
	if nodeCount == 0 {
		startNode := &entity.FlowNode{
			FlowId:    f.Id,
			Type:      "start",
			Label:     "开始",
			PositionX: 100,
			PositionY: 100,
		}
		r.nodeModel.Create(startNode)
		endNode := &entity.FlowNode{
			FlowId:    f.Id,
			Type:      "end",
			Label:     "结束",
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

	return fmt.Sprintf("流程创建成功！\nID: %d\n名称: %s\n分类: %s\n节点数: %d\n连线数: %d",
		f.Id, f.Name, f.Category, nodeCount, edgeCount), nil
}

func (r *ChatRunner) flowUpdate(args map[string]any) (string, error) {
	id, err := getUintArg(args, "flow_id")
	if err != nil {
		return "", err
	}
	f, err := r.flowModel.FindById(id)
	if err != nil {
		return "", fmt.Errorf("未找到 ID=%d 的流程", id)
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

	nodeCount := 0
	edgeCount := 0

	// 提供了新节点 → 替换全部节点和连线
	res := r.saveFlowNodesEdges(id, args)
	if res.hasNodes {
		r.nodeModel.DeleteByFlowId(id)
		r.edgeModel.DeleteByFlowId(id)
		res = r.saveFlowNodesEdges(id, args)
		nodeCount = res.nodeCnt
		edgeCount = res.edgeCnt
	}

	if err := r.flowModel.Update(f); err != nil {
		return "", fmt.Errorf("更新流程失败: %w", err)
	}
	return fmt.Sprintf("流程更新成功！\nID: %d\n名称: %s\n节点数: %d\n连线数: %d",
		f.Id, f.Name, nodeCount, edgeCount), nil
}

func (r *ChatRunner) flowDelete(args map[string]any) (string, error) {
	id, err := getUintArg(args, "flow_id")
	if err != nil {
		return "", err
	}
	if _, err := r.flowModel.FindById(id); err != nil {
		return "", fmt.Errorf("未找到 ID=%d 的流程", id)
	}
	r.nodeModel.DeleteByFlowId(id)
	r.edgeModel.DeleteByFlowId(id)
	if err := r.flowModel.Delete(id); err != nil {
		return "", fmt.Errorf("删除失败: %w", err)
	}
	return fmt.Sprintf("流程 [ID:%d] 已删除", id), nil
}

func getUintArg(args map[string]any, key string) (uint, error) {
	v, ok := args[key]
	if !ok {
		return 0, fmt.Errorf("缺少参数 %s", key)
	}
	switch n := v.(type) {
	case float64:
		return uint(n), nil
	case int:
		return uint(n), nil
	case int64:
		return uint(n), nil
	default:
		return 0, fmt.Errorf("参数 %s 类型错误", key)
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
		parts = append(parts, "模型:"+model)
	}
	if prompt, ok := cfg["prompt"].(string); ok && prompt != "" {
		p := prompt
		if len(p) > 40 { p = p[:40] + "..." }
		parts = append(parts, "提示:"+p)
	}
	if tl, ok := cfg["thinking_level"].(string); ok && tl != "" && tl != "off" {
		parts = append(parts, "思考:"+tl)
	}
	if oft, ok := cfg["output_format_type"].(string); ok && oft != "" {
		parts = append(parts, "格式:"+oft)
	}
	if confirm, ok := cfg["confirm_only"].(bool); ok && confirm {
		parts = append(parts, "确认")
	}
	if itemsKey, ok := cfg["items_key"].(string); ok && itemsKey != "" {
		parts = append(parts, "来源:"+itemsKey)
	}
	if len(parts) == 0 { return "" }
	return " (" + strings.Join(parts, ", ") + ")"
}
