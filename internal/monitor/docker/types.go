package docker

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
	composeProjectLabel = "com.docker.compose.project"
	composeServiceLabel = "com.docker.compose.service"

	// Defaults, limits, and timeouts
	maxConcurrentStatsFetch = 10
	defaultDockerTimeout    = 5 * time.Second
	containerTimeFormat     = "2006-01-02 15:04:05"
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
}
