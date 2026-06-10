// Package ocr provides optical character recognition provider interfaces.
package ocr

import (
	"context"

	"github.com/chuccp/go-ai-agent/ai/types"
	"github.com/chuccp/go-web-frame/config"
)

// Provider is the interface for OCR services.
type Provider interface {
	Name() string
	Init(ctx context.Context, cfg config.IConfig) error
	Recognize(image []byte, opts *RecognizeOptions) (*RecognizeResult, error)
	GetModels() []string
	GetProviderInfo() types.ProviderInfo
}

// RecognizeOptions holds OCR parameters.
type RecognizeOptions struct {
	Model    string `json:"model"`
	Language string `json:"language"`
}

// RecognizeResult holds the result of OCR.
type RecognizeResult struct {
	Text       string    `json:"text"`
	Confidence float64   `json:"confidence"`
	Blocks     []TextBlock `json:"blocks,omitempty"`
}

// TextBlock is a region of recognized text.
type TextBlock struct {
	Text   string  `json:"text"`
	X, Y   int     `json:"x"`
	Width  int     `json:"width"`
	Height int     `json:"height"`
	Conf   float64 `json:"confidence"`
}
