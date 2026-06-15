package nodes

// Node types
const (
	TypeStart     = "start"
	TypeEnd       = "end"
	TypeLLM       = "llm"
	TypeUserInput = "user_input"
	TypeForEach   = "for_each"
	TypeSplit     = "split"
	TypeTransform = "transform"
	TypeCondition = "condition"
	TypeScript    = "script"
	TypeIterator  = "iterator"
	TypeLoop      = "loop"
	TypeSwitch    = "switch"
	TypeExecute   = "execute"
	TypeImageGen  = "image_gen"
	TypeAudioGen  = "audio_gen"
	TypeVideoGen  = "video_gen"
)

// Node output data keys
const (
	KeyOutput = "output"
	KeyPrompt = "prompt"
)

// DefaultModel -- empty means resolve from system default at runtime
const DefaultModel = ""
