package rest

import (
	"github.com/bytedance/sonic"
	"strconv"

	"github.com/chuccp/go-ai-agent/entity"
	flowModel "github.com/chuccp/go-ai-agent/model"
	"github.com/chuccp/go-ai-agent/runner"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/web"
)

type FlowRest struct {
	context        *core.Context
	flowModel      *flowModel.FlowModel
	nodeModel      *flowModel.FlowNodeModel
	edgeModel      *flowModel.FlowEdgeModel
	executionModel *flowModel.FlowExecutionModel
	flowRunner     *runner.FlowRunner
}

func NewFlowRest(fr *runner.FlowRunner) *FlowRest { return &FlowRest{flowRunner: fr} }

type FlowDetail struct {
	*entity.FlowDefinition
	Nodes []*entity.FlowNode `json:"nodes"`
	Edges []*entity.FlowEdge `json:"edges"`
}

func (r *FlowRest) Init(ctx *core.Context) error {
	r.context = ctx
	r.flowModel = core.GetModel[*flowModel.FlowModel](ctx)
	r.nodeModel = core.GetModel[*flowModel.FlowNodeModel](ctx)
	r.edgeModel = core.GetModel[*flowModel.FlowEdgeModel](ctx)
	r.executionModel = core.GetModel[*flowModel.FlowExecutionModel](ctx)
	if r.flowRunner != nil { r.flowRunner.Init(ctx) }

	r.context.Get("/api/flows", r.listFlows)
	r.context.Post("/api/flows", r.createFlow)
	r.context.Get("/api/flows/node-types", r.listNodeTypes)
	r.context.Get("/api/flows/executions", r.listExecutions)
	r.context.Get("/api/flows/:id", r.getFlow)
	r.context.Put("/api/flows/:id", r.updateFlow)
	r.context.Delete("/api/flows/:id", r.deleteFlow)
	r.context.Post("/api/flows/:id/duplicate", r.duplicateFlow)
	r.context.Post("/api/flows/:id/execute", r.executeFlow)

	return nil
}

func (r *FlowRest) listFlows(req *web.Request) (any, error) {
	c := req.Query("category")
	var fs []*entity.FlowDefinition
	var err error
	if c != "" { fs, err = r.flowModel.ListByCategory(c) } else { fs, err = r.flowModel.List() }
	if err != nil { return nil, err }
	return web.Data(fs), nil
}

func (r *FlowRest) createFlow(req *web.Request) (any, error) {
	j, _ := req.Json()
	n := j.GetString("name")
	if n == "" { n = "Untitled Flow" }
	f := &entity.FlowDefinition{Name: n, Description: j.GetString("description"), Category: j.GetString("category"), Config: j.GetString("config")}
	if err := r.flowModel.Create(f); err != nil { return nil, err }
	r.saveNodesAndEdges(f.Id, map[string]any(*j))
	return web.Data(f), nil
}

func (r *FlowRest) getFlow(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	f, err := r.flowModel.FindById(id)
	if err != nil { return nil, err }
	ns, _ := r.nodeModel.FindByFlowId(id)
	es, _ := r.edgeModel.FindByFlowId(id)
	return web.Data(&FlowDetail{FlowDefinition: f, Nodes: ns, Edges: es}), nil
}

func (r *FlowRest) updateFlow(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	f, err := r.flowModel.FindById(id)
	if err != nil { return nil, err }
	j, _ := req.Json()
	if v := j.GetString("name"); v != "" { f.Name = v }
	if v := j.GetString("description"); v != "" { f.Description = v }
	if v := j.GetString("category"); v != "" { f.Category = v }
	if v := j.GetString("config"); v != "" { f.Config = v }
	r.saveNodesAndEdges(id, map[string]any(*j))
	if err := r.flowModel.Update(f); err != nil { return nil, err }
	return web.Data(f), nil
}

func (r *FlowRest) duplicateFlow(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	src, err := r.flowModel.FindById(id)
	if err != nil { return nil, err }
	clone := &entity.FlowDefinition{Name: src.Name + " (copy)", Description: src.Description, Category: src.Category, Config: src.Config}
	if err := r.flowModel.Create(clone); err != nil { return nil, err }
	sns, _ := r.nodeModel.FindByFlowId(id)
	idMap := make(map[uint]uint)
	for _, n := range sns {
		oid := n.Id; n.Id = 0; n.FlowId = clone.Id
		r.nodeModel.Create(n)
		idMap[oid] = n.Id
	}
	ses, _ := r.edgeModel.FindByFlowId(id)
	for _, e := range ses {
		e.Id = 0; e.FlowId = clone.Id
		if nid, ok := idMap[e.SourceNodeId]; ok { e.SourceNodeId = nid }
		if nid, ok := idMap[e.TargetNodeId]; ok { e.TargetNodeId = nid }
		r.edgeModel.Create(e)
	}
	return web.Data(clone), nil
}

