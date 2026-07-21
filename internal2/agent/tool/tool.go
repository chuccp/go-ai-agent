package tool

import (
	"context"

	"github.com/chuccp/go-ai-agent/internal2/agent/question"
)

// sessionIDKey is the context key for the active chat session ID. It lets
// tools (e.g. ask_user) know which session they're running in so they can
// route frontend events to the right connection. Set by the agent runner
// via WithSessionID.
type sessionIDKey struct{}

// WithSessionID returns a context carrying the session ID.
func WithSessionID(ctx context.Context, sessionID uint) context.Context {
	return context.WithValue(ctx, sessionIDKey{}, sessionID)
}

// SessionIDFrom extracts the session ID from a context (0 if absent).
func SessionIDFrom(ctx context.Context) uint {
	if v, ok := ctx.Value(sessionIDKey{}).(uint); ok {
		return v
	}
	return 0
}

// Definition is a tool definition in Anthropic standard format
type Definition struct {
	Type        string `json:"type,omitempty"`         // Only needed for built-in tools, e.g. "web_search_20250305"
	Name        string `json:"name"`                   // Function name
	Description string `json:"description,omitempty"`  // Description
	InputSchema any    `json:"input_schema,omitempty"` // JSON Schema (custom tools)
}

// Call is a tool call request
type Call struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string of input
}

// Result is the result of executing a tool
type Result struct {
	CallID  string `json:"call_id"`
	Output  string `json:"output"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// Executor is the interface for tool executors
type Executor interface {
	Execute(ctx context.Context, call Call) (string, error)
	Definition() Definition
}

// FlowActionHandler handles flow CRUD operations (injected by runner)
type FlowActionHandler func(action string, args map[string]any) (string, error)

// FlowExecutionHandler handles flow execution operations (injected by runner).
// The context allows cancellable blocking operations (e.g. waiting for user input
// during flow execution). The tool blocks until the flow completes or the context
// is cancelled — the agent loop is naturally paused during this time, mirroring
// opencode's Deferred-based question tool pattern.
type FlowExecutionHandler func(ctx context.Context, action string, args map[string]any) (string, error)

// ModelActionHandler handles AI model CRUD operations (injected by runner)
type ModelActionHandler func(action string, params map[string]any) (string, error)

// FlowHandlerSetter is implemented by tools that need a flow action handler.
// Registry.Register auto-injects the handler via this interface.
type FlowHandlerSetter interface {
	SetFlowHandler(handler FlowActionHandler)
}

// FlowExecutionHandlerSetter is implemented by tools that need a flow execution handler.
type FlowExecutionHandlerSetter interface {
	SetFlowExecutionHandler(handler FlowExecutionHandler)
}

// ModelHandlerSetter is implemented by tools that need a model action handler.
// Registry.Register auto-injects the handler via this interface.
type ModelHandlerSetter interface {
	SetModelHandler(handler ModelActionHandler)
}

// QuestionService is the interface tools need to ask the user questions.
// Implemented by internal/agent/question.Service.
type QuestionService interface {
	Ask(sessionID uint, toolCallID string, questions []question.Question) question.AskResult
}

// QuestionServiceSetter is implemented by tools that need the QuestionService.
// Registry.Register auto-injects the service via this interface.
type QuestionServiceSetter interface {
	SetQuestionService(svc QuestionService)
}
