package runner

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/chuccp/go-ai-agent/agent"
	"github.com/chuccp/go-ai-agent/ai/chat/common"
	aiTypes "github.com/chuccp/go-ai-agent/ai/types"
	"github.com/chuccp/go-ai-agent/entity"
	"github.com/chuccp/go-web-frame/log"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// ==================== 共享的会话准备逻辑 ====================

type chatPrep struct {
	modelPath    string
	history      []common.ChatMessage
	userMessage  string
	sessionID    uint
	opts         *common.LLMOptions
	contentParts []common.ContentPart // 多模态内容（图片等）
}

func (r *ChatRunner) prepareChat(conn *websocket.Conn, req WSRequest) chatPrep {
	cp := chatPrep{}

	cp.modelPath = req.Model
	if cp.modelPath == "" {
		cp.modelPath = r.defaultModelPath
	}

	cp.history = req.Messages
	if len(cp.history) > 0 {
		cp.userMessage = cp.history[len(cp.history)-1].Content
		cp.history = cp.history[:len(cp.history)-1]
	}

	// Process file attachments early to get extra text and content parts
	if len(req.Attachments) > 0 {
		var extraText string
		var err error
		cp.contentParts, extraText, err = r.processAttachments(req.Attachments, cp.modelPath)
		if err != nil {
			r.sendJSON(conn, WSResponse{Type: "error", Message: err.Error()})
			return cp
		}
		if extraText != "" {
			cp.userMessage += "\n\n" + extraText
		}
		cp.userMessage = strings.TrimSpace(cp.userMessage)
	}

	cp.sessionID = req.SessionID
	if cp.sessionID > 0 {
		msgs, err := r.messageModel.FindBySessionId(cp.sessionID)
		if err == nil && len(msgs) > 0 {
			cp.history = make([]common.ChatMessage, 0, len(msgs))
			for _, m := range msgs {
				cp.history = append(cp.history, common.ChatMessage{Role: m.Role, Content: m.Content})
			}
		}
	} else {
		session := &entity.ChatSession{Title: r.generateTitle(cp.userMessage)}
		if err := r.sessionModel.Create(session); err == nil {
			cp.sessionID = session.Id
			r.sendJSON(conn, WSResponse{Type: "session_created", SessionID: cp.sessionID})
		}
	}

	if cp.sessionID > 0 {
		r.messageModel.Create(&entity.ChatMessage{
			SessionId: cp.sessionID,
			Role:      "user",
			Content:   cp.userMessage,
		})
	}

	cp.opts = common.NewLLMOptions()
	if req.Options != nil {
		for k, v := range req.Options {
			cp.opts.Set(k, v)
		}
	}
	return cp
}

// ==================== chat / agent 处理 ====================

func (r *ChatRunner) handleChat(conn *websocket.Conn, req WSRequest) {
	cp := r.prepareChat(conn, req)

	// Build the user message with any content parts (multi-modal)
	userMsg := common.ChatMessage{
		Role:         "user",
		Content:      cp.userMessage,
		ContentParts: cp.contentParts,
	}
	history := append(cp.history, userMsg)

	if req.Stream {
		r.handleStreamChatFull(conn, cp.modelPath, history, cp.opts, cp.sessionID)
	} else {
		r.handleNonStreamChatFull(conn, cp.modelPath, history, cp.opts, cp.sessionID)
	}
}

func (r *ChatRunner) handleAgent(conn *websocket.Conn, req WSRequest) {
	cp := r.prepareChat(conn, req)

	sender := &wsSender{conn: conn, runner: r}
	chatID := fmt.Sprintf("%d", cp.sessionID)
	c := agent.NewChat(context.Background(), chatID, cp.modelPath, r.chatService, cp.opts, sender)

	startIter := len(cp.history) / 2
	c.SetIteration(startIter)

	for _, m := range cp.history {
		c.AddUserMessage(m.Content)
	}
	c.AddUserMessage(cp.userMessage)

	var assistantContent strings.Builder
	sender.onChunk = func(content string, reasoning bool) {
		if !reasoning {
			assistantContent.WriteString(content)
		}
	}
	sender.onDone = func() {
		if cp.sessionID > 0 && assistantContent.Len() > 0 {
			r.messageModel.Create(&entity.ChatMessage{
				SessionId: cp.sessionID,
				Role:      "assistant",
				Content:   assistantContent.String(),
			})
		}
	}
	c.Process()
}