func (r *FlowRest) deleteFlow(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	r.nodeModel.DeleteByFlowId(id)
	r.edgeModel.DeleteByFlowId(id)
	if err := r.flowModel.Delete(id); err != nil { return nil, err }
	return web.Ok("deleted"), nil
}

func (r *FlowRest) saveNodesAndEdges(flowId uint, jsonMap map[string]any) {
	var ns []*entity.FlowNode
	idMap := make(map[uint]uint)
	if nodesRaw, ok := jsonMap["nodes"]; ok {
		nodesBytes, _ := sonic.Marshal(nodesRaw)
		if sonic.Unmarshal(nodesBytes, &ns) != nil { return }
		r.nodeModel.DeleteByFlowId(flowId)
		oldIds := make([]uint, len(ns))
		for i, n := range ns {
			oldIds[i] = n.Id
			n.Id = 0
			n.FlowId = flowId
			r.nodeModel.Create(n)
		}
		// 重新加载获取 DB 真实 ID，建立 oldId→newId 映射
		savedNodes, _ := r.nodeModel.FindByFlowId(flowId)
		for i, sn := range savedNodes {
			if i < len(oldIds) {
				idMap[oldIds[i]] = sn.Id
			}
		}
	}
	if edgesRaw, ok := jsonMap["edges"]; ok {
		edgesBytes, _ := sonic.Marshal(edgesRaw)
		var es []*entity.FlowEdge
		if sonic.Unmarshal(edgesBytes, &es) != nil { return }
		r.edgeModel.DeleteByFlowId(flowId)
		for _, e := range es {
			e.Id = 0
			e.FlowId = flowId
			if nid, ok := idMap[e.SourceNodeId]; ok { e.SourceNodeId = nid }
			if nid, ok := idMap[e.TargetNodeId]; ok { e.TargetNodeId = nid }
			r.edgeModel.Create(e)
		}
	}
}

func (r *FlowRest) executeFlow(req *web.Request) (any, error) {
	fid := req.ParamUint("id")
	j, _ := req.Json()
	sid := uint(j.GetInt("session_id"))
	exec := &entity.FlowExecution{FlowId: fid, SessionId: sid, Status: "created", Context: "{}"}
	if err := r.executionModel.Create(exec); err != nil { return nil, err }
	return web.Data(map[string]any{"execution_id": exec.Id, "flow_id": fid}), nil
}

func (r *FlowRest) listExecutions(req *web.Request) (any, error) {
	var es []*entity.FlowExecution
	var err error
	if fid := req.Query("flow_id"); fid != "" {
		id, _ := strconv.ParseUint(fid, 10, 64)
		es, err = r.executionModel.FindByFlowId(uint(id))
	} else if sid := req.Query("session_id"); sid != "" {
		id, _ := strconv.ParseUint(sid, 10, 64)
		es, err = r.executionModel.FindBySessionId(uint(id))
	} else {
		return web.Data([]any{}), nil
	}
	if err != nil { return nil, err }
	return web.Data(es), nil
}

func (r *FlowRest) listNodeTypes(req *web.Request) (any, error) {
	types := []map[string]any{
		{"type": "start", "label": "开始", "description": "流程入口"},
		{"type": "end", "label": "结束", "description": "流程出口"},
		{"type": "llm", "label": "LLM 调用", "description": "调用大语言模型生成内容"},
		{"type": "user_input", "label": "用户输入", "description": "等待用户确认或输入"},
		{"type": "split", "label": "文本拆分", "description": "按分隔符拆分文本为JSON数组"},
		{"type": "condition", "label": "条件分支", "description": "if/else 条件判断"},
		{"type": "transform", "label": "数据变换", "description": "Go模板自定义数据变换"},
		{"type": "for_each", "label": "ForEach 遍历", "description": "遍历JSON数组逐项处理"},
		{"type": "iterator", "label": "按序迭代", "description": "逐项顺序处理，失败跳过"},
		{"type": "loop", "label": "循环执行", "description": "重复执行直到条件满足"},
		{"type": "script", "label": "Script 脚本", "description": "Starlark Python 自定义代码"},
		{"type": "image_gen", "label": "图片生成", "description": "调用图片生成模型"},
		{"type": "audio_gen", "label": "音频生成", "description": "调用语音合成模型"},
		{"type": "video_gen", "label": "视频生成", "description": "调用视频生成模型"},
	}
	return web.Data(types), nil
}
