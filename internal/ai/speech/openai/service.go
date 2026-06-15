package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/chuccp/go-ai-agent/internal/ai/speech"
	"github.com/chuccp/go-ai-agent/internal/ai/types"
	"github.com/chuccp/go-web-frame/config"
	"github.com/chuccp/go-web-frame/log"
	"go.uber.org/zap"
)

// ProviderDefaults maps provider name → {baseURL, model}.
var ProviderDefaults = map[string][2]string{
	"openai": {"https://api.openai.com", "tts-1"},
}

// Provider implements speech.Provider for OpenAI-compatible text-to-speech.
type Provider struct {
	name         string
	configPrefix string
	config       Config
	mu           sync.RWMutex
	initialized  bool
}

func NewProvider(name string) *Provider { return &Provider{name: name} }

func (p *Provider) Name() string { return p.name }

func (p *Provider) SetConfigPrefix(prefix string) { p.configPrefix = prefix }

func (p *Provider) Init(_ context.Context, cfg config.IConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.initialized {
		return nil
	}
	key := p.configPrefix
	if key == "" {
		key = "chat." + p.name
	}
	var oc Config
	if err := cfg.UnmarshalKey(key, &oc); err != nil {
		return fmt.Errorf("load %s config failed: %w", key, err)
	}
	if def, ok := ProviderDefaults[p.name]; ok {
		if oc.BaseURL == "" {
			oc.BaseURL = def[0]
		}
		if oc.Model == "" {
			oc.Model = def[1]
		}
	}
	if oc.BaseURL == "" {
		return fmt.Errorf("%s.baseUrl is required", key)
	}
	p.config = oc
	p.initialized = true
	log.Info("OpenAI speech provider initialized",
		zap.String("name", p.name),
		zap.String("baseUrl", oc.GetBaseURL()),
		zap.String("model", oc.GetModel()))
	return nil
}

func (p *Provider) GetModels() []string { return []string{p.config.GetModel()} }

func (p *Provider) GetVoices() []speech.Voice {
	// OpenAI TTS voices
	return []speech.Voice{
		{ID: "alloy", Name: "Alloy", Language: "en"},
		{ID: "echo", Name: "Echo", Language: "en"},
		{ID: "fable", Name: "Fable", Language: "en"},
		{ID: "onyx", Name: "Onyx", Language: "en"},
		{ID: "nova", Name: "Nova", Language: "en"},
		{ID: "shimmer", Name: "Shimmer", Language: "en"},
	}
}

func (p *Provider) GetProviderInfo() types.ProviderInfo {
	if def, ok := ProviderDefaults[p.name]; ok {
		return types.ProviderInfo{Model: def[1], BaseURL: def[0]}
	}
	return types.ProviderInfo{Model: p.config.GetModel(), BaseURL: p.config.GetBaseURL()}
}

// Synthesize implements speech.Provider.Synthesize.
// It calls the OpenAI-compatible /v1/audio/speech endpoint.
func (p *Provider) Synthesize(text string, opts *speech.SynthesizeOptions) (*speech.SynthesizeResult, error) {
	if err := p.checkInitialized(); err != nil {
		return nil, err
	}

	model := p.config.GetModel()
	voice := "alloy"
	format := "mp3"

	if opts != nil {
		if opts.Model != "" {
			model = opts.Model
		}
		if opts.Voice != "" {
			voice = opts.Voice
		}
		if opts.Format != "" {
			format = opts.Format
		}
	}

	body := map[string]any{
		"model": model,
		"input": text,
		"voice": voice,
	}
	if format != "mp3" {
		body["response_format"] = format
	}

	data, err := p.postJSON(context.Background(), p.config.GetBaseURL()+"/v1/audio/speech", body)
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(".", "data", "gen", "audio")
	_ = os.MkdirAll(dir, 0755)
	filename := fmt.Sprintf("%d_%d.%s", time.Now().Unix(), time.Now().Nanosecond(), format)
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil, err
	}

	return &speech.SynthesizeResult{
		AudioData: data,
		Format:    format,
		URL:       "/api/files/gen/audio/" + filename,
	}, nil
}

func (p *Provider) checkInitialized() error {
	if !p.initialized {
		return fmt.Errorf("OpenAI speech provider %s not initialized", p.name)
	}
	return nil
}

func (p *Provider) postJSON(ctx context.Context, url string, body map[string]any) ([]byte, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("speech API error %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}
