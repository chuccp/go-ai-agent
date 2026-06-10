package nodes

// 节点类型
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
	TypeImageGen  = "image_gen"
	TypeAudioGen  = "audio_gen"
	TypeVideoGen  = "video_gen"
)

// 节点输出数据键
const (
	KeyOutput = "output"
	KeyPrompt = "prompt"
)

// 默认模型 — empty means resolve from system default at runtime
const DefaultModel = ""
