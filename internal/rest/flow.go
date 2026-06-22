package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/chuccp/go-ai-agent/internal/entity"
	"github.com/chuccp/go-ai-agent/internal/flow/appstore"
	flowModel "github.com/chuccp/go-ai-agent/internal/model"
	"github.com/chuccp/go-ai-agent/internal/service"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/web"
)

type FlowRest struct {
	context     *core.Context
	flowModel   *flowModel.FlowModel
	flowService *service.FlowService
	appStore    *appstore.Store
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
	r.flowService = core.GetService[*service.FlowService](ctx)
	if r.flowService != nil {
		r.appStore = r.flowService.GetAppStore()
	}

	r.context.Get("/api/flows", r.listFlows)
	r.context.Post("/api/flows", r.createFlow)
	r.context.Get("/api/flows/node-types", r.listNodeTypes)
	r.context.Get("/api/flows/:id", r.getFlow)
	r.context.Put("/api/flows/:id", r.updateFlow)
	r.context.Delete("/api/flows/:id", r.deleteFlow)
	r.context.Post("/api/flows/:id/duplicate", r.duplicateFlow)
	r.context.Get("/api/flows/:id/export", r.exportFlow)
	r.context.Post("/api/flows/import", r.importFlow)
	r.context.Post("/api/flows/:id/icon", r.uploadIcon)
	r.context.Get("/api/flows/:id/icon", r.getIcon)

	return nil
}

func (r *FlowRest) listFlows(req *web.Request) (any, error) {
	c := req.Query("category")
	var fs []*entity.FlowDefinition
	var err error
	if c != "" {
		fs, err = r.flowModel.WithContext(req.Ctx()).ListByCategory(c)
	} else {
		fs, err = r.flowModel.WithContext(req.Ctx()).List()
	}
	if err != nil {
		return nil, err
	}
	return web.Data(fs), nil
}

func (r *FlowRest) createFlow(req *web.Request) (any, error) {
	j, _ := req.Json()
	name := j.GetString("name")
	if name == "" {
		name = "Untitled Flow"
	}
	nodes, edges := extractNodesAndEdges(map[string]any(*j))
	f, err := r.flowService.CreateFlow(
		name,
		j.GetString("description"),
		j.GetString("category"),
		j.GetString("config"),
		j.GetString("form_schema"),
		j.GetString("settings"),
		j.GetString("icon"),
		nodes, edges,
	)
	if err != nil {
		return nil, err
	}
	return web.Data(f), nil
}

func (r *FlowRest) getFlow(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	f, nodes, edges, err := r.flowService.GetFlowDetail(id)
	if err != nil {
		return nil, err
	}
	return web.Data(&FlowDetail{FlowDefinition: f, Nodes: nodes, Edges: edges}), nil
}

func (r *FlowRest) updateFlow(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	j, _ := req.Json()
	nodes, edges := extractNodesAndEdges(map[string]any(*j))
	if err := r.flowService.UpdateFlow(
		id,
		j.GetString("name"),
		j.GetString("description"),
		j.GetString("category"),
		j.GetString("config"),
		j.GetString("form_schema"),
		j.GetString("settings"),
		j.GetString("icon"),
		nodes, edges,
	); err != nil {
		return nil, err
	}
	f, _, _, _ := r.flowService.GetFlowDetail(id)
	return web.Data(f), nil
}

func (r *FlowRest) duplicateFlow(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	clone, err := r.flowService.DuplicateFlow(id)
	if err != nil {
		return nil, err
	}
	return web.Data(clone), nil
}

func (r *FlowRest) deleteFlow(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	if err := r.flowService.DeleteFlow(id); err != nil {
		return nil, err
	}
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


// exportFlow downloads a flow as a ZIP package.
func (r *FlowRest) exportFlow(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	f, err := r.flowModel.WithContext(req.Ctx()).FindById(id)
	if err != nil {
		return nil, err
	}

	data, err := r.appStore.PackageToZip(f.Path)
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

	const maxUploadSize = 10 << 20 // 10 MB
	if files[0].Size > maxUploadSize {
		return nil, fmt.Errorf("uploaded file too large: %d bytes (max %d bytes)", files[0].Size, maxUploadSize)
	}

	file, err := files[0].Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxUploadSize+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}
	if int64(len(data)) > maxUploadSize {
		return nil, fmt.Errorf("uploaded file too large (max %d bytes)", maxUploadSize)
	}

	newPath, content, err := r.appStore.InstallFromZip(data)
	if err != nil {
		return nil, err
	}

	flow := &entity.FlowDefinition{
		Name:        content.Name,
		Description: content.Description,
		Category:    content.Category,
		Path:        newPath,
		Icon:        content.Icon,
		Config:      content.Config,
		FormSchema:  content.FormSchema,
		Settings:    content.Settings,
	}
	if err := r.flowModel.WithContext(req.Ctx()).Create(flow); err != nil {
		_ = r.appStore.DeleteApp(newPath)
		return nil, fmt.Errorf("failed to create flow: %w", err)
	}

	return web.Data(flow), nil
}

// uploadIcon handles icon file upload for a flow.
func (r *FlowRest) uploadIcon(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	f, err := r.flowModel.WithContext(req.Ctx()).FindById(id)
	if err != nil {
		return nil, err
	}

	form, err := req.MultipartForm()
	if err != nil {
		return nil, fmt.Errorf("failed to parse upload form: %w", err)
	}

	files := form.File["file"]
	if len(files) == 0 {
		return nil, fmt.Errorf("no file uploaded (field name: file)")
	}

	const maxIconSize = 5 << 20 // 5 MB
	if files[0].Size > maxIconSize {
		return nil, fmt.Errorf("icon file too large: %d bytes (max %d bytes)", files[0].Size, maxIconSize)
	}

	// Validate file type
	ext := strings.ToLower(strings.TrimPrefix(files[0].Filename, "."))
	if ext == "" && files[0].Header != nil {
		ext = strings.ToLower(strings.TrimPrefix(files[0].Header.Get("Content-Type"), "image/"))
	}
	validExts := map[string]bool{"png": true, "jpg": true, "jpeg": true, "svg": true, "webp": true, "gif": true}
	if !validExts[ext] {
		return nil, fmt.Errorf("unsupported icon format: %s (allowed: png, jpg, svg, webp, gif)", ext)
	}

	file, err := files[0].Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxIconSize+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read icon data: %w", err)
	}

	iconFilename, err := r.appStore.SaveIcon(f.Path, data, ext)
	if err != nil {
		return nil, fmt.Errorf("failed to save icon: %w", err)
	}

	f.Icon = iconFilename
	if err := r.flowModel.WithContext(req.Ctx()).Update(f); err != nil {
		return nil, fmt.Errorf("failed to update flow icon: %w", err)
	}

	return web.Data(map[string]any{"icon": iconFilename}), nil
}

// getIcon serves the icon file for a flow.
func (r *FlowRest) getIcon(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	f, err := r.flowModel.WithContext(req.Ctx()).FindById(id)
	if err != nil {
		return nil, err
	}

	data, mimeType, err := r.appStore.ReadIcon(f.Path, f.Icon)
	if err != nil {
		resp := req.Response()
		resp.WriteHeader(404)
		return nil, nil
	}

	resp := req.Response()
	resp.Header().Set("Content-Type", mimeType)
	resp.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = resp.Write(data)
	return nil, nil
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
