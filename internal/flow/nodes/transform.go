package nodes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/chuccp/go-ai-agent/internal/flow/engine"
)

// TransformNodeConfig Data transform node config
type TransformNodeConfig struct {
	Template string `json:"template"` // Go template, outputs JSON
}

// TransformNode Use Go template to transform upstream data
// All upstream node outputs accessible in template: {{.generate_story.output}} {{.split.count}}
type TransformNode struct{}

func NewTransformNode() *TransformNode { return &TransformNode{} }

func (n *TransformNode) Type() string { return "transform" }

func (n *TransformNode) Execute(ctx *engine.ExecutionContext, config string) (*engine.NodeOutput, error) {
	cfg, err := engine.GetNodeConfig[TransformNodeConfig](config)
	if err != nil {
		return nil, err
	}
	if cfg.Template == "" {
		return nil, fmt.Errorf("transform: template is required")
	}

	// Build template data: all upstream node outputs + global data
	data := make(map[string]any)
	for k, v := range ctx.Data {
		data[k] = v
	}
	// Use NodeOutputs as nested structure for template access, e.g. .generate_story.output
	for label, output := range ctx.AllNodeOutputs() {
		data[label] = output.Data
	}

	tmpl, err := template.New("transform").Parse(cfg.Template)
	if err != nil {
		return nil, fmt.Errorf("transform: template parse error: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("transform: template execute error: %w", err)
	}

	result := buf.String()

	// Attempt JSON parse for validation
	var jsonCheck any
	isJSON := json.Unmarshal([]byte(result), &jsonCheck) == nil

	return &engine.NodeOutput{
		Data: map[string]any{
			KeyOutput: result,
			"is_json": isJSON,
		},
		Status: engine.StatusSuccess,
	}, nil
}
