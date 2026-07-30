package service

import (
	"encoding/json"
	"strconv"

	"github.com/chuccp/go-agent-sdk/chat"
	"github.com/chuccp/go-ai-agent/cmd/server/entity"
	"github.com/chuccp/go-ai-agent/cmd/server/model"
	"github.com/chuccp/go-web-frame/core"
)

// HistoryStoreImpl 实现 agent.HistoryStore 接口，使用数据库持久化聊天记录。
type HistoryStoreImpl struct {
	messageModel *model.ChatMessageModel
}

func (h *HistoryStoreImpl) Init(ctx *core.Context) error {
	h.messageModel = core.GetModel[*model.ChatMessageModel](ctx)
	return nil
}

// LoadHistory 从数据库加载指定会话的历史消息，还原为 SDK 的 chat.Message 格式。
func (h *HistoryStoreImpl) LoadHistory(sessionID string) ([]chat.Message, error) {
	id, err := strconv.ParseUint(sessionID, 10, 64)
	if err != nil {
		return nil, nil // 非法 sessionID，视为新会话
	}

	rows, err := h.messageModel.Query().
		Where("session_id = ?", uint(id)).
		Order("created_at asc").
		All()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}

	messages := make([]chat.Message, 0, len(rows))
	for _, row := range rows {
		msg := chat.Message{
			Role: chat.Role(row.Role),
		}
		// 反序列化 content blocks
		if row.Content != "" {
			var blocks []chat.ContentBlock
			if err := json.Unmarshal([]byte(row.Content), &blocks); err == nil {
				msg.Content = blocks
			} else {
				// 兼容旧数据：纯文本存储
				msg.Content = []chat.ContentBlock{{Type: chat.ContentTypeText, Text: row.Content}}
			}
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

// SaveHistory 将完整历史全量覆盖写入数据库。
// 策略：先删除该会话的旧消息，再批量插入新消息。
func (h *HistoryStoreImpl) SaveHistory(sessionID string, messages []chat.Message) error {
	id, err := strconv.ParseUint(sessionID, 10, 64)
	if err != nil {
		return nil
	}
	sid := uint(id)

	// 删除旧记录
	if err := h.messageModel.Delete().Where("session_id = ?", sid).Delete(); err != nil {
		return err
	}

	// 批量插入新记录
	for _, msg := range messages {
		contentJSON, _ := json.Marshal(msg.Content)
		row := &entity.ChatMessage{
			SessionId: sid,
			Role:      string(msg.Role),
			Content:   string(contentJSON),
		}
		if err := h.messageModel.Save(row); err != nil {
			return err
		}
	}
	return nil
}
