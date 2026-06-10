package types

// Category values for ai_models.category column.
const (
	CategoryLLM       = "llm"
	CategoryImage     = "image"
	CategoryVoice     = "voice"
	CategoryVideo     = "video"
	CategoryEmbedding = "embedding"
	CategoryOCR       = "ocr"
	CategorySTT       = "stt"   // speech-to-text
	CategoryOther     = "other" // 水印消除、去背景、超分辨率等小众模型，通过 InputTypes/OutputTypes 定义行为
)

// Modality constants for input/output type declarations.
const (
	ModalityText  = "text"
	ModalityImage = "image"
	ModalityAudio = "audio"
	ModalityVideo = "video"
)

// ProviderInfo describes a provider's default model and base URL.
type ProviderInfo struct {
	Model   string `json:"model"`
	BaseURL string `json:"baseUrl"`
}
