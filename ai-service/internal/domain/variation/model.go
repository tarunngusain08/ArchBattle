package variation

type Request struct {
	BasePrompt string `json:"basePrompt"`
	Count      int    `json:"count"`
}

type Response struct {
	Variations []string `json:"variations"`
}
