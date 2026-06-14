package service

import (
	"github.com/chuccp/go-ai-agent/entity"
	"github.com/chuccp/go-ai-agent/model"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/db"
)

type FlowService struct {
	context   *core.Context
	flowModel *model.FlowModel
	nodeModel *model.FlowNodeModel
	edgeModel *model.FlowEdgeModel
}

func (s *FlowService) Init(ctx *core.Context) error {
	s.context = ctx
	s.flowModel = core.GetModel[*model.FlowModel](ctx)
	s.nodeModel = core.GetModel[*model.FlowNodeModel](ctx)
	s.edgeModel = core.GetModel[*model.FlowEdgeModel](ctx)
	return nil
}

func (s *FlowService) CreateFlow(name, description, category, config string, nodes []*entity.FlowNode, edges []*entity.FlowEdge) (*entity.FlowDefinition, error) {
	f := &entity.FlowDefinition{Name: name, Description: description, Category: category, Config: config}
	err := s.context.GetTransaction().Exec(func(tx *db.DB) error {
		flowModel := core.GetReNewModel[*model.FlowModel](tx, s.context)
		if err := flowModel.Create(f); err != nil {
			return err
		}
		return s.saveNodesAndEdges(tx, f.Id, nodes, edges)
	})
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (s *FlowService) UpdateFlow(id uint, name, description, category, config string, nodes []*entity.FlowNode, edges []*entity.FlowEdge) error {
	return s.context.GetTransaction().Exec(func(tx *db.DB) error {
		flowModel := core.GetReNewModel[*model.FlowModel](tx, s.context)
		f, err := flowModel.FindById(id)
		if err != nil {
			return err
		}
		if name != "" {
			f.Name = name
		}
		if description != "" {
			f.Description = description
		}
		if category != "" {
			f.Category = category
		}
		if config != "" {
			f.Config = config
		}
		if err := flowModel.Update(f); err != nil {
			return err
		}
		return s.saveNodesAndEdges(tx, id, nodes, edges)
	})
}

func (s *FlowService) DuplicateFlow(id uint) (*entity.FlowDefinition, error) {
	var clone *entity.FlowDefinition
	err := s.context.GetTransaction().Exec(func(tx *db.DB) error {
		flowModel := core.GetReNewModel[*model.FlowModel](tx, s.context)
		nodeModel := core.GetReNewModel[*model.FlowNodeModel](tx, s.context)
		edgeModel := core.GetReNewModel[*model.FlowEdgeModel](tx, s.context)

		src, err := flowModel.FindById(id)
		if err != nil {
			return err
		}
		clone = &entity.FlowDefinition{Name: src.Name + " (copy)", Description: src.Description, Category: src.Category, Config: src.Config}
		if err := flowModel.Create(clone); err != nil {
			return err
		}

		sns, err := nodeModel.FindByFlowId(id)
		if err != nil {
			return err
		}
		idMap := make(map[uint]uint)
		for _, n := range sns {
			oid := n.Id
			n.Id = 0
			n.FlowId = clone.Id
			if err := nodeModel.Create(n); err != nil {
				return err
			}
			idMap[oid] = n.Id
		}

		ses, err := edgeModel.FindByFlowId(id)
		if err != nil {
			return err
		}
		for _, e := range ses {
			e.Id = 0
			e.FlowId = clone.Id
			if nid, ok := idMap[e.SourceNodeId]; ok {
				e.SourceNodeId = nid
			}
			if nid, ok := idMap[e.TargetNodeId]; ok {
				e.TargetNodeId = nid
			}
			if err := edgeModel.Create(e); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return clone, nil
}

func (s *FlowService) DeleteFlow(id uint) error {
	return s.context.GetTransaction().Exec(func(tx *db.DB) error {
		nodeModel := core.GetReNewModel[*model.FlowNodeModel](tx, s.context)
		edgeModel := core.GetReNewModel[*model.FlowEdgeModel](tx, s.context)
		flowModel := core.GetReNewModel[*model.FlowModel](tx, s.context)

		if err := nodeModel.DeleteByFlowId(id); err != nil {
			return err
		}
		if err := edgeModel.DeleteByFlowId(id); err != nil {
			return err
		}
		return flowModel.Delete(id)
	})
}

func (s *FlowService) saveNodesAndEdges(tx *db.DB, flowId uint, nodes []*entity.FlowNode, edges []*entity.FlowEdge) error {
	if len(nodes) == 0 && len(edges) == 0 {
		return nil
	}
	nodeModel := core.GetReNewModel[*model.FlowNodeModel](tx, s.context)
	edgeModel := core.GetReNewModel[*model.FlowEdgeModel](tx, s.context)

	idMap := make(map[uint]uint)
	if len(nodes) > 0 {
		if err := nodeModel.DeleteByFlowId(flowId); err != nil {
			return err
		}
		oldIds := make([]uint, len(nodes))
		for i, n := range nodes {
			oldIds[i] = n.Id
			n.Id = 0
			n.FlowId = flowId
			if err := nodeModel.Create(n); err != nil {
				return err
			}
		}
		savedNodes, err := nodeModel.FindByFlowId(flowId)
		if err != nil {
			return err
		}
		for i, sn := range savedNodes {
			if i < len(oldIds) {
				idMap[oldIds[i]] = sn.Id
			}
		}
	}
	if len(edges) > 0 {
		if err := edgeModel.DeleteByFlowId(flowId); err != nil {
			return err
		}
		for _, e := range edges {
			e.Id = 0
			e.FlowId = flowId
			if nid, ok := idMap[e.SourceNodeId]; ok {
				e.SourceNodeId = nid
			}
			if nid, ok := idMap[e.TargetNodeId]; ok {
				e.TargetNodeId = nid
			}
			if err := edgeModel.Create(e); err != nil {
				return err
			}
		}
	}
	return nil
}
