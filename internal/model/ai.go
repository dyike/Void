package model

// AskResult holds the completed AI response.
type AskResult struct {
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

// AIEvent represents a unified streaming event sent to the frontend.
type AIEvent struct {
	Type      string `json:"type"`
	Content   string `json:"content,omitempty"`
	Name      string `json:"name,omitempty"`
	Args      string `json:"args,omitempty"`
	Result    string `json:"result,omitempty"`
	IsError   bool   `json:"isError,omitempty"`
	Turn      int    `json:"turn,omitempty"`
	Seq       int64  `json:"seq"`
	Timestamp int64  `json:"timestamp"`
}
