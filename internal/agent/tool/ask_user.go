package tool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/chuccp/go-ai-agent/internal/agent/question"
)

// AskUser is the opencode-style "question" tool. The LLM calls it to ask the
// user one or more questions and BLOCKS until the user responds (or rejects,
// or the context is cancelled). The agent loop is naturally suspended while
// this tool executes — no system-prompt soft constraint is required.
type AskUser struct {
	questionSvc QuestionService
}

func (t *AskUser) SetQuestionService(svc QuestionService) {
	t.questionSvc = svc
}

func (t *AskUser) Definition() Definition {
	return Definition{
		Name: "ask_user",
		Description: `Ask the user questions and wait for their answers. This tool BLOCKS until the user responds — do NOT end your turn or relay the question yourself; the frontend shows the question UI automatically.

Use this tool to:
1. Gather user preferences or requirements
2. Clarify ambiguous instructions
3. Get decisions on implementation choices
4. Offer choices about what direction to take

Usage notes:
- When "custom" is enabled (default), a "Type your own answer" option is added automatically; don't include "Other" or catch-all options.
- Answers are returned as arrays of labels; set "multiple": true to allow selecting more than one.
- If you recommend a specific option, make that the first option in the list and add "(Recommended)" at the end of the label.
- Never guess when uncertain — call ask_user instead.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"questions": map[string]any{
					"type":        "array",
					"description": "Questions to ask",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"question": map[string]any{
								"type":        "string",
								"description": "Complete question",
							},
							"header": map[string]any{
								"type":        "string",
								"description": "Very short label (max 30 chars)",
							},
							"options": map[string]any{
								"type":        "array",
								"description": "Available choices",
								"items": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"label": map[string]any{
											"type":        "string",
											"description": "Display text (1-5 words, concise)",
										},
										"description": map[string]any{
											"type":        "string",
											"description": "Explanation of choice",
										},
									},
								},
							},
							"multiple": map[string]any{
								"type":        "boolean",
								"description": "Allow selecting multiple choices",
							},
							"custom": map[string]any{
								"type":        "boolean",
								"description": "Allow typing a custom answer (default: true)",
							},
						},
						"required": []string{"question", "header"},
					},
				},
			},
			"required": []string{"questions"},
		},
	}
}

func (t *AskUser) Execute(ctx context.Context, call Call) (string, error) {
	if t.questionSvc == nil {
		return "", fmt.Errorf("question service not initialized")
	}

	var params struct {
		Questions []question.Question `json:"questions"`
	}
	if err := json.Unmarshal([]byte(call.Arguments), &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if len(params.Questions) == 0 {
		return "", fmt.Errorf("questions is required")
	}

	sessionID := SessionIDFrom(ctx)
	res := t.questionSvc.Ask(sessionID, call.ID, params.Questions)

	answers, rejected, err := question.Wait(ctx, res)
	if err != nil {
		return "", err
	}
	if rejected {
		return "The user dismissed this question.", nil
	}

	// Format the answers as model-facing prose, mirroring opencode's question tool.
	formatted := ""
	for i, q := range params.Questions {
		ans := ""
		if i < len(answers) && len(answers[i]) > 0 {
			ans = joinLabels(answers[i])
		} else {
			ans = "Unanswered"
		}
		if i > 0 {
			formatted += ", "
		}
		formatted += fmt.Sprintf("%q=%q", q.Question, ans)
	}
	return fmt.Sprintf("User has answered your questions: %s. You can now continue with the user's answers in mind.", formatted), nil
}

// joinLabels joins answer labels with ", " (multi-select case).
func joinLabels(labels []string) string {
	out := ""
	for i, l := range labels {
		if i > 0 {
			out += ", "
		}
		out += l
	}
	return out
}
