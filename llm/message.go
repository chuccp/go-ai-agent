package llm

type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ContentPart struct {
	Type     string `json:"type"` // "text" | "image"
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"` // Image URL or base64 data URL
}

type ChatMessage struct {
	Role         string        `json:"role"`
	Content      string        `json:"content"`
	ContentParts []ContentPart `json:"content_parts,omitempty"` // Multimodal content; provider uses this field when non-empty
	ToolCalls    []ToolCall    `json:"tool_calls,omitempty"`    // Tool calls from assistant (native function calling)
	ToolCallID   string        `json:"tool_call_id,omitempty"`  // Tool result message: ID of the tool call this responds to
	Name         string        `json:"name,omitempty"`          // Tool result message: name of the tool
}

type ChatMessages struct {
	model string
}

type ChatResponse struct {
}
