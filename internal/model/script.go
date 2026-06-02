package model

// ScriptMode controls how the output is displayed.
type ScriptMode string

const (
	ModeSilent     ScriptMode = "silent"     // run and ignore output
	ModeCompact    ScriptMode = "compact"    // show last line in HUD
	ModeFullOutput ScriptMode = "fullOutput" // show full output in results view
	ModeInline     ScriptMode = "inline"     // replaces the command item with output
)

// Argument defines a script input parameter.
type Argument struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "text"|"password"|"dropdown"
	Placeholder string `json:"placeholder"`
	Required    bool   `json:"required"`
}

// ScriptCommand is a discovered shell script with metadata.
type ScriptCommand struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Mode        ScriptMode `json:"mode"`
	Icon        string     `json:"icon,omitempty"`
	ScriptPath  string     `json:"scriptPath"`
	Arguments   []Argument `json:"arguments,omitempty"`
	RefreshTime string     `json:"refreshTime,omitempty"`
	PackageName string     `json:"packageName,omitempty"`
}

// ExecutionResult holds the output of a script run.
type ExecutionResult struct {
	CommandID string     `json:"commandId"`
	Title     string     `json:"title"`
	Mode      ScriptMode `json:"mode"`
	ExitCode  int        `json:"exitCode"`
	Stdout    string     `json:"stdout"`
	Stderr    string     `json:"stderr"`
	Output    string     `json:"output"`
	FirstLine string     `json:"firstLine"`
	LastLine  string     `json:"lastLine"`
}
