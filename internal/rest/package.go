package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/chuccp/go-ai-agent/internal/entity"
	"github.com/chuccp/go-ai-agent/internal/flow/export"
	"github.com/chuccp/go-ai-agent/internal/model"
	"github.com/chuccp/go-ai-agent/internal/service"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/web"
)

type PackageRest struct {
	context        *core.Context
	packageModel   *model.PackageModel
	packageResModel *model.PackageResourceModel
	packageCfgModel *model.PackageConfigModel
	flowModel      *model.FlowModel
	nodeModel      *model.FlowNodeModel
	edgeModel      *model.FlowEdgeModel
	skillModel     *model.SkillModel
	promptModel    *model.SkillPromptModel
	flowService    *service.FlowService
}

func NewPackageRest() *PackageRest { return &PackageRest{} }

type PackageDetail struct {
	*entity.Package
	Flows []*entity.FlowDefinition `json:"flows"`
	Skills []*entity.Skill         `json:"skills"`
}

func (r *PackageRest) Init(ctx *core.Context) error {
	r.context = ctx
	r.packageModel = core.GetModel[*model.PackageModel](ctx)
	r.packageResModel = core.GetModel[*model.PackageResourceModel](ctx)
	r.packageCfgModel = core.GetModel[*model.PackageConfigModel](ctx)
	r.flowModel = core.GetModel[*model.FlowModel](ctx)
	r.nodeModel = core.GetModel[*model.FlowNodeModel](ctx)
	r.edgeModel = core.GetModel[*model.FlowEdgeModel](ctx)
	r.skillModel = core.GetModel[*model.SkillModel](ctx)
	r.promptModel = core.GetModel[*model.SkillPromptModel](ctx)
	r.flowService = core.GetService[*service.FlowService](ctx)

	ctx.Get("/api/packages", r.listPackages)
	ctx.Get("/api/packages/:id", r.getPackage)
	ctx.Delete("/api/packages/:id", r.deletePackage)
	ctx.Get("/api/packages/:id/export", r.exportPackage)
	ctx.Post("/api/packages/import", r.importPackage)
	return nil
}

func (r *PackageRest) listPackages(req *web.Request) (any, error) {
	pkgs, err := r.packageModel.WithContext(req.Ctx()).Query().Order("updated_at desc").All()
	if err != nil {
		return nil, err
	}
	return web.Data(pkgs), nil
}

func (r *PackageRest) getPackage(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	pkg, err := r.packageModel.WithContext(req.Ctx()).FindByPK(id)
	if err != nil {
		return nil, err
	}
	flows, _ := r.flowModel.WithContext(req.Ctx()).Query().Where("package_id = ?", id).All()
	skills, _ := r.skillModel.WithContext(req.Ctx()).Query().Where("package_id = ?", id).All()
	return web.Data(&PackageDetail{Package: pkg, Flows: flows, Skills: skills}), nil
}

func (r *PackageRest) deletePackage(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	_ = r.packageCfgModel.WithContext(req.Ctx()).Delete().Where("package_id = ?", id).Delete()
	_ = r.packageResModel.WithContext(req.Ctx()).Delete().Where("package_id = ?", id).Delete()
	if skills, err := r.skillModel.WithContext(req.Ctx()).Query().Where("package_id = ?", id).All(); err == nil {
		for _, s := range skills {
			_ = r.promptModel.WithContext(req.Ctx()).Delete().Where("skill_id = ?", s.Id).Delete()
			_ = r.skillModel.WithContext(req.Ctx()).DeleteByPK(s.Id)
		}
	}
	if flows, err := r.flowModel.WithContext(req.Ctx()).Query().Where("package_id = ?", id).All(); err == nil {
		for _, f := range flows {
			_ = r.nodeModel.WithContext(req.Ctx()).DeleteByFlowId(f.Id)
			_ = r.edgeModel.WithContext(req.Ctx()).DeleteByFlowId(f.Id)
			_ = r.flowModel.WithContext(req.Ctx()).DeleteByPK(f.Id)
		}
	}
	if err := r.packageModel.WithContext(req.Ctx()).DeleteByPK(id); err != nil {
		return nil, err
	}
	return web.Ok("deleted"), nil
}

