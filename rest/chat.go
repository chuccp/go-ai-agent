package rest

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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
}

func NewChatRest() *ChatRest { return &ChatRest{} }

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

		chatRunner := core.GetRunner[*runner.ChatRunner](context)

	c.context.Get("/api/sessions", c.listSessions)
	c.context.Post("/api/sessions", c.createSession)
	c.context.Delete("/api/sessions/:id", c.deleteSession)
	c.context.Get("/api/sessions/:id/messages", c.getSessionMessages)
	c.context.Get("/api/models", c.listModels)
	c.context.WebSocket("/ws/chat", chatRunner.HandleWebSocket)

	// File upload
	c.context.Post("/api/upload", c.uploadFile)

	// Ensure upload directory exists
	uploadDir := "./data/uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Warn("Failed to create upload directory")
	}

	log.Info("WebSocket route registered: /ws/chat")
	return nil
}

func (c *ChatRest) listSessions(request *web.Request) (any, error) {
	sessions, err := c.sessionModel.WithContext(request.Ctx()).List()
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
	if err := c.sessionModel.WithContext(request.Ctx()).Create(session); err != nil {
		return nil, err
	}

	return web.Data(session), nil
}

func (c *ChatRest) deleteSession(request *web.Request) (any, error) {
	id := request.ParamUint("id")
	if err := c.sessionModel.WithContext(request.Ctx()).Delete(id); err != nil {
		return nil, err
	}
	return web.Ok("deleted"), nil
}

func (c *ChatRest) getSessionMessages(request *web.Request) (any, error) {
	sessionId := request.ParamUint("id")
	messages, err := c.messageModel.WithContext(request.Ctx()).FindBySessionId(sessionId)
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

	dbModels, err := c.aiModelModel.WithContext(request.Ctx()).ListByCategory(category)
	if err != nil {
		return nil, err
	}

	models := make([]map[string]any, 0, len(dbModels))
	for _, m := range dbModels {
		models = append(models, map[string]any{
			"id":                  fmt.Sprintf("%d.%s", m.Id, m.Model),
			"name":                m.Name,
			"provider":            m.Provider,
			"model":               m.Model,
			"category":            m.Category,
			"is_default":          m.IsDefault,
			"supports_multimodal": m.SupportsMultimodal,
		})
	}
	return web.Data(map[string]interface{}{
		"models": models,
	}), nil
}

// uploadFile handles file uploads for chat attachments.
func (c *ChatRest) uploadFile(req *web.Request) (any, error) {
	form, err := req.MultipartForm()
	if err != nil {
		return nil, fmt.Errorf("failed to parse upload form: %w", err)
	}

	files := form.File["file"]
	if len(files) == 0 {
		return nil, fmt.Errorf("upload file not found (field name: file)")
	}

	header := files[0]
	file, err := header.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Generate unique filename
	id := make([]byte, 8)
	if _, err := rand.Read(id); err != nil {
		return nil, fmt.Errorf("failed to generate random filename: %w", err)
	}
	ext := filepath.Ext(header.Filename)
	safeName := hex.EncodeToString(id) + "_" + strings.ReplaceAll(header.Filename, " ", "_")
	uploadDir := "./data/uploads"
	savePath := filepath.Join(uploadDir, safeName)

	dst, err := os.Create(savePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = dst.Close() }()

	if _, err := io.Copy(dst, file); err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// Detect MIME type from extension if not provided
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = detectMimeType(ext)
	}

	return web.Data(map[string]any{
		"id":   id,
		"name": header.Filename,
		"type": mimeType,
		"size": header.Size,
		"path": safeName,
	}), nil
}

func detectMimeType(ext string) string {
	switch strings.ToLower(ext) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".pdf":
		return "application/pdf"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".doc":
		return "application/msword"
	case ".txt", ".md":
		return "text/plain"
	case ".csv":
		return "text/csv"
	default:
		return "application/octet-stream"
	}
}
