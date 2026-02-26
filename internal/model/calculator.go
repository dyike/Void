package model

// CalcResult holds the evaluation output.
type CalcResult struct {
	Input       string `json:"input"`
	InputLabel  string `json:"inputLabel,omitempty"`
	Result      string `json:"result"`
	ResultLabel string `json:"resultLabel,omitempty"`
	IsValid     bool   `json:"isValid"`
}
