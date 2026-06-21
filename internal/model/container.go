package model

// ContainerCLIStatus describes whether Apple's container CLI is usable.
type ContainerCLIStatus struct {
	Available bool   `json:"available"`
	Supported bool   `json:"supported"`
	Path      string `json:"path,omitempty"`
	Version   string `json:"version,omitempty"`
	System    string `json:"system,omitempty"`
	OSVersion string `json:"osVersion,omitempty"`
	Error     string `json:"error,omitempty"`
}

// HostResourceInfo describes local machine resources available for run presets.
type HostResourceInfo struct {
	CPUCores    int    `json:"cpuCores"`
	MemoryBytes uint64 `json:"memoryBytes"`
	MemoryMiB   int    `json:"memoryMiB"`
}

// ContainerInfo is a compact, UI-friendly view of an Apple container.
type ContainerInfo struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Image     string   `json:"image"`
	Status    string   `json:"status"`
	IP        string   `json:"ip,omitempty"`
	CPUs      string   `json:"cpus,omitempty"`
	Memory    string   `json:"memory,omitempty"`
	OS        string   `json:"os,omitempty"`
	Arch      string   `json:"arch,omitempty"`
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

// NetworkInfo is a compact, UI-friendly view of an Apple container network.
type NetworkInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Mode        string   `json:"mode,omitempty"`
	Plugin      string   `json:"plugin,omitempty"`
	IPv4Subnet  string   `json:"ipv4Subnet,omitempty"`
	IPv4Gateway string   `json:"ipv4Gateway,omitempty"`
	IPv6Subnet  string   `json:"ipv6Subnet,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	Options     []string `json:"options,omitempty"`
	CreatedAt   string   `json:"createdAt,omitempty"`
}

// CreateNetworkOptions captures the fields supported by `container network create`.
type CreateNetworkOptions struct {
	Name     string   `json:"name"`
	Internal bool     `json:"internal,omitempty"`
	Labels   []string `json:"labels,omitempty"`
	Options  []string `json:"options,omitempty"`
	Plugin   string   `json:"plugin,omitempty"`
	Subnet   string   `json:"subnet,omitempty"`
	SubnetV6 string   `json:"subnetV6,omitempty"`
}

// RunContainerOptions captures the first-pass fields Void exposes in the guided run flow.
type RunContainerOptions struct {
	Image      string   `json:"image"`
	Name       string   `json:"name,omitempty"`
	Ports      []string `json:"ports,omitempty"`
	Envs       []string `json:"envs,omitempty"`
	Volumes    []string `json:"volumes,omitempty"`
	Command    string   `json:"command,omitempty"`
	Detach     bool     `json:"detach"`
	Remove     bool     `json:"remove"`
	Network    string   `json:"network,omitempty"`
	DNS        []string `json:"dns,omitempty"`
	NoDNS      bool     `json:"noDns,omitempty"`
	Platform   string   `json:"platform,omitempty"`
	CPUs       string   `json:"cpus,omitempty"`
	Memory     string   `json:"memory,omitempty"`
	Workdir    string   `json:"workdir,omitempty"`
	User       string   `json:"user,omitempty"`
	Entrypoint string   `json:"entrypoint,omitempty"`
	Labels     []string `json:"labels,omitempty"`
	Init       bool     `json:"init,omitempty"`
	ReadOnly   bool     `json:"readOnly,omitempty"`
	SSH        bool     `json:"ssh,omitempty"`
}

// ContainerCommandResult returns command output plus the exact CLI command Void built.
type ContainerCommandResult struct {
	Command string `json:"command"`
	Output  string `json:"output"`
}
