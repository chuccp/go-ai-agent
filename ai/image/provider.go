// Package image provides image generation provider interfaces.
package image

import (
	"context"

	"github.com/chuccp/go-ai-agent/ai/types"
	"github.com/chuccp/go-web-frame/config"
)

// Provider is the interface for image generation services.
type Provider interface {
	Name() string
	Init(ctx context.Context, cfg config.IConfig) error
	Generate(prompt string, opts *GenerateOptions) (*GenerateResult, error)
	GetModels() []string
	GetProviderInfo() types.ProviderInfo
}

// GenerateOptions holds image generation parameters.
type GenerateOptions struct {
	Model     string `json:"model"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	NumImages int    `json:"num_images"`
	Quality   string `json:"quality"`
	Style     string `json:"style"`
}

// GenerateResult holds the result of image generation.
type GenerateResult struct {
	URLs       []string `json:"urls"`
	Base64Data []string `json:"base64_data"`
	Count      int      `json:"count"`
}
