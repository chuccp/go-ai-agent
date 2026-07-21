// Package video provides video generation provider interfaces.
package video

import (
	"context"

	"github.com/chuccp/go-ai-agent/internal2/ai/types"
	"github.com/chuccp/go-web-frame/config"
)

// Provider is the interface for video generation services.
type Provider interface {
	Name() string
	Init(ctx context.Context, cfg config.IConfig) error
	Generate(prompt string, opts *GenerateOptions) (*GenerateResult, error)
	GetModels() []string
	GetProviderInfo() types.ProviderInfo
}

// GenerateOptions holds video generation parameters.
type GenerateOptions struct {
	Model      string `json:"model"`
	Duration   int    `json:"duration"`   // seconds
	Resolution string `json:"resolution"` // e.g. "1080p"
	FPS        int    `json:"fps"`
	Style      string `json:"style"`
}

// GenerateResult holds the result of video generation.
type GenerateResult struct {
	URL  string `json:"url"`
	Size int64  `json:"size"`
}
