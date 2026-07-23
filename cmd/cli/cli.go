package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/chuccp/go-web-frame/core"
)

type handleMsg func(tea.Msg) tea.Cmd

// ── Styles ─────────────────────────────────────────────────────────────

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Padding(0, 1)

	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)

	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	assistantStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("117")).
			Bold(true)

	assistantTextStyle = lipgloss.NewStyle().
				PaddingLeft(2)

	cmdStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("221"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))
)

// ── Message ────────────────────────────────────────────────────────────

type Message struct {
	Role    string // "user", "assistant", "system"
	Content string
}

// ── Model ──────────────────────────────────────────────────────────────

// Model is the Bubble Tea model for the chat TUI.
type Model struct {
	messages []Message
	input    textinput.Model
	width    int
	height   int
	quitting bool

	// Streaming state
	streaming  bool
	streamFull string
	streamPos  int

	// Scroll offset (for future: PgUp/PgDn to scroll history)
	scrollOffset int
	ctx          *core.Context
	handleMsg    handleMsg
}

// streamTick advances the typewriter effect.
type streamTick time.Time

// NewModel creates a new chat Model.
func NewModel(ctx *core.Context) *Model {
	ti := textinput.New()
	ti.Placeholder = "Type a message... ( /help for commands )"
	ti.CharLimit = 4000
	ti.Width = 60
	ti.Focus()
	return &Model{
		ctx:      ctx,
		messages: make([]Message, 0),
		input:    ti,
	}
}

// ── Init ───────────────────────────────────────────────────────────────

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// ── Update ─────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = msg.Width - 4
		return m, nil

	case tea.KeyMsg:
		if m.streaming {
			return m, nil // ignore input while streaming
		}

		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			text := strings.TrimSpace(m.input.Value())
			if text == "" {
				return m, nil
			}
			if strings.HasPrefix(text, "/") {
				response := m.handleCommand(text)
				if response == "__QUIT__" {
					m.quitting = true
					return m, tea.Quit
				}
				m.messages = append(m.messages, Message{Role: "system", Content: text})
				if response != "" {
					m.messages = append(m.messages, Message{Role: "assistant", Content: response})
				}
			} else {
				//m.messages = append(m.messages, Message{Role: "user", Content: text})
				//m.startStreaming(text)
			}
			m.input.Reset()
			return m, m.startStreamTick()

		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
		}

	case streamTick:
		if m.streaming {
			advance := 3
			if remaining := len(m.streamFull) - m.streamPos; advance > remaining {
				advance = remaining
			}
			m.streamPos += advance

			if m.streamPos >= len(m.streamFull) {
				m.messages = append(m.messages, Message{
					Role:    "assistant",
					Content: m.streamFull,
				})
				m.streaming = false
				m.streamFull = ""
				m.streamPos = 0
				return m, nil
			}
			return m, tickCmd()
		}
	}

	return m, tea.Batch(cmds...)
}

// ── Streaming ──────────────────────────────────────────────────────────

func tickCmd() tea.Cmd {
	return tea.Tick(15*time.Millisecond, func(t time.Time) tea.Msg {
		return streamTick(t)
	})
}

func (m *Model) startStreaming(userText string) {
	m.streamFull = fmt.Sprintf(
		"I received: %q\n\nThis is a simulated response. "+
			"Hook up a real AI provider to get actual responses. "+
			"The streaming typewriter effect works — each character "+
			"appears one by one just like Claude Code does.",
		userText,
	)
	m.streamPos = 0
	m.streaming = true
}

func (m *Model) startStreamTick() tea.Cmd {
	return tickCmd()
}

// ── Commands ───────────────────────────────────────────────────────────

func (m *Model) handleCommand(text string) string {
	parts := strings.Fields(text)
	cmd := parts[0]

	switch cmd {
	case "/help":
		return fmt.Sprintf(`Commands:
  %s  Show this help
  %s Clear chat history
  %s List available models
  %s Exit the chat`,
			"/help", "/clear", "/models", "/quit")

	case "/clear":
		m.messages = m.messages[:0]
		return "✓ Chat history cleared."

	case "/models":
		return fmt.Sprintf(`Available models:
  • claude-opus-4-8
  • claude-sonnet-4-6
  • deepseek-v4-flash
  • deepseek-v4-pro
  • gpt-4o`)

	case "/quit", "/exit":
		return "__QUIT__"

	default:
		return fmt.Sprintf("Unknown command: %s (type /help)", cmd)
	}
}
func (m Model) View() string {
	if m.quitting {
		return dimStyle.Render("Bye! 👋") + "\n"
	}
	var b strings.Builder
	// ── Top bar ──
	b.WriteString(titleStyle.Render("💬 Chat CLI"))
	b.WriteString(dimStyle.Render(" — /help for commands  |  Ctrl+C to quit") + "\n")
	b.WriteString(dividerStyle.Render(strings.Repeat("─", m.width)) + "\n")

	// ── Messages area (fill remaining space) ──
	msgArea := m.height - 4 // top(2) + bottom divider(1) + input(1)
	if msgArea < 1 {
		msgArea = 1
	}

	lines := m.collectMessageLines()
	// Include streaming partial text
	if m.streaming {
		streamHeader := assistantStyle.Render("  🤖 Assistant:")
		partial := m.streamFull[:m.streamPos]
		cursor := ""
		if m.streamPos < len(m.streamFull) {
			cursor = lipgloss.NewStyle().Foreground(lipgloss.Color("117")).Render("▊")
		}
		lines = append(lines, streamHeader)
		for _, l := range strings.Split(assistantTextStyle.Render(partial+cursor), "\n") {
			lines = append(lines, l)
		}
	}

	// Show only the last N lines that fit
	start := 0
	if len(lines) > msgArea {
		start = len(lines) - msgArea
	}
	for i := start; i < len(lines); i++ {
		b.WriteString(lines[i] + "\n")
	}

	// Fill remaining space with blank lines so input stays at bottom
	for i := len(lines) - start; i < msgArea; i++ {
		b.WriteString("\n")
	}

	// ── Bottom bar ──
	b.WriteString(dividerStyle.Render(strings.Repeat("─", m.width)) + "\n")
	b.WriteString(promptStyle.Render("> ") + m.input.View() + "\n")

	return b.String()
}

