package drafter

type Request struct {
	Topic string `json:"topic"`
	Tier  string `json:"tier"`
	Mode  string `json:"mode"`
	Seed  string `json:"seed"`
}

type BulkRequest struct {
	Topic string `json:"topic"`
	Tier  string `json:"tier"`
	Mode  string `json:"mode"`
	Count int    `json:"count"`
}

type Draft struct {
	Scenario       string   `json:"scenario"`
	Prompt         string   `json:"prompt"`
	Options        []string `json:"options"`
	CorrectAnswers []int    `json:"correctAnswers"`
	Rationale      string   `json:"rationale"`
}
