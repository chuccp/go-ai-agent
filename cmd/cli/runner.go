package main

import "github.com/chuccp/go-web-frame/core"

type CommandRunner struct {
	core.IRunner
}

func (receiver *CommandRunner) Init(ctx *core.Context) error {

	return nil
}

func (receiver *CommandRunner) Run() error {
	return nil
}
