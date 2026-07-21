package main

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// Model 是 Bubble Tea 的状态模型
type Model struct {
	messages []string // 已提交的消息列表
	input    string   // 当前输入框内容
	width    int      // 终端宽度
	height   int      // 终端高度
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "enter":
			if strings.TrimSpace(m.input) != "" {
				m.messages = append(m.messages, m.input)
				m.input = ""
			}

		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}

		default:
			// KeyPressMsg.Text 包含实际输入的字符
			if msg.Text != "" {
				m.input += msg.Text
			}
		}
	}

	return m, nil
}

func (m Model) View() tea.View {
	var b strings.Builder

	// 标题
	b.WriteString("💬 Bubble Tea Chat\n")
	b.WriteString("Type a message and press Enter. Ctrl+C to quit.\n")
	b.WriteString(strings.Repeat("─", m.width) + "\n")

	// 消息列表（只显示最近的消息，适配窗口高度）
	visibleLines := m.height - 6
	if visibleLines < 1 {
		visibleLines = 1
	}
	start := 0
	if len(m.messages) > visibleLines {
		start = len(m.messages) - visibleLines
	}
	for _, msg := range m.messages[start:] {
		b.WriteString(fmt.Sprintf("  → %s\n", msg))
	}

	// 填充空白行
	for i := len(m.messages[start:]); i < visibleLines; i++ {
		b.WriteString("\n")
	}

	// 分隔线
	divider := strings.Repeat("─", m.width)
	if divider == "" {
		divider = "────────────────────────────────────────"
	}
	b.WriteString(divider + "\n")

	// 输入行
	if m.input == "" {
		b.WriteString("> _\n")
	} else {
		b.WriteString(fmt.Sprintf("> %s\n", m.input))
	}

	return tea.NewView(b.String())
}

func main() {
	p := tea.NewProgram(Model{})
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
	}
}
