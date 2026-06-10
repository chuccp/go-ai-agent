// Package speech provides text-to-speech (TTS) provider interfaces.
package speech

import (
	"context"

	"github.com/chuccp/go-ai-agent/ai/types"
	"github.com/chuccp/go-web-frame/config"
)

// Provider is the interface for text-to-speech services.
type Provider interface {
	Name() string
	Init(ctx context.Context, cfg config.IConfig) error
	Synthesize(text string, opts *SynthesizeOptions) (*SynthesizeResult, error)
	GetModels() []string
	GetVoices() []Voice
	GetProviderInfo() types.ProviderInfo
}

// SynthesizeOptions holds TTS parameters.
type SynthesizeOptions struct {
	Model    string  `json:"model"`
	Voice    string  `json:"voice"`
	Speed    float64 `json:"speed"`
	Format   string  `json:"format"` // mp3, wav, ogg
	Language string  `json:"language"`
}

// SynthesizeResult holds the result of speech synthesis.
type SynthesizeResult struct {
	AudioData []byte `json:"audio_data"`
	Format    string `json:"format"`
	Duration  int    `json:"duration"` // milliseconds
}

// Voice describes an available TTS voice.
type Voice struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Language string `json:"language"`
	Gender   string `json:"gender"`
}
