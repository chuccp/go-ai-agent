package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chuccp/go-web-frame/core"
)

// Update(Msg) (Model, Cmd)

type Command struct {
	core.IRunner
	ctx   *core.Context
	model *Model
}

func (receiver *Command) Init(ctx *core.Context) error {
	receiver.ctx = ctx
	receiver.model = NewModel(receiver.ctx)
	return nil
}
func (receiver *Command) Run() error {
	p := tea.NewProgram(receiver.model)
	if _, err := p.Run(); err != nil {
		fmt.Printf("⚠ TTY not available, using simple mode.\n\n")
		RunSimpleREPL()
	}
	return nil
}
