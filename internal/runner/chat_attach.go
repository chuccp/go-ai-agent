package runner

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"

	aiTypes "github.com/chuccp/go-ai-agent/internal/ai/types"
	"github.com/chuccp/go-ai-agent/internal/ai/chat/common"
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
				return nil, "", fmt.Errorf("failed to read image: %w", err)
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
				extraText.WriteString("[Image: " + att.Name + " - OCR recognition failed]\n")
			} else {
				extraText.WriteString("[Image OCR result: " + att.Name + "]\n" + text + "\n")
			}

		case isImage:
			return nil, "", fmt.Errorf("Current model does not support image processing. Please configure a multimodal model or OCR model.")

		case strings.HasPrefix(att.Type, "text/"):
			data, err := os.ReadFile(filePath)
			if err != nil {
				return nil, "", fmt.Errorf("failed to read text file: %w", err)
			}
			extraText.WriteString("[File content: " + att.Name + "]\n" + string(data) + "\n")

		default:
			extraText.WriteString("[Uploaded file: " + att.Name + " (" + att.Type + ") - use read_document tool to read]\n")
		}
	}

	return parts, extraText.String(), nil
}

// ocrImage uses a multimodal LLM model to recognize text in an image.
func (r *ChatRunner) ocrImage(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read image: %w", err)
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
		return "", fmt.Errorf("No multimodal model found for OCR")
	}

	imageURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
	msg := common.ChatMessage{
		Role: "user",
		Content: "Please recognize and extract all text from the image. Return only the recognized text, no explanations or additional content.",
		ContentParts: []common.ContentPart{
			{Type: "image", ImageURL: imageURL},
		},
	}
	messages := []common.ChatMessage{msg}

	result, err := r.chatService.ChatWithHistoryWithContext(context.Background(), ocrModelPath, messages, "", nil)
	if err != nil {
		return "", fmt.Errorf("OCR recognition failed: %w", err)
	}
	return result, nil
}
