package model

// ContainerCLIStatus describes whether Apple's container CLI is usable.
type ContainerCLIStatus struct {
	Available bool   `json:"available"`
	Path      string `json:"path,omitempty"`
	Version   string `json:"version,omitempty"`
	System    string `json:"system,omitempty"`
	Error     string `json:"error,omitempty"`
}

// ContainerInfo is a compact, UI-friendly view of an Apple container.
type ContainerInfo struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Image     string   `json:"image"`
	Status    string   `json:"status"`
	Ports     []string `json:"ports"`
	CreatedAt string   `json:"createdAt,omitempty"`
	Age       string   `json:"age,omitempty"`
}

// ImageInfo is a compact, UI-friendly view of a local Apple container image.
type ImageInfo struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	Tag       string `json:"tag,omitempty"`
	Digest    string `json:"digest,omitempty"`
	Size      string `json:"size,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}

// RunContainerOptions captures the first-pass fields Void exposes in the guided run flow.
type RunContainerOptions struct {
	Image   string   `json:"image"`
	Name    string   `json:"name,omitempty"`
	Ports   []string `json:"ports,omitempty"`
	Envs    []string `json:"envs,omitempty"`
	Volumes []string `json:"volumes,omitempty"`
	Command string   `json:"command,omitempty"`
	Detach  bool     `json:"detach"`
	Remove  bool     `json:"remove"`
}

// ContainerCommandResult returns command output plus the exact CLI command Void built.
type ContainerCommandResult struct {
	Command string `json:"command"`
	Output  string `json:"output"`
}
