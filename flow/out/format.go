package out

// OutFormat defines a structured output format
type OutFormat struct {
	Type       string       `json:"type"`                  // "text", "json"
	JSONSchema *JSONSchema  `json:"json_schema,omitempty"` // JSON schema definition
	Example    string       `json:"example,omitempty"`     // Example JSON
}

// JSONSchema defines a JSON structure
type JSONSchema struct {
	Type       string                 `json:"type"`                  // "object", "array"
	Properties map[string]SchemaField `json:"properties,omitempty"`  // Object fields
	Items      *SchemaField           `json:"items,omitempty"`       // Array element type
}

// SchemaField defines a single field
type SchemaField struct {
	Type        string `json:"type"`                  // "string", "number", "integer", "boolean", "array", "object"
	Description string `json:"description,omitempty"` // Field description
}

// BuildPromptInstruction generates a system prompt instruction from the OutFormat
func (f *OutFormat) BuildPromptInstruction() string {
	if f == nil || f.Type == "text" {
		return ""
	}
	if f.Type != "json" {
		return ""
	}

	if f.Example != "" {
		return "Output valid JSON based on this example:\n" + f.Example + "\nDo not include any other content."
	}

	if f.JSONSchema == nil {
		return "Output valid JSON."
	}

	schema := f.JSONSchema
	var inst string
	switch schema.Type {
	case "array":
		inst = "Output a pure JSON array"
		if schema.Items != nil {
			inst += ", each element: " + schema.Items.Description
		}
	case "object":
		inst = "Output a pure JSON object"
		if len(schema.Properties) > 0 {
			inst += " with the following fields:\n"
			for name, field := range schema.Properties {
				inst += "- " + name + " (" + field.Type + "): " + field.Description + "\n"
			}
		}
	default:
		inst = "Output valid JSON"
	}
	return inst + "\nDo not include any other content."
}

// NewTextOutFormat creates a text output format
func NewTextOutFormat() *OutFormat {
	return &OutFormat{Type: "text"}
}

// NewJSONOutFormat creates a JSON output format
func NewJSONOutFormat() *OutFormat {
	return &OutFormat{Type: "json"}
}

// NewExampleJSONOutFormat creates an output format from an example JSON
func NewExampleJSONOutFormat(example string) *OutFormat {
	return &OutFormat{Type: "json", Example: example}
}
