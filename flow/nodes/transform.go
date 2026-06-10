package nodes

import (
	"bytes"
	"github.com/bytedance/sonic"
	"fmt"
	"text/template"

	"github.com/chuccp/go-ai-agent/flow/engine"
)

// TransformNodeConfig 数据变换节点配置
type TransformNodeConfig struct {
	Template string `json:"template"` // Go template，输出 JSON
}

// TransformNode 使用 Go template 对上游数据进行变换
// 模板中可访问所有上游节点输出：{{.生成故事.output}} {{.分段.count}}
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

	// 构建模板数据：所有上游节点输出 + 全局数据
	data := make(map[string]any)
	for k, v := range ctx.Data {
		data[k] = v
	}
	// 把 NodeOutputs 作为嵌套结构，方便模板中 .生成故事.output 访问
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

	// 尝试解析为 JSON 验证
	var jsonCheck any
	isJSON := sonic.Unmarshal([]byte(result), &jsonCheck) == nil

	return &engine.NodeOutput{
		Data: map[string]any{
			KeyOutput: result,
			"is_json": isJSON,
		},
		Status: engine.StatusSuccess,
	}, nil
}
