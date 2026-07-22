package main

import wf "github.com/chuccp/go-web-frame"

//	func main() {
//		p := tea.NewProgram(NewModel(), tea.WithInput(os.Stdin))
//		if _, err := p.Run(); err != nil {
//			fmt.Printf("⚠ TTY not available, using simple mode.\n\n")
//			RunSimpleREPL()
//		}
//	}
func main() {
	builder := wf.NewBuilder()
	builder.Runner(&Command{})
	frame := builder.Build()
	err := frame.Start()
	if err != nil {
		return
	}
}