func (r *ChatRunner) handleStreamChat(conn *websocket.Conn, path string, history []common.ChatMessage, text string, opts *common.LLMOptions, sessionID uint) {
	handler := common.NewStreamHandler()
	var mu sync.Mutex
	var assistantContent strings.Builder

	handler.OnContent(func(content string, reasoning bool) {
		mu.Lock()
		defer mu.Unlock()
		if !reasoning {
			assistantContent.WriteString(content)
		}
		r.sendJSON(conn, WSResponse{Type: "chunk", Content: content, Done: false})
	})

	handler.OnComplete(func(content string, reasoningStr string) {
		mu.Lock()
		defer mu.Unlock()
		r.sendJSON(conn, WSResponse{Type: "chunk", Content: "", Done: true})
		if sessionID > 0 && assistantContent.Len() > 0 {
			r.messageModel.Create(&entity.ChatMessage{SessionId: sessionID, Role: "assistant", Content: assistantContent.String()})
		}
	})

	handler.OnError(func(err error) {
		r.sendJSON(conn, WSResponse{Type: "error", Message: err.Error(), Done: true})
	})

	if err := r.chatService.ChatStreamWithContext(context.Background(), path, history, text, handler, opts); err != nil {
		r.sendJSON(conn, WSResponse{Type: "error", Message: err.Error(), Done: true})
	}
}

// handleStreamChatFull is like handleStreamChat but takes complete messages array
// including the user message (possibly with ContentParts for multi-modal).
func (r *ChatRunner) handleStreamChatFull(conn *websocket.Conn, path string, messages []common.ChatMessage, opts *common.LLMOptions, sessionID uint) {
	handler := common.NewStreamHandler()
	var mu sync.Mutex
	var assistantContent strings.Builder

	handler.OnContent(func(content string, reasoning bool) {
		mu.Lock()
		defer mu.Unlock()
		if !reasoning {
			assistantContent.WriteString(content)
		}
		r.sendJSON(conn, WSResponse{Type: "chunk", Content: content, Done: false})
	})

	handler.OnComplete(func(content string, reasoningStr string) {
		mu.Lock()
		defer mu.Unlock()
		r.sendJSON(conn, WSResponse{Type: "chunk", Content: "", Done: true})
		if sessionID > 0 && assistantContent.Len() > 0 {
			r.messageModel.Create(&entity.ChatMessage{SessionId: sessionID, Role: "assistant", Content: assistantContent.String()})
		}
	})

	handler.OnError(func(err error) {
		r.sendJSON(conn, WSResponse{Type: "error", Message: err.Error(), Done: true})
	})

	// Pass full messages as history (last is user message with ContentParts), empty text
	if err := r.chatService.ChatStreamWithContext(context.Background(), path, messages, "", handler, opts); err != nil {
		r.sendJSON(conn, WSResponse{Type: "error", Message: err.Error(), Done: true})
	}
}

// handleNonStreamChatFull is like handleNonStreamChat but takes complete messages array.
func (r *ChatRunner) handleNonStreamChatFull(conn *websocket.Conn, path string, messages []common.ChatMessage, opts *common.LLMOptions, sessionID uint) {
	content, err := r.chatService.ChatWithHistoryWithContext(context.Background(), path, messages, "", opts)
	if err != nil {
		r.sendJSON(conn, WSResponse{Type: "error", Message: err.Error(), Done: true})
		return
	}
	r.sendJSON(conn, WSResponse{Type: "chunk", Content: content, Done: true})
	if sessionID > 0 && content != "" {
		r.messageModel.Create(&entity.ChatMessage{SessionId: sessionID, Role: "assistant", Content: content})
	}
}

func (r *ChatRunner) handleNonStreamChat(conn *websocket.Conn, path string, history []common.ChatMessage, text string, opts *common.LLMOptions, sessionID uint) {
	var content string
	var err error
	if len(history) > 0 {
		content, err = r.chatService.ChatWithHistoryWithContext(context.Background(), path, history, text, opts)
	} else {
		content, err = r.chatService.ChatWithContext(context.Background(), path, text, opts)
	}
	if err != nil {
		r.sendJSON(conn, WSResponse{Type: "error", Message: err.Error(), Done: true})
		return
	}
	r.sendJSON(conn, WSResponse{Type: "chunk", Content: content, Done: true})
	if sessionID > 0 && content != "" {
		r.messageModel.Create(&entity.ChatMessage{SessionId: sessionID, Role: "assistant", Content: content})
	}
}

// ==================== wsSender ====================

type wsSender struct {
	conn    *websocket.Conn
	runner  *ChatRunner
	onChunk func(content string, reasoning bool)
	onDone  func()
}

func (s *wsSender) Send(event agent.Event) {
	resp := WSResponse{
		Type:           event.Type,
		Content:        event.Content,
		Reasoning:      event.Reasoning,
		Message:        event.Message,
		Done:           event.Done,
		Iteration:      event.Iteration,
		ConversationID: event.ConversationID,
	}
	switch event.Type {
	case "chunk":
		if !event.Done && s.onChunk != nil {
			s.onChunk(event.Content, event.Reasoning != "")
		}
		if event.Done && s.onDone != nil {
			s.onDone()
		}
	case "tool_call":
		resp.Message = fmt.Sprintf("🔧 %s(%s)", event.ToolName, event.ToolInput)
	case "tool_result":
		resp.Message = fmt.Sprintf("📋 %s", event.Message)
	}
	s.runner.sendJSON(s.conn, resp)
}

