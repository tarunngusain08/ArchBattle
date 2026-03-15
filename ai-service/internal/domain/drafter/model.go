package drafter

type Request struct {
	Topic string `json:"topic"`
	Tier  string `json:"tier"`
	Mode  string `json:"mode"`
	Seed  string `json:"seed"`
}

type Draft struct {
	Prompt         string   `json:"prompt"`
	Options        []string `json:"options"`
	CorrectAnswers []int    `json:"correctAnswers"`
	Rationale      string   `json:"rationale"`
}
