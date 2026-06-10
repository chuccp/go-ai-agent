// Package embedding provides text embedding / vector model provider interfaces.
package embedding

import (
	"context"

	"github.com/chuccp/go-ai-agent/ai/types"
	"github.com/chuccp/go-web-frame/config"
)

// Provider is the interface for embedding / vector services.
type Provider interface {
	Name() string
	Init(ctx context.Context, cfg config.IConfig) error
	Embed(texts []string, opts *EmbedOptions) (*EmbedResult, error)
	GetModels() []string
	GetProviderInfo() types.ProviderInfo
}

// EmbedOptions holds embedding parameters.
type EmbedOptions struct {
	Model string `json:"model"`
}

// EmbedResult holds the result of text embedding.
type EmbedResult struct {
	Vectors  [][]float64 `json:"vectors"`
	Dim      int         `json:"dim"`
	Tokens   int         `json:"tokens"`
}