// ── Message rendering ──────────────────────────────────────────────────

// collectMessageLines renders all messages to individual lines, newest last.
func (m Model) collectMessageLines() []string {
	if len(m.messages) == 0 {
		return []string{dimStyle.Render("  Start typing to chat...")}
	}

	var lines []string
	for _, msg := range m.messages {
		switch msg.Role {
		case "user":
			lines = append(lines, userStyle.Render(fmt.Sprintf("  You: %s", msg.Content)))
		case "assistant":
			lines = append(lines, assistantStyle.Render("  🤖 Assistant:"))
			for _, l := range strings.Split(assistantTextStyle.Render(msg.Content), "\n") {
				lines = append(lines, l)
			}
		case "system":
			lines = append(lines, cmdStyle.Render(fmt.Sprintf("  %s", msg.Content)))
		}
		lines = append(lines, "") // blank separator between messages
	}
	return lines
}

// ── Simple REPL fallback (no TTY required) ─────────────────────────────

// RunSimpleREPL is a standard-library chat REPL that works in any terminal,
// including IDE environments without /dev/tty (GoLand, etc.).
func RunSimpleREPL() {
	const (
		Reset  = "\033[0m"
		Bold   = "\033[1m"
		Dim    = "\033[2m"
		Green  = "\033[32m"
		Cyan   = "\033[36m"
		Yellow = "\033[33m"
		Red    = "\033[31m"
	)
	style := func(s, code string) string { return code + s + Reset }

	fmt.Println(style("💬 Chat CLI", Bold) + style(" — /help for commands", Dim))
	fmt.Println(style(strings.Repeat("─", 60), Dim))
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	messages := make([]Message, 0)

	fmt.Print(style("> ", Green))

	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			fmt.Print(style("> ", Green))
			continue
		}

		// Slash commands
		if strings.HasPrefix(text, "/") {
			parts := strings.Fields(text)
			switch parts[0] {
			case "/help":
				fmt.Println("\n  " + style("Commands:", Bold))
				for _, l := range []string{
					"/help   Show this help",
					"/clear  Clear history",
					"/models List models",
					"/quit   Exit",
				} {
					fmt.Printf("  %s\n", style(l, Yellow))
				}
			case "/clear":
				messages = messages[:0]
				fmt.Printf("  %s\n", style("✓ Chat history cleared.", Green))
			case "/models":
				fmt.Println("\n  " + style("Available models:", Bold))
				for _, m := range []string{
					"claude-opus-4-8", "claude-sonnet-4-6",
					"deepseek-v4-flash", "deepseek-v4-pro", "gpt-4o",
				} {
					fmt.Printf("    • %s\n", m)
				}
			case "/quit", "/exit":
				fmt.Println(style("\nBye! 👋", Dim))
				return
			default:
				fmt.Printf("  %s (type %s)\n",
					style("Unknown: "+parts[0], Red), style("/help", Dim))
			}
			fmt.Println()
			fmt.Print(style("> ", Green))
			continue
		}

		// Chat message
		messages = append(messages, Message{Role: "user", Content: text})
		fmt.Printf("  %s %s\n", style("You:", Dim), text)

		// Simulated assistant response with typewriter effect
		response := fmt.Sprintf(
			"Received: %q. This is a simulated response — "+
				"hook up a real AI provider later.", text,
		)
		fmt.Print(style("  🤖 Assistant:", Bold+Cyan) + "\n  ")
		for _, ch := range response {
			fmt.Print(string(ch))
			time.Sleep(12 * time.Millisecond)
		}
		fmt.Println()
		fmt.Println()

		messages = append(messages, Message{Role: "assistant", Content: response})
		fmt.Print(style("> ", Green))
	}
}
