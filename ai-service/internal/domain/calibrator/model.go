package calibrator

type Request struct {
	Prompt string `json:"prompt"`
}

type Response struct {
	Difficulty int    `json:"difficulty"`
	Reasoning  string `json:"reasoning"`
}
