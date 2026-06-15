package model

import (
	"context"
	"testing"

	"github.com/chuccp/go-ai-agent/internal/entity"
	fwModel "github.com/chuccp/go-web-frame/model"
)

func TestFlowModel_WithContext(t *testing.T) {
	em := fwModel.NewEntryModel[*entity.FlowDefinition, uint](nil, "test")
	m := &FlowModel{EntryModel: em}

	ctx := context.Background()
	m2 := m.WithContext(ctx)

	if m2 == m {
		t.Error("WithContext should return a new instance")
	}
	if m2.EntryModel == nil {
		t.Fatal("WithContext returned nil EntryModel")
	}
	if m2.EntryModel == m.EntryModel {
		t.Error("EntryModel should be a new instance after WithContext")
	}
}

func TestFlowNodeModel_WithContext(t *testing.T) {
	em := fwModel.NewEntryModel[*entity.FlowNode, uint](nil, "test")
	m := &FlowNodeModel{EntryModel: em}
	m2 := m.WithContext(context.Background())
	if m2.EntryModel == m.EntryModel {
		t.Error("EntryModel should be a new instance after WithContext")
	}
}

func TestFlowEdgeModel_WithContext(t *testing.T) {
	em := fwModel.NewEntryModel[*entity.FlowEdge, uint](nil, "test")
	m := &FlowEdgeModel{EntryModel: em}
	m2 := m.WithContext(context.Background())
	if m2.EntryModel == m.EntryModel {
		t.Error("EntryModel should be a new instance after WithContext")
	}
}

func TestFlowExecutionModel_WithContext(t *testing.T) {
	em := fwModel.NewEntryModel[*entity.FlowExecution, uint](nil, "test")
	m := &FlowExecutionModel{EntryModel: em}
	m2 := m.WithContext(context.Background())
	if m2.EntryModel == m.EntryModel {
		t.Error("EntryModel should be a new instance after WithContext")
	}
}

func TestAIModelModel_WithContext(t *testing.T) {
	em := fwModel.NewEntryModel[*entity.AIModel, uint](nil, "test")
	m := &AIModelModel{EntryModel: em}
	m2 := m.WithContext(context.Background())
	if m2.EntryModel == m.EntryModel {
		t.Error("EntryModel should be a new instance after WithContext")
	}
}

func TestAIModelModel_WithContext_NilEntryModel(t *testing.T) {
	m := &AIModelModel{}
	m2 := m.WithContext(context.Background())
	if m2.EntryModel != nil {
		t.Error("EntryModel should remain nil when original is nil")
	}
}

func TestChatSessionModel_WithContext(t *testing.T) {
	em := fwModel.NewEntryModel[*entity.ChatSession, uint](nil, "test")
	msgEm := fwModel.NewEntryModel[*entity.ChatMessage, uint](nil, "messages")
	msgModel := &ChatMessageModel{EntryModel: msgEm}
	m := &ChatSessionModel{EntryModel: em, messageModel: msgModel}

	m2 := m.WithContext(context.Background())
	if m2.messageModel == nil {
		t.Fatal("WithContext lost messageModel reference")
	}
	if m2.messageModel == m.messageModel {
		t.Error("messageModel should be a new instance after WithContext")
	}
}

func TestChatSessionModel_WithContext_NilEntryModel(t *testing.T) {
	msgEm := fwModel.NewEntryModel[*entity.ChatMessage, uint](nil, "messages")
	msgModel := &ChatMessageModel{EntryModel: msgEm}
	m := &ChatSessionModel{messageModel: msgModel}
	m2 := m.WithContext(context.Background())
	if m2.messageModel == nil {
		t.Fatal("WithContext should preserve messageModel even when EntryModel is nil")
	}
}

func TestChatMessageModel_WithContext(t *testing.T) {
	em := fwModel.NewEntryModel[*entity.ChatMessage, uint](nil, "test")
	m := &ChatMessageModel{EntryModel: em}
	m2 := m.WithContext(context.Background())
	if m2.EntryModel == m.EntryModel {
		t.Error("EntryModel should be a new instance after WithContext")
	}
}

func TestAdminUserModel_WithContext(t *testing.T) {
	em := fwModel.NewEntryModel[*entity.AdminUser, uint](nil, "test")
	m := &AdminUserModel{EntryModel: em}
	m2 := m.WithContext(context.Background())
	if m2.EntryModel == m.EntryModel {
		t.Error("EntryModel should be a new instance after WithContext")
	}
}

func TestAdminUserModel_WithContext_NilEntryModel(t *testing.T) {
	m := &AdminUserModel{}
	m2 := m.WithContext(context.Background())
	if m2.EntryModel != nil {
		t.Error("EntryModel should remain nil when original is nil")
	}
}
