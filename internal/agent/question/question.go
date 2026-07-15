// Package question implements an opencode-style question service.
//
// It provides a blocking ask/reply mechanism that lets the agent (and tools)
// ask the user questions and suspend until the user responds. This mirrors
// opencode's packages/opencode/src/question/index.ts:
//
//   - Ask registers a question, emits an "asked" event to the frontend, and
//     returns a channel that blocks until the user replies (or rejects).
//   - Reply / Reject wake the blocked caller.
//
// The agent loop is naturally paused while a tool blocks on Ask — no soft
// "relay the waiting_prompt" system-prompt rule is needed, and no LLM
// cooperation is required (works with weak instruction-following models).
package question

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// Option is one selectable choice in a question.
type Option struct {
	Label       string `json:"label"`       // Display text (1-5 words, concise)
	Description string `json:"description"` // Explanation of choice
}

// Question is a single prompt shown to the user.
type Question struct {
	Question string   `json:"question"`         // Complete question
	Header   string   `json:"header"`           // Very short label (max 30 chars)
	Options  []Option `json:"options,omitempty"` // Available choices
	Multiple bool     `json:"multiple,omitempty"`
	Custom   bool     `json:"custom,omitempty"` // Allow custom answer (default true)
}

// Request is a registered question pending a user answer.
type Request struct {
	ID        uint64     `json:"id"`
	SessionID uint       `json:"session_id"`
	ToolCall  string     `json:"tool_call,omitempty"` // originating tool call ID (optional)
	Questions []Question `json:"questions"`
}

// Answer is the user's reply: one array of selected labels per question, in order.
type Answer [][]string

// AskResult holds the channels returned by Ask. Exactly one will deliver.
type AskResult struct {
	ID       uint64
	Answers  <-chan Answer
	Rejected <-chan struct{}
}

type pendingEntry struct {
	req        Request
	answerCh   chan Answer
	rejectedCh chan struct{}
}

// Service manages pending questions and is the Go equivalent of opencode's
// Question.Service. It is concurrency-safe.
type Service struct {
	mu      sync.Mutex
	nextID  uint64
	pending map[uint64]*pendingEntry
	onAsk   func(req Request) // broadcast callback (set by ChatRunner)
}

// NewService creates a QuestionService. onAsk is invoked whenever a new
// question is registered; the callback is responsible for forwarding the
// request to the frontend via WebSocket.
func NewService(onAsk func(req Request)) *Service {
	return &Service{
		pending: make(map[uint64]*pendingEntry),
		onAsk:   onAsk,
	}
}

// SetOnAsk replaces the broadcast callback (used when the service is created
// before the broadcaster is wired up).
func (s *Service) SetOnAsk(fn func(req Request)) {
	s.mu.Lock()
	s.onAsk = fn
	s.mu.Unlock()
}

// Ask registers a question, invokes onAsk to notify the frontend, and returns
// channels that block until the user replies or rejects. The caller (a tool)
// is expected to select on Answers / Rejected / ctx.Done().
func (s *Service) Ask(sessionID uint, toolCallID string, questions []Question) AskResult {
	id := atomic.AddUint64(&s.nextID, 1)
	entry := &pendingEntry{
		req: Request{
			ID:        id,
			SessionID: sessionID,
			ToolCall:  toolCallID,
			Questions: questions,
		},
		answerCh:   make(chan Answer, 1),
		rejectedCh: make(chan struct{}, 1),
	}
	s.mu.Lock()
	s.pending[id] = entry
	onAsk := s.onAsk
	s.mu.Unlock()

	if onAsk != nil {
		onAsk(entry.req)
	}
	return AskResult{ID: id, Answers: entry.answerCh, Rejected: entry.rejectedCh}
}

// Reply delivers the user's answers to the blocked Ask caller.
func (s *Service) Reply(id uint64, answers Answer) error {
	s.mu.Lock()
	entry, ok := s.pending[id]
	if ok {
		delete(s.pending, id)
	}
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("question not found: %d", id)
	}
	entry.answerCh <- answers
	return nil
}

// Reject signals the blocked Ask caller that the user dismissed the question.
func (s *Service) Reject(id uint64) error {
	s.mu.Lock()
	entry, ok := s.pending[id]
	if ok {
		delete(s.pending, id)
	}
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("question not found: %d", id)
	}
	close(entry.rejectedCh)
	return nil
}

// List returns all currently pending questions (snapshot).
func (s *Service) List() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Request, 0, len(s.pending))
	for _, e := range s.pending {
		out = append(out, e.req)
	}
	return out
}

// CancelAll rejects every pending question. Called when a session is aborted
// or the runner shuts down so blocked tools unblock promptly.
func (s *Service) CancelAll() {
	s.mu.Lock()
	ids := make([]uint64, 0, len(s.pending))
	for id, e := range s.pending {
		ids = append(ids, id)
		close(e.rejectedCh)
	}
	s.pending = make(map[uint64]*pendingEntry)
	s.mu.Unlock()
	_ = ids
}

// Wait blocks until the user answers, rejects, or ctx is cancelled.
// Returns the answers, a rejected flag, and a ctx-error flag.
func Wait(ctx context.Context, r AskResult) (Answer, bool, error) {
	select {
	case a := <-r.Answers:
		return a, false, nil
	case <-r.Rejected:
		return nil, true, nil
	case <-ctx.Done():
		return nil, false, ctx.Err()
	}
}
