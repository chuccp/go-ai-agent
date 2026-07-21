package service

import (
	"github.com/chuccp/go-ai-agent/internal2/entity"
	"github.com/chuccp/go-ai-agent/internal2/flow/appstore"
	"github.com/chuccp/go-ai-agent/internal2/model"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/db"
	"github.com/chuccp/go-web-frame/log"
	"go.uber.org/zap"
)

type FlowService struct {
	context   *core.Context
	flowModel *model.FlowModel
	appStore  *appstore.Store
}

func (s *FlowService) Init(ctx *core.Context) error {
	s.context = ctx
	s.flowModel = core.GetModel[*model.FlowModel](ctx)

	appsPath := ctx.GetConfig().GetStringOrDefault("flow.appsPath", "./data/apps")
	s.appStore = appstore.New(appsPath)
	if err := s.appStore.EnsureBaseDir(); err != nil {
		log.Warn("Failed to create apps directory", zap.String("path", appsPath), zap.Error(err))
	}

	log.Info("FlowService initialized", zap.String("appsPath", appsPath))
	return nil
}

// GetAppStore exposes the app store for REST handlers and runners.
func (s *FlowService) GetAppStore() *appstore.Store {
	return s.appStore
}

// CreateFlow creates a new app directory, writes flow.json, and inserts a DB metadata row.
func (s *FlowService) CreateFlow(name, description, category, config, formSchema, settings, icon string, nodes []*entity.FlowNode, edges []*entity.FlowEdge) (*entity.FlowDefinition, error) {
	// 1. Create app directory
	appPath, err := s.appStore.CreateAppDir()
	if err != nil {
		return nil, err
	}

	// 2. Handle icon: generate SVG if none provided or emoji
	if icon == "" {
		iconFilename, err := s.appStore.SaveSVGIcon(appPath, name)
		if err != nil {
			_ = s.appStore.DeleteApp(appPath)
			return nil, err
		}
		icon = iconFilename
	}

	// 3. Write flow.json
	content := &appstore.FlowContent{
		Name:        name,
		Description: description,
		Category:    category,
		Config:      config,
		FormSchema:  formSchema,
		Settings:    settings,
		Icon:        icon,
		Nodes:       nodes,
		Edges:       edges,
	}
	if err := s.appStore.SaveFlow(appPath, content); err != nil {
		_ = s.appStore.DeleteApp(appPath)
		return nil, err
	}

	// 4. Insert DB metadata row
	f := &entity.FlowDefinition{
		Name:        name,
		Description: description,
		Category:    category,
		Path:        appPath,
		Icon:        icon,
		Config:      config,
		FormSchema:  formSchema,
		Settings:    settings,
	}
	err = s.context.GetTransaction().Exec(func(tx *db.DB) error {
		flowModel := core.GetReNewModel[*model.FlowModel](tx, s.context)
		return flowModel.Create(f)
	})
	if err != nil {
		_ = s.appStore.DeleteApp(appPath)
		return nil, err
	}
	return f, nil
}

// UpdateFlow loads the existing flow.json, merges changes, and saves.
func (s *FlowService) UpdateFlow(id uint, name, description, category, config, formSchema, settings, icon string, nodes []*entity.FlowNode, edges []*entity.FlowEdge) error {
	return s.context.GetTransaction().Exec(func(tx *db.DB) error {
		flowModel := core.GetReNewModel[*model.FlowModel](tx, s.context)
		f, err := flowModel.FindById(id)
		if err != nil {
			return err
		}

		// Load existing flow.json
		content, err := s.appStore.LoadFlow(f.Path)
		if err != nil {
			return err
		}

		// Merge non-empty fields
		if name != "" {
			f.Name = name
			content.Name = name
		}
		if description != "" {
			f.Description = description
			content.Description = description
		}
		if category != "" {
			f.Category = category
			content.Category = category
		}
		if config != "" {
			content.Config = config
		}
		if formSchema != "" {
			content.FormSchema = formSchema
		}
		if settings != "" {
			content.Settings = settings
		}
		if icon != "" {
			f.Icon = icon
			content.Icon = icon
		}
		if nodes != nil {
			content.Nodes = nodes
		}
		if edges != nil {
			content.Edges = edges
		}

		// Save flow.json
		if err := s.appStore.SaveFlow(f.Path, content); err != nil {
			return err
		}

		// Update DB row
		return flowModel.Update(f)
	})
}

// GetFlowDetail loads the flow definition and its on-disk content (nodes, edges, config).
func (s *FlowService) GetFlowDetail(id uint) (*entity.FlowDefinition, []*entity.FlowNode, []*entity.FlowEdge, error) {
	f, err := s.flowModel.FindById(id)
	if err != nil {
		return nil, nil, nil, err
	}

	content, err := s.appStore.LoadFlow(f.Path)
	if err != nil {
		return nil, nil, nil, err
	}

	// Hydrate on-disk fields onto the entity
	f.Config = content.Config
	f.FormSchema = content.FormSchema
	f.Settings = content.Settings

	return f, content.Nodes, content.Edges, nil
}

// DuplicateFlow copies the app directory and creates a new DB row.
func (s *FlowService) DuplicateFlow(id uint) (*entity.FlowDefinition, error) {
	var clone *entity.FlowDefinition
	err := s.context.GetTransaction().Exec(func(tx *db.DB) error {
		flowModel := core.GetReNewModel[*model.FlowModel](tx, s.context)

		src, err := flowModel.FindById(id)
		if err != nil {
			return err
		}

		// Copy app directory
		newPath, err := s.appStore.CopyApp(src.Path)
		if err != nil {
			return err
		}

		// Load and rename the copied flow.json
		content, err := s.appStore.LoadFlow(newPath)
		if err != nil {
			_ = s.appStore.DeleteApp(newPath)
			return err
		}
		content.Name = src.Name + " (copy)"
		if err := s.appStore.SaveFlow(newPath, content); err != nil {
			_ = s.appStore.DeleteApp(newPath)
			return err
		}

		clone = &entity.FlowDefinition{
			Name:        content.Name,
			Description: src.Description,
			Category:    src.Category,
			Path:        newPath,
			Icon:        src.Icon,
		}
		if err := flowModel.Create(clone); err != nil {
			_ = s.appStore.DeleteApp(newPath)
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return clone, nil
}

// DeleteFlow removes the app directory and the DB row.
func (s *FlowService) DeleteFlow(id uint) error {
	return s.context.GetTransaction().Exec(func(tx *db.DB) error {
		flowModel := core.GetReNewModel[*model.FlowModel](tx, s.context)
		f, err := flowModel.FindById(id)
		if err != nil {
			return err
		}
		// Delete app directory
		if f.Path != "" {
			_ = s.appStore.DeleteApp(f.Path)
		}
		return flowModel.Delete(id)
	})
}
