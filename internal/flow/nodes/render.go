package nodes

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/chuccp/go-ai-agent/internal/flow/engine"
)

// Match {{anything}} placeholders
var placeholderRe = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// renderPrompt Replace {{label.field}} and {{label}} placeholders in prompt
func renderPrompt(tmpl string, ctx *engine.ExecutionContext) string {
	if tmpl == "" {
		return ""
	}
	return placeholderRe.ReplaceAllStringFunc(tmpl, func(match string) string {
		key := match[2 : len(match)-2]

		// Exact match label.field
		if v, ok := ctx.Get(key); ok {
			return fmt.Sprintf("%v", v)
		}

		// Match entire label, return JSON
		if output, ok := ctx.GetNodeOutput(key); ok {
			b, err := json.MarshalIndent(output.Data, "", "  ")
			if err != nil {
				return match
			}
			return string(b)
		}

		// Suffix match for label.field
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
