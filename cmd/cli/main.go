package main

import (
	wf "github.com/chuccp/go-web-frame"
	"github.com/chuccp/go-web-frame/config"
)

func main() {

	loadConfig, err := config.LoadConfig("./application.yml")
	if err != nil {
		return
	}
	builder := wf.NewBuilder(loadConfig)
	builder.Runner(&Command{})
	frame := builder.Build()
	err0 := frame.Start()
	if err0 != nil {
		return
	}
}
