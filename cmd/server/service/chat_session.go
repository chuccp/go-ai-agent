package service

import (
	"context"

	"github.com/chuccp/go-ai-agent/cmd/server/entity"
	"github.com/chuccp/go-ai-agent/cmd/server/model"
	"github.com/chuccp/go-web-frame/core"
)

// ChatSessionService provides business logic for chat session and message CRUD.
// It wraps the model layer and is consumed by REST handlers via core.GetService.
type ChatSessionService struct {
	context      *core.Context
	sessionModel *model.ChatSessionModel
	messageModel *model.ChatMessageModel
}

// Init implements core.IService. It resolves models from the core context.
func (s *ChatSessionService) Init(ctx *core.Context) error {
	s.context = ctx
	s.sessionModel = core.GetModel[*model.ChatSessionModel](ctx)
	s.messageModel = core.GetModel[*model.ChatMessageModel](ctx)
	return nil
}

// ListSessions returns all chat sessions ordered by most recently updated.
func (s *ChatSessionService) ListSessions() ([]*entity.ChatSession, error) {
	return s.sessionModel.Query().
		Order("updated_at desc").
		All()
}

// CreateSession creates a new chat session with the given title.
func (s *ChatSessionService) CreateSession(ctx context.Context, title string) (*entity.ChatSession, error) {
	session := &entity.ChatSession{Title: title}
	if err := s.sessionModel.WithContext(ctx).Save(session); err != nil {
		return nil, err
	}
	return session, nil
}

// DeleteSession deletes a session and all its messages.
func (s *ChatSessionService) DeleteSession(ctx context.Context, id uint) error {
	// Delete all messages in this session first
	if err := s.messageModel.WithContext(ctx).
		Delete().Where("session_id = ?", id).Delete(); err != nil {
		return err
	}
	// Delete the session itself
	return s.sessionModel.WithContext(ctx).DeleteByPK(id)
}

// GetSessionMessages returns all messages for a session ordered by creation time.
func (s *ChatSessionService) GetSessionMessages(sessionId uint) ([]*entity.ChatMessage, error) {
	return s.messageModel.Query().
		Where("session_id = ?", sessionId).
		Order("created_at asc").
		All()
}
