package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/chuccp/go-ai-agent/chat"
)

// blockBuilder 在流式接收过程中累积构建一个 content block。
type blockBuilder struct {
	block   chat.ContentBlock
	rawJSON strings.Builder // tool_use 类型的 input_json_delta 累积
}

func (s *chatSession) run() {
	defer func() {
		s.mu.Lock()
		s.isRun = false
		s.mu.Unlock()
	}()

	for {
		messages := s.build()
		if messages == nil {
			return
		}

		provider := s.provider
		if provider == "" {
			provider = s.unifiedChatService.DefaultProvider()
		}

		resp, err := s.unifiedChatService.ChatWithStream(provider, messages)
		if err != nil {
			s.addEvent(&Event{Type: EventTypeError, Message: err.Error(), Done: true})
			return
		}

		var (
			blocks     []chat.ContentBlock
			current    *blockBuilder
			stopReason chat.StopReason
			textBuf    strings.Builder
			streamDone bool
		)

		flushBlock := func() {
			if current != nil {
				if current.block.Type == chat.ContentTypeToolUse && current.rawJSON.Len() > 0 {
					var input any
					if err := json.Unmarshal([]byte(current.rawJSON.String()), &input); err != nil {
						log.Printf("tool_use JSON 解析失败: %v, raw=%s", err, current.rawJSON.String())
					}
					current.block.Input = input
				}
				blocks = append(blocks, current.block)
				current = nil
			}
		}

		for evt := resp.ReadEvent(); evt != nil; evt = resp.ReadEvent() {
			switch e := evt.(type) {
			case *chat.ContentBlockStartEvent:
				flushBlock()
				current = &blockBuilder{
					block: chat.ContentBlock{
						Type: e.ContentBlock.Type,
						ID:   e.ContentBlock.ID,
						Name: e.ContentBlock.Name,
					},
				}

			case *chat.ContentBlockDeltaEvent:
				if current == nil {
					continue
				}
				switch e.Delta.Type {
				case "text_delta":
					current.block.Text += e.Delta.Text
					textBuf.WriteString(e.Delta.Text)
					s.addEvent(&Event{
						Type:           EventTypeChunk,
						Content:        e.Delta.Text,
						ConversationID: s.id,
					})
				case "input_json_delta":
					current.rawJSON.WriteString(e.Delta.PartialJSON)
				}

			case *chat.ContentBlockStopEvent:
				flushBlock()

			case *chat.MessageDeltaEvent:
				stopReason = e.StopReason

			case *chat.ErrorEvent:
				s.addEvent(&Event{Type: EventTypeError, Message: e.Error(), Done: true})
				return

			case *chat.MessageStopEvent:
				streamDone = true
			}
		}

		if !streamDone {
			return
		}

		switch stopReason {
		case chat.StopReasonToolUse:
			s.history = append(s.history, chat.Message{
				Role:    chat.RoleAssistant,
				Content: blocks,
			})

			toolResults := make([]chat.ContentBlock, 0, len(blocks))
			for _, block := range blocks {
				if block.Type != chat.ContentTypeToolUse {
					continue
				}
				exec, ok := s.toolExecutors[block.Name]
				if !ok {
					toolResults = append(toolResults, chat.ContentBlock{
						Type:      chat.ContentTypeToolResult,
						ToolUseID: block.ID,
						Content:   []chat.ContentBlock{{Type: chat.ContentTypeText, Text: fmt.Sprintf("未知工具: %s", block.Name)}},
					})
					continue
				}

				args, _ := block.Input.(map[string]any)
				output, execErr := exec.Execute(args)

				s.addEvent(&Event{
					Type:           EventTypeChunk,
					Content:        fmt.Sprintf("\n🔧 执行命令: %v\n%s\n", args, output),
					ConversationID: s.id,
				})

				resultText := output
				if execErr != nil {
					resultText = fmt.Sprintf("错误: %v", execErr)
				}
				toolResults = append(toolResults, chat.ContentBlock{
					Type:      chat.ContentTypeToolResult,
					ToolUseID: block.ID,
					Content:   []chat.ContentBlock{{Type: chat.ContentTypeText, Text: resultText}},
				})
			}

			s.history = append(s.history, chat.Message{
				Role:    chat.RoleUser,
				Content: toolResults,
			})

			continue

		default:
			assistantBlocks := make([]chat.ContentBlock, 0, len(blocks))
			hasText := false
			for _, block := range blocks {
				if block.Type == chat.ContentTypeText && block.Text != "" {
					assistantBlocks = append(assistantBlocks, block)
					hasText = true
				}
			}
			if !hasText && textBuf.Len() > 0 {
				assistantBlocks = append(assistantBlocks, chat.ContentBlock{Type: chat.ContentTypeText, Text: textBuf.String()})
			}
			s.history = append(s.history, chat.Message{
				Role:    chat.RoleAssistant,
				Content: assistantBlocks,
			})
			s.addEvent(&Event{Type: EventTypeDone, Done: true, ConversationID: s.id})
		}
	}
}
