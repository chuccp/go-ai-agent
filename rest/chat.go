package rest

import (
	"fmt"

	aiTypes "github.com/chuccp/go-ai-agent/ai/types"
	"github.com/chuccp/go-ai-agent/config"
	"github.com/chuccp/go-ai-agent/entity"
	"github.com/chuccp/go-ai-agent/model"
	"github.com/chuccp/go-ai-agent/runner"
	wf "github.com/chuccp/go-web-frame"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/log"
	"github.com/chuccp/go-web-frame/web"
)

type ChatRest struct {
	context      *core.Context
	system       *config.System
	sessionModel *model.ChatSessionModel
	messageModel *model.ChatMessageModel
	aiModelModel *model.AIModelModel
	chatRunner   *runner.ChatRunner
	flowRunner   *runner.FlowRunner
}

func NewChatRest(r *runner.ChatRunner, fr *runner.FlowRunner) *ChatRest {
	return &ChatRest{chatRunner: r, flowRunner: fr}
}

func (c *ChatRest) Init(context *core.Context) error {
	c.context = context
	var err error
	c.system, err = wf.UnmarshalKeyConfig[*config.System](config.SYSTEM, context)
	if err != nil {
		return err
	}

	c.sessionModel = core.GetModel[*model.ChatSessionModel](context)
	c.messageModel = core.GetModel[*model.ChatMessageModel](context)
	c.aiModelModel = core.GetModel[*model.AIModelModel](context)

	c.context.Get("/api/sessions", c.listSessions)
	c.context.Post("/api/sessions", c.createSession)
	c.context.Delete("/api/sessions/:id", c.deleteSession)
	c.context.Get("/api/sessions/:id/messages", c.getSessionMessages)
	c.context.Get("/api/models", c.listModels)
	c.chatRunner.SetFlowRunner(c.flowRunner)
	c.context.WebSocket("/ws/chat", c.chatRunner.HandleWebSocket)

	log.Info("WebSocket 路由已注册: /ws/chat")
	return nil
}

func (c *ChatRest) listSessions(request *web.Request) (any, error) {
	sessions, err := c.sessionModel.List()
	if err != nil {
		return nil, err
	}
	return web.Data(sessions), nil
}

func (c *ChatRest) createSession(request *web.Request) (any, error) {
	jsonObj, err := request.Json()
	if err != nil {
		return nil, err
	}

	title := jsonObj.GetString("title")
	if title == "" {
		title = "New Chat"
	}

	session := &entity.ChatSession{Title: title}
	if err := c.sessionModel.Create(session); err != nil {
		return nil, err
	}

	return web.Data(session), nil
}

func (c *ChatRest) deleteSession(request *web.Request) (any, error) {
	id := request.ParamUint("id")
	if err := c.sessionModel.Delete(id); err != nil {
		return nil, err
	}
	return web.Ok("deleted"), nil
}

func (c *ChatRest) getSessionMessages(request *web.Request) (any, error) {
	sessionId := request.ParamUint("id")
	messages, err := c.messageModel.FindBySessionId(sessionId)
	if err != nil {
		return nil, err
	}
	return web.Data(messages), nil
}

func (c *ChatRest) listModels(request *web.Request) (any, error) {
	category := request.Query("category")
	if category == "" {
		category = aiTypes.CategoryLLM
	}

	dbModels, err := c.aiModelModel.ListByCategory(category)
	if err != nil {
		return nil, err
	}

	models := make([]map[string]any, 0, len(dbModels))
	for _, m := range dbModels {
		models = append(models, map[string]any{
			"id":        fmt.Sprintf("%d.%s", m.Id, m.Model),
			"name":      m.Name,
			"provider":  m.Provider,
			"model":     m.Model,
			"category":  m.Category,
			"is_default": m.IsDefault,
		})
	}
	return web.Data(map[string]interface{}{
		"models": models,
	}), nil
}
