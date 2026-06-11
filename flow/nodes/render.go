package nodes

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/chuccp/go-ai-agent/flow/engine"
)

// 匹配 {{任意内容}} 占位符
var placeholderRe = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// renderPrompt 替换 prompt 中的 {{label.field}} 和 {{label}} 占位符
func renderPrompt(tmpl string, ctx *engine.ExecutionContext) string {
	if tmpl == "" {
		return ""
	}
	return placeholderRe.ReplaceAllStringFunc(tmpl, func(match string) string {
		key := match[2 : len(match)-2]

		// 精确匹配 label.field
		if v, ok := ctx.Get(key); ok {
			return fmt.Sprintf("%v", v)
		}

		// 匹配整个 label，返回 JSON
		if output, ok := ctx.GetNodeOutput(key); ok {
			b, err := json.MarshalIndent(output.Data, "", "  ")
			if err != nil {
				return match
			}
			return string(b)
		}

		// label.field 的后缀匹配
		for label, output := range ctx.AllNodeOutputs() {
			if prefix := label + "."; len(key) > len(prefix) && key[:len(prefix)] == prefix {
				field := key[len(prefix):]
				if v, ok := output.Data[field]; ok {
					return fmt.Sprintf("%v", v)
				}
			}
		}

		return match
	})
}

// AllNodeOutputs returns all node outputs (used by renderPrompt for suffix matching)
// Defined as a method on ExecutionContext below
