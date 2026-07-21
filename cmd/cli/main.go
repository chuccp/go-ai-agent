package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Model struct {
	messages []string
	scanner  *bufio.Scanner
	running  bool
	handlers map[string]func(args []string) bool
}

func New() *Model {
	m := &Model{
		scanner: bufio.NewScanner(os.Stdin),
		running: true,
	}
	m.handlers = map[string]func(args []string) bool{
		"/help":    m.cmdHelp,
		"/history": m.cmdHistory,
		"/clear":   m.cmdClear,
		"/models":  m.cmdModels,
		"/quit":    m.cmdQuit,
	}
	return m
}

func (m *Model) Run() {
	fmt.Println("💬 Chat CLI  (type /help for commands, /quit to exit)")
	fmt.Println(strings.Repeat("─", 60))

	for m.running {
		fmt.Print("> ")
		if !m.scanner.Scan() {
			break
		}
		m.handle(m.scanner.Text())
	}

	if err := m.scanner.Err(); err != nil {
		fmt.Println("Error:", err)
	}
}

func (m *Model) handle(line string) {
	text := strings.TrimSpace(line)
	if text == "" {
		return
	}

	// 斜杠命令
	if strings.HasPrefix(text, "/") {
		parts := strings.Fields(text)
		cmd := parts[0]
		if fn, ok := m.handlers[cmd]; ok {
			fn(parts[1:])
		} else {
			fmt.Printf("  Unknown command: %s (type /help)\n", cmd)
		}
		return
	}

	// 普通消息
	m.messages = append(m.messages, text)
	fmt.Printf("  → %s\n\n", text)
}

func (m *Model) cmdHelp(_ []string) bool {
	fmt.Println("  Commands:")
	fmt.Println("    /help       Show this help")
	fmt.Println("    /history    Show chat history")
	fmt.Println("    /clear      Clear chat history")
	fmt.Println("    /models     List available AI models")
	fmt.Println("    /quit       Exit program")
	return true
}

func (m *Model) cmdHistory(_ []string) bool {
	if len(m.messages) == 0 {
		fmt.Println("  (no messages yet)")
		return true
	}
	for i, msg := range m.messages {
		fmt.Printf("  [%d] %s\n", i+1, msg)
	}
	return true
}

func (m *Model) cmdClear(_ []string) bool {
	m.messages = nil
	fmt.Println("  ✓ Chat history cleared.")
	return true
}

func (m *Model) cmdModels(_ []string) bool {
	fmt.Println("  Available models:")
	fmt.Println("    deepseek-v4-flash")
	fmt.Println("    claude-opus-4-8")
	fmt.Println("    gpt-4o")
	return true
}

func (m *Model) cmdQuit(_ []string) bool {
	fmt.Println("Bye!")
	m.running = false
	return false
}

func main() {
	New().Run()
}