func (r *PackageRest) exportPackage(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	pkg, err := r.packageModel.WithContext(req.Ctx()).FindByPK(id)
	if err != nil {
		return nil, err
	}

	flows, err := r.flowModel.WithContext(req.Ctx()).Query().Where("package_id = ?", id).All()
	if err != nil || len(flows) == 0 {
		return nil, fmt.Errorf("no flows in package")
	}
	primary := flows[0]

	ns, _ := r.nodeModel.WithContext(req.Ctx()).FindByFlowId(primary.Id)
	es, _ := r.edgeModel.WithContext(req.Ctx()).FindByFlowId(primary.Id)
	fd := export.FlowData{
		Name:        primary.Name,
		Description: primary.Description,
		Category:    primary.Category,
		Config:      primary.Config,
		FormSchema:  primary.FormSchema,
		Settings:    primary.Settings,
		Icon:        primary.Icon,
		Nodes:       ns,
		Edges:       es,
	}

	skills, _ := r.skillModel.WithContext(req.Ctx()).Query().Where("package_id = ?", id).All()
	var skillDatas []export.SkillData
	for _, s := range skills {
		prompts, _ := r.promptModel.WithContext(req.Ctx()).Query().Where("skill_id = ?", s.Id).All()
		sd := export.SkillData{
			SkillId:      s.SkillId,
			Name:         s.Name,
			Version:      s.Version,
			Description:  s.Description,
			Icon:         s.Icon,
			Inputs:       s.Inputs,
			Outputs:      s.Outputs,
			DefaultModel: s.DefaultModel,
		}
		for _, p := range prompts {
			sd.Prompts = append(sd.Prompts, struct {
				Name    string `json:"name"`
				Content string `json:"content"`
			}{Name: p.Name, Content: p.Content})
		}
		skillDatas = append(skillDatas, sd)
	}

	resources := make(map[string][]byte)
	resItems, _ := r.packageResModel.WithContext(req.Ctx()).Query().Where("package_id = ?", id).All()
	for _, res := range resItems {
		resources[path.Base(res.Path)] = res.Content
	}

	cfgItems, _ := r.packageCfgModel.WithContext(req.Ctx()).Query().Where("package_id = ?", id).All()
	var cfgLines []string
	for _, c := range cfgItems {
		cfgLines = append(cfgLines, fmt.Sprintf("%s: %s", c.Key, c.Value))
	}
	config := []byte(strings.Join(cfgLines, "\n"))

	meta := export.Meta{
		Type:        export.PackageType,
		Version:     export.CurrentVersion,
		Kind:        pkg.Kind,
		Name:        pkg.Name,
		Description: pkg.Description,
		Icon:        pkg.Icon,
		PrimaryFlow: "flows/main/flow.json",
		Flows:       []string{"flows/main/flow.json"},
		Config:      "config/config.yml",
	}

	data, err := export.BuildFullPackage(meta, fd, skillDatas, resources, config)
	if err != nil {
		return nil, fmt.Errorf("export failed: %w", err)
	}

	fileName := strings.ReplaceAll(pkg.Name, " ", "_") + ".zip"
	resp := req.Response()
	resp.Header().Set("Content-Type", "application/zip")
	resp.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	_, _ = resp.Write(data)
	return nil, nil
}

func (r *PackageRest) importPackage(req *web.Request) (any, error) {
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
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	fp, err := export.ParseFullPackage(data)
	if err != nil {
		return nil, err
	}

	pkg := &entity.Package{
		PackageId:   fp.Meta.Name,
		Name:        fp.Meta.Name,
		Version:     fp.Meta.VersionStr,
		Description: fp.Meta.Description,
		Icon:        fp.Meta.Icon,
		Kind:        fp.Meta.Kind,
	}
	metaBytes, _ := json.Marshal(fp.Meta)
	pkg.Meta = string(metaBytes)
	if err := r.packageModel.WithContext(req.Ctx()).Save(pkg); err != nil {
		return nil, fmt.Errorf("failed to create package: %w", err)
	}

	flow := &entity.FlowDefinition{
		PackageId:   pkg.Id,
		Name:        fp.Flow.Name,
		Description: fp.Flow.Description,
		Category:    fp.Flow.Category,
		Config:      fp.Flow.Config,
		FormSchema:  fp.Flow.FormSchema,
		Settings:    fp.Flow.Settings,
		Icon:        fp.Flow.Icon,
	}
	if err := r.flowModel.WithContext(req.Ctx()).Save(flow); err != nil {
		return nil, fmt.Errorf("failed to create flow: %w", err)
	}

	idMap := make(map[uint]uint)
	for _, n := range fp.Flow.Nodes {
		oldID := n.Id
		n.Id = 0
		n.FlowId = flow.Id
		if err := r.nodeModel.WithContext(req.Ctx()).Save(n); err != nil {
			return nil, fmt.Errorf("failed to create node: %w", err)
		}
		idMap[oldID] = n.Id
	}
	for _, e := range fp.Flow.Edges {
		e.Id = 0
		e.FlowId = flow.Id
		if nid, ok := idMap[e.SourceNodeId]; ok {
			e.SourceNodeId = nid
		}
		if nid, ok := idMap[e.TargetNodeId]; ok {
			e.TargetNodeId = nid
		}
		if err := r.edgeModel.WithContext(req.Ctx()).Save(e); err != nil {
			return nil, fmt.Errorf("failed to create edge: %w", err)
		}
	}

	for _, s := range fp.Skills {
		skill := &entity.Skill{
			PackageId:    pkg.Id,
			SkillId:      s.SkillId,
			Name:         s.Name,
			Version:      s.Version,
			Description:  s.Description,
			Icon:         s.Icon,
			Inputs:       s.Inputs,
			Outputs:      s.Outputs,
			DefaultModel: s.DefaultModel,
			Enabled:      true,
		}
		if err := r.skillModel.WithContext(req.Ctx()).Save(skill); err != nil {
			continue
		}
		for _, p := range s.Prompts {
			_ = r.promptModel.WithContext(req.Ctx()).Save(&entity.SkillPrompt{
				SkillId: skill.Id,
				Name:    p.Name,
				Content: p.Content,
			})
		}
	}

	for name, content := range fp.Resources {
		_ = r.packageResModel.WithContext(req.Ctx()).Save(&entity.PackageResource{
			PackageId:   pkg.Id,
			Path:        "resources/" + name,
			ContentType: "application/octet-stream",
			Content:     content,
		})
	}

	return web.Data(pkg), nil
}
