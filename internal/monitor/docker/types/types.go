package types

import "time"

const (
	// Docker daemon statuses
	DockerStatusRunning      = "running"
	DockerStatusStopped      = "stopped"
	DockerStatusNotInstalled = "not_installed"

	// Localized status displays
	DockerStatusDisplayRunning      = "Running"
	DockerStatusDisplayStopped      = "Stopped"
	DockerStatusDisplayNotInstalled = "Not installed"

	// Healthcheck statuses
	HealthStatusHealthy   = "healthy"
	HealthStatusUnhealthy = "unhealthy"
	HealthStatusStarting  = "starting"

	// Docker compose label names
	ComposeProjectLabel = "com.docker.compose.project"
	ComposeServiceLabel = "com.docker.compose.service"

	// Defaults, limits, and timeouts
	MaxConcurrentStatsFetch = 10
	DefaultDockerTimeout    = 5 * time.Second
	ContainerTimeFormat     = "2006-01-02 15:04:05"
)

// ContainerItem represents a single container's high-level status and metrics.
type ContainerItem struct {
	ID              string
	Name            string
	Image           string
	State           string
	Status          string
	Health          string
	Ports           string
	ComposeProject  string
	ComposeService  string
	Labels          map[string]string
	Mounts          []string
	IPAddress       string
	NetworkName     string
	Created         string
	StartedAt       string
	RestartPolicy   string
	CPUPerc         float64
	MemUsageMB      float64
	MemLimitMB      float64
	MemPerc         float64
	NetRxBytes      uint64
	NetTxBytes      uint64
	BlockReadBytes  uint64
	BlockWriteBytes uint64
	ErrorSnippet    string
}

// ContainerDetail represents detailed inspection data for a single container.
type ContainerDetail struct {
	ID              string
	Name            string
	Image           string
	Created         string
	State           string
	Status          string
	Health          string
	IP              string
	Gateway         string
	MACAddress      string
	RestartPolicy   string
	Command         string
	Mounts          []string
	Env             []string
	CPUPerc         float64
	MemUsageMB      float64
	MemPerc         float64
	NetRxBytes      uint64
	NetTxBytes      uint64
	BlockReadBytes  uint64
	BlockWriteBytes uint64
}

// DockerSummary provides an overview of the Docker daemon state and containers.
type DockerSummary struct {
	Status            string // "running", "stopped", "not_installed"
	StatusDisplay     string
	RunningContainers int
	StoppedContainers int
	TotalContainers   int
	ImagesCount       int
	Version           string
	Containers        []ContainerItem
	IsPodman          bool
	IsRootless        bool
	SocketPath        string
}

// DockerDiskItem represents disk space information for a single Docker resource category.
type DockerDiskItem struct {
	TotalCount  int
	ActiveCount int
	TotalSize   int64
	Reclaimable int64
}

// DockerDiskUsage describes the total disk space usage across Docker images, containers, volumes, and build cache.
type DockerDiskUsage struct {
	Images     DockerDiskItem
	Containers DockerDiskItem
	Volumes    DockerDiskItem
	BuildCache DockerDiskItem
	TotalSize  int64
	TotalRecl  int64
}

// DockerImage represents a local Docker image.
type DockerImage struct {
	ID         string
	FullID     string
	Repository string
	Tag        string
	AllTags    []string
	Size       int64
	Created    string
	Containers int
	Digest     string
	InUse      bool
	Dangling   bool
}

// DockerVolume represents a Docker storage volume.
type DockerVolume struct {
	Name           string
	Driver         string
	Scope          string
	Mountpoint     string
	Size           int64
	RefCount       int
	Containers     []string
	ComposeProject string
}

// DockerNetwork represents a Docker virtual network.
type DockerNetwork struct {
	ID                  string
	Name                string
	Driver              string
	Scope               string
	Internal            bool
	Attachable          bool
	Containers          int
	ConnectedContainers []string
	Subnet              string
	Gateway             string
	ComposeProject      string
}

// ExecOptions specifies options for launching an interactive session in a container.
type ExecOptions struct {
	Mode      string // "auto", "sh", "zsh", "root", "attach", "custom"
	User      string // optional user (e.g., "0" or "root")
	CustomCmd string // optional custom command when Mode == "custom"
	Terminal  string // optional TERM value for embedded terminal emulation
}

// ContainerStats holds aggregated container resource usage metrics.
type ContainerStats struct {
	CPUPerc         float64
	MemUsageMB      float64
	MemLimitMB      float64
	MemPerc         float64
	NetRxBytes      uint64
	NetTxBytes      uint64
	BlockReadBytes  uint64
	BlockWriteBytes uint64
}

// ContainerStatsRaw reflects the JSON response from Docker's /stats endpoint.
type ContainerStatsRaw struct {
	CPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
		OnlineCPUs     int    `json:"online_cpus"`
	} `json:"cpu_stats"`
	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage uint64 `json:"usage"`
		Limit uint64 `json:"limit"`
		Stats struct {
			InactiveFile      uint64 `json:"inactive_file"`
			TotalInactiveFile uint64 `json:"total_inactive_file"`
		} `json:"stats"`
	} `json:"memory_stats"`
	Networks map[string]struct {
		RxBytes uint64 `json:"rx_bytes"`
		TxBytes uint64 `json:"tx_bytes"`
	} `json:"networks"`
	Network struct {
		RxBytes uint64 `json:"rx_bytes"`
		TxBytes uint64 `json:"tx_bytes"`
	} `json:"network"`
	BlkioStats struct {
		IOServiceBytesRecursive []struct {
			Major uint64 `json:"major"`
			Minor uint64 `json:"minor"`
			Op    string `json:"op"`
			Value uint64 `json:"value"`
		} `json:"io_service_bytes_recursive"`
	} `json:"blkio_stats"`
}
