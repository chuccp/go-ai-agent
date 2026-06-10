package out

// OutFormat 结构化输出格式定义
type OutFormat struct {
	Type       string       `json:"type"`                  // "text", "json"
	JSONSchema *JSONSchema  `json:"json_schema,omitempty"` // JSON schema 定义
	Example    string       `json:"example,omitempty"`     // 示例 JSON
}

// JSONSchema JSON 结构定义
type JSONSchema struct {
	Type       string                 `json:"type"`                  // "object", "array"
	Properties map[string]SchemaField `json:"properties,omitempty"`  // 对象字段
	Items      *SchemaField           `json:"items,omitempty"`       // 数组元素类型
}

// SchemaField 字段定义
type SchemaField struct {
	Type        string `json:"type"`                  // "string", "number", "integer", "boolean", "array", "object"
	Description string `json:"description,omitempty"` // 字段描述
}

// BuildPromptInstruction 根据 OutFormat 生成 system prompt 指令
func (f *OutFormat) BuildPromptInstruction() string {
	if f == nil || f.Type == "text" {
		return ""
	}
	if f.Type != "json" {
		return ""
	}

	if f.Example != "" {
		return "请输出合法JSON，参考以下示例：\n" + f.Example + "\n不要包含其他内容。"
	}

	if f.JSONSchema == nil {
		return "请输出合法JSON。"
	}

	schema := f.JSONSchema
	var inst string
	switch schema.Type {
	case "array":
		inst = "请输出纯JSON数组"
		if schema.Items != nil {
			inst += "，每个元素为" + schema.Items.Description
		}
	case "object":
		inst = "请输出纯JSON对象"
		if len(schema.Properties) > 0 {
			inst += "，包含以下字段：\n"
			for name, field := range schema.Properties {
				inst += "- " + name + " (" + field.Type + "): " + field.Description + "\n"
			}
		}
	default:
		inst = "请输出合法JSON"
	}
	return inst + "\n不要包含其他内容。"
}

// NewTextOutFormat 创建文本输出格式
func NewTextOutFormat() *OutFormat {
	return &OutFormat{Type: "text"}
}

// NewJSONOutFormat 创建 JSON 输出格式
func NewJSONOutFormat() *OutFormat {
	return &OutFormat{Type: "json"}
}

// NewExampleJSONOutFormat 从示例 JSON 创建输出格式
func NewExampleJSONOutFormat(example string) *OutFormat {
	return &OutFormat{Type: "json", Example: example}
}
