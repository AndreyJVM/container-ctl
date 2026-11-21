package swarm

import "time"

const (
	swarmTimeout        = 5 * time.Second
	defaultSwarmTail    = 100
	swarmIDTruncateLen  = 12
	swarmStatusActive   = "Active"
	swarmStatusInactive = "Inactive"
	swarmTextInactive   = "Docker Swarm is not active"
)

// SwarmNode represents a single cluster node in Docker Swarm.
type SwarmNode struct {
	ID           string
	Hostname     string
	Role         string
	State        string
	Availability string
	IsLeader     bool
	ActiveTasks  int
}

// SwarmService represents a deployed cluster service in Docker Swarm.
type SwarmService struct {
	ID              string
	Name            string
	Image           string
	Mode            string
	Replicas        string
	RunningReplicas int
	TargetReplicas  int
	Ports           string
}

// SwarmSummary represents high-level Swarm cluster status and topology.
type SwarmSummary struct {
	IsActive      bool
	StatusDisplay string
	NodeID        string
	IsManager     bool
	NodesCount    int
	ManagersCount int
	Nodes         []SwarmNode
	Services      []SwarmService
	Text          string
}
