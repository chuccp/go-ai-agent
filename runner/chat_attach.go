package runner

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"

	aiTypes "github.com/chuccp/go-ai-agent/ai/types"
	"github.com/chuccp/go-ai-agent/ai/chat/common"
	"github.com/chuccp/go-web-frame/log"
	"go.uber.org/zap"
)

// getModelCapabilities checks if the current model supports multimodal and OCR.
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

// processAttachments handles uploaded attachments, returning ContentParts and enriched text.
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
