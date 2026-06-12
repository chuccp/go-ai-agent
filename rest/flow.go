package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/chuccp/go-ai-agent/entity"
	"github.com/chuccp/go-ai-agent/flow/export"
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
	r.context.Get("/api/flows/:id/export", r.exportFlow)
	r.context.Post("/api/flows/import", r.importFlow)

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
		nodesBytes, _ := json.Marshal(nodesRaw)
		if json.Unmarshal(nodesBytes, &ns) != nil { return }
		r.nodeModel.DeleteByFlowId(flowId)
		oldIds := make([]uint, len(ns))
		for i, n := range ns {
			oldIds[i] = n.Id
			n.Id = 0
			n.FlowId = flowId
			r.nodeModel.Create(n)
		}
		// Reload to get real DB IDs, build oldId->newId mapping
		savedNodes, _ := r.nodeModel.FindByFlowId(flowId)
		for i, sn := range savedNodes {
			if i < len(oldIds) {
				idMap[oldIds[i]] = sn.Id
			}
		}
	}
	if edgesRaw, ok := jsonMap["edges"]; ok {
		edgesBytes, _ := json.Marshal(edgesRaw)
		var es []*entity.FlowEdge
		if json.Unmarshal(edgesBytes, &es) != nil { return }
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

// exportFlow downloads a flow as a ZIP package.
func (r *FlowRest) exportFlow(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	f, err := r.flowModel.FindById(id)
	if err != nil {
		return nil, err
	}
	ns, _ := r.nodeModel.FindByFlowId(id)
	es, _ := r.edgeModel.FindByFlowId(id)

	data, err := export.BuildFlowPackage(f.Name, ns, es, f.Description, f.Category)
	if err != nil {
		return nil, fmt.Errorf("export failed: %w", err)
	}

	fileName := strings.ReplaceAll(f.Name, " ", "_") + ".zip"
	resp := req.Response()
	resp.Header().Set("Content-Type", "application/zip")
	resp.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	resp.Header().Set("Content-Transfer-Encoding", "binary")
	_, _ = resp.Write(data)
	return nil, nil
}

// importFlow imports a flow from an uploaded ZIP package.
func (r *FlowRest) importFlow(req *web.Request) (any, error) {
	form, err := req.MultipartForm()
	if err != nil {
		return nil, fmt.Errorf("failed to parse upload form: %w", err)
	}

	files := form.File["file"]
	if len(files) == 0 {
		return nil, fmt.Errorf("no file uploaded (field name: file)")
	}

	file, err := files[0].Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}

	fd, err := export.ParseFlowPackage(data)
	if err != nil {
		return nil, err
	}

	flow := &entity.FlowDefinition{
		Name:        fd.Name,
		Description: fd.Description,
		Category:    fd.Category,
	}
	if err := r.flowModel.Create(flow); err != nil {
		return nil, fmt.Errorf("failed to create flow: %w", err)
	}

	// Build old ID → new ID mapping while inserting nodes
	idMap := make(map[uint]uint)
	for _, n := range fd.Nodes {
		oldID := n.Id
		n.Id = 0
		n.FlowId = flow.Id
		if err := r.nodeModel.Create(n); err != nil {
			return nil, fmt.Errorf("failed to create node: %w", err)
		}
		idMap[oldID] = n.Id
	}

	for _, e := range fd.Edges {
		e.Id = 0
		e.FlowId = flow.Id
		if nid, ok := idMap[e.SourceNodeId]; ok {
			e.SourceNodeId = nid
		}
		if nid, ok := idMap[e.TargetNodeId]; ok {
			e.TargetNodeId = nid
		}
		if err := r.edgeModel.Create(e); err != nil {
			return nil, fmt.Errorf("failed to create edge: %w", err)
		}
	}

	return web.Data(flow), nil
}

func (r *FlowRest) listNodeTypes(req *web.Request) (any, error) {
	types := []map[string]any{
		{"type": "start", "label": "Start", "description": "Flow entry point"},
		{"type": "end", "label": "End", "description": "Flow exit point"},
		{"type": "llm", "label": "LLM Call", "description": "Call a large language model to generate content"},
		{"type": "user_input", "label": "User Input", "description": "Wait for user confirmation or input"},
		{"type": "split", "label": "Text Split", "description": "Split text by delimiter into a JSON array"},
		{"type": "condition", "label": "Condition", "description": "if/else conditional branch (Starlark boolean expression)"},
		{"type": "switch", "label": "Switch", "description": "Multi-branch switch (Starlark string expression)"},
		{"type": "transform", "label": "Transform", "description": "Go template custom data transformation"},
		{"type": "for_each", "label": "ForEach", "description": "Split array and invoke a function for each item in parallel"},
		{"type": "iterator", "label": "Iterator", "description": "Split array and invoke a function for each item sequentially"},
		{"type": "loop", "label": "Loop", "description": "Repeat execution until condition is met"},
		{"type": "execute", "label": "Execute", "description": "Run a local shell command"},
		{"type": "script", "label": "Script", "description": "Starlark Python custom code"},
		{"type": "image_gen", "label": "Image Gen", "description": "Call image generation model"},
		{"type": "audio_gen", "label": "Audio Gen", "description": "Call speech synthesis model"},
		{"type": "video_gen", "label": "Video Gen", "description": "Call video generation model"},
	}
	return web.Data(types), nil
}
