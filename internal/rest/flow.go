package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/chuccp/go-ai-agent/internal/entity"
	"github.com/chuccp/go-ai-agent/internal/flow/export"
	flowModel "github.com/chuccp/go-ai-agent/internal/model"
	"github.com/chuccp/go-ai-agent/internal/service"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/web"
)

type FlowRest struct {
	context        *core.Context
	flowModel      *flowModel.FlowModel
	nodeModel      *flowModel.FlowNodeModel
	edgeModel      *flowModel.FlowEdgeModel
	executionModel *flowModel.FlowExecutionModel
	flowService    *service.FlowService
}

func NewFlowRest() *FlowRest { return &FlowRest{} }

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
	r.flowService = core.GetService[*service.FlowService](ctx)

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
	if c != "" { fs, err = r.flowModel.WithContext(req.Ctx()).ListByCategory(c) } else { fs, err = r.flowModel.WithContext(req.Ctx()).List() }
	if err != nil { return nil, err }
	return web.Data(fs), nil
}

func (r *FlowRest) createFlow(req *web.Request) (any, error) {
	j, _ := req.Json()
	name := j.GetString("name")
	if name == "" { name = "Untitled Flow" }
	nodes, edges := extractNodesAndEdges(map[string]any(*j))
	f, err := r.flowService.CreateFlow(name, j.GetString("description"), j.GetString("category"), j.GetString("config"), nodes, edges)
	if err != nil { return nil, err }
	return web.Data(f), nil
}

func (r *FlowRest) getFlow(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	f, err := r.flowModel.WithContext(req.Ctx()).FindById(id)
	if err != nil { return nil, err }
	ns, _ := r.nodeModel.WithContext(req.Ctx()).FindByFlowId(id)
	es, _ := r.edgeModel.WithContext(req.Ctx()).FindByFlowId(id)
	return web.Data(&FlowDetail{FlowDefinition: f, Nodes: ns, Edges: es}), nil
}

func (r *FlowRest) updateFlow(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	j, _ := req.Json()
	nodes, edges := extractNodesAndEdges(map[string]any(*j))
	if err := r.flowService.UpdateFlow(id, j.GetString("name"), j.GetString("description"), j.GetString("category"), j.GetString("config"), nodes, edges); err != nil {
		return nil, err
	}
	f, _ := r.flowModel.WithContext(req.Ctx()).FindById(id)
	return web.Data(f), nil
}

func (r *FlowRest) duplicateFlow(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	clone, err := r.flowService.DuplicateFlow(id)
	if err != nil { return nil, err }
	return web.Data(clone), nil
}

func (r *FlowRest) deleteFlow(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	if err := r.flowService.DeleteFlow(id); err != nil { return nil, err }
	return web.Ok("deleted"), nil
}

func extractNodesAndEdges(jsonMap map[string]any) ([]*entity.FlowNode, []*entity.FlowEdge) {
	var ns []*entity.FlowNode
	if nodesRaw, ok := jsonMap["nodes"]; ok {
		nodesBytes, _ := json.Marshal(nodesRaw)
		json.Unmarshal(nodesBytes, &ns)
	}
	var es []*entity.FlowEdge
	if edgesRaw, ok := jsonMap["edges"]; ok {
		edgesBytes, _ := json.Marshal(edgesRaw)
		json.Unmarshal(edgesBytes, &es)
	}
	return ns, es
}

func (r *FlowRest) executeFlow(req *web.Request) (any, error) {
	fid := req.ParamUint("id")
	j, _ := req.Json()
	sid := uint(j.GetInt("session_id"))
	exec := &entity.FlowExecution{FlowId: fid, SessionId: sid, Status: "created", Context: "{}"}
	if err := r.executionModel.WithContext(req.Ctx()).Create(exec); err != nil { return nil, err }
	return web.Data(map[string]any{"execution_id": exec.Id, "flow_id": fid}), nil
}

func (r *FlowRest) listExecutions(req *web.Request) (any, error) {
	var es []*entity.FlowExecution
	var err error
	if fid := req.Query("flow_id"); fid != "" {
		id, _ := strconv.ParseUint(fid, 10, 64)
		es, err = r.executionModel.WithContext(req.Ctx()).FindByFlowId(uint(id))
	} else if sid := req.Query("session_id"); sid != "" {
		id, _ := strconv.ParseUint(sid, 10, 64)
		es, err = r.executionModel.WithContext(req.Ctx()).FindBySessionId(uint(id))
	} else {
		return web.Data([]any{}), nil
	}
	if err != nil { return nil, err }
	return web.Data(es), nil
}

// exportFlow downloads a flow as a ZIP package.
func (r *FlowRest) exportFlow(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	f, err := r.flowModel.WithContext(req.Ctx()).FindById(id)
	if err != nil {
		return nil, err
	}
	ns, _ := r.nodeModel.WithContext(req.Ctx()).FindByFlowId(id)
	es, _ := r.edgeModel.WithContext(req.Ctx()).FindByFlowId(id)

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
	if err := r.flowModel.WithContext(req.Ctx()).Create(flow); err != nil {
		return nil, fmt.Errorf("failed to create flow: %w", err)
	}

	// Build old ID → new ID mapping while inserting nodes
	idMap := make(map[uint]uint)
	for _, n := range fd.Nodes {
		oldID := n.Id
		n.Id = 0
		n.FlowId = flow.Id
		if err := r.nodeModel.WithContext(req.Ctx()).Create(n); err != nil {
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
		if err := r.edgeModel.WithContext(req.Ctx()).Create(e); err != nil {
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