// ==================== 工具方法 ====================

func (r *ChatRunner) sendJSON(conn *websocket.Conn, resp WSResponse) {
	data, err := sonic.Marshal(resp)
	if err != nil {
		log.Error("序列化 WS 响应失败", zap.Error(err))
		return
	}
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Error("发送 WS 消息失败", zap.Error(err))
	}
}

func (r *ChatRunner) generateTitle(text string) string {
	if len(text) > 30 {
		return text[:30] + "..."
	}
	return text
}

// parseModelDBID 从 "1.default" 格式解析出数据库 ID
func parseModelDBID(path string) uint {
	parts := strings.SplitN(path, ".", 2)
	if len(parts) == 0 {
		return 0
	}
	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0
	}
	return uint(id)
}

// getModelCapabilities 检查当前模型的多模态能力和 OCR 可用性
func (r *ChatRunner) getModelCapabilities(modelPath string) (multimodal bool, hasOCR bool) {
	dbID := parseModelDBID(modelPath)
	if dbID > 0 {
		aiModel := r.aiModel()
		if aiModel != nil {
			if m, err := aiModel.FindById(dbID); err == nil && m != nil {
				multimodal = m.SupportsMultimodal
			}
			ocrs, err := aiModel.ListByCategory("ocr")
			hasOCR = err == nil && len(ocrs) > 0
		}
	}
	return
}

// processAttachments 处理上传的附件，返回 ContentParts 和增强后的文本
func (r *ChatRunner) processAttachments(attachments []Attachment, modelPath string) ([]common.ContentPart, string, error) {
	multimodal, hasOCR := r.getModelCapabilities(modelPath)
	var parts []common.ContentPart
	var extraText strings.Builder
	uploadDir := "./data/uploads"

	for _, att := range attachments {
		filePath := uploadDir + "/" + att.Path
		isImage := strings.HasPrefix(att.Type, "image/")

		switch {
		case isImage && multimodal:
			data, err := os.ReadFile(filePath)
			if err != nil {
				return nil, "", fmt.Errorf("读取图片失败: %w", err)
			}
			mediaType := att.Type
			if mediaType == "" {
				mediaType = "image/png"
			}
			parts = append(parts, common.ContentPart{
				Type:     "image",
				ImageURL: "data:" + mediaType + ";base64," + base64.StdEncoding.EncodeToString(data),
			})

		case isImage && hasOCR:
			text, err := r.ocrImage(filePath)
			if err != nil {
				log.Warn("OCR failed, using placeholder", zap.Error(err))
				extraText.WriteString("[图片: " + att.Name + " - OCR 识别失败]\n")
			} else {
				extraText.WriteString("[图片 OCR 识别结果: " + att.Name + "]\n" + text + "\n")
			}

		case isImage:
			return nil, "", fmt.Errorf("当前模型不支持图片处理。请配置多模态模型或 OCR 模型。")

		case strings.HasPrefix(att.Type, "text/"):
			data, err := os.ReadFile(filePath)
			if err != nil {
				return nil, "", fmt.Errorf("读取文本文件失败: %w", err)
			}
			extraText.WriteString("[文件内容: " + att.Name + "]\n" + string(data) + "\n")

		default:
			// Documents (PDF, DOCX, etc.) — marked for tool processing
			extraText.WriteString("[上传文件: " + att.Name + " (" + att.Type + ") - 可以使用 read_document 工具读取]\n")
		}
	}

	return parts, extraText.String(), nil
}

// ocrImage uses a multimodal LLM model to recognize text in an image.
func (r *ChatRunner) ocrImage(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("读取图片失败: %w", err)
	}

	// Find a multimodal model to do OCR
	aiModel := r.aiModel()
	if aiModel == nil {
		return "", fmt.Errorf("AI model not initialized")
	}
	list, err := aiModel.List()
	if err != nil {
		return "", err
	}
	var ocrModelPath string
	for _, m := range list {
		if m.SupportsMultimodal && m.Category == aiTypes.CategoryLLM {
			ocrModelPath = strconv.FormatUint(uint64(m.Id), 10) + ".default"
			break
		}
	}
	if ocrModelPath == "" {
		return "", fmt.Errorf("未找到支持多模态的模型用于 OCR")
	}

	imageURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
	msg := common.ChatMessage{
		Role: "user",
		Content: "请识别并提取图片中的所有文字内容。只返回识别出的文字，不要添加任何解释或额外内容。",
		ContentParts: []common.ContentPart{
			{Type: "image", ImageURL: imageURL},
		},
	}
	messages := []common.ChatMessage{msg}

	result, err := r.chatService.ChatWithHistoryWithContext(context.Background(), ocrModelPath, messages, "", nil)
	if err != nil {
		return "", fmt.Errorf("OCR 识别失败: %w", err)
	}
	return result, nil
}
