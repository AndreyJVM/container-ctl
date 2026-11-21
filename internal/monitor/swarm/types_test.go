package swarm

import (
	"testing"
	"time"
)

func TestSwarmConstants(t *testing.T) {
	if swarmTimeout != 5*time.Second {
		t.Errorf("unexpected swarmTimeout: %v", swarmTimeout)
	}
	if defaultSwarmTail != 100 {
		t.Errorf("unexpected defaultSwarmTail: %d", defaultSwarmTail)
	}
	if swarmIDTruncateLen != 12 {
		t.Errorf("unexpected swarmIDTruncateLen: %d", swarmIDTruncateLen)
	}
	if swarmStatusActive != "Active" {
		t.Errorf("unexpected swarmStatusActive: %s", swarmStatusActive)
	}
	if swarmStatusInactive != "Inactive" {
		t.Errorf("unexpected swarmStatusInactive: %s", swarmStatusInactive)
	}
	if swarmTextInactive != "Docker Swarm is not active" {
		t.Errorf("unexpected swarmTextInactive: %s", swarmTextInactive)
	}
}

func TestSwarmTypes_Instantiation(t *testing.T) {
	node := SwarmNode{
		ID:           "node-12345678",
		Hostname:     "worker-1",
		Role:         "worker",
		State:        "ready",
		Availability: "active",
		IsLeader:     false,
		ActiveTasks:  2,
	}
	if node.ID != "node-12345678" || node.ActiveTasks != 2 {
		t.Errorf("unexpected SwarmNode: %+v", node)
	}

	service := SwarmService{
		ID:              "svc-12345678",
		Name:            "api-service",
		Image:           "api:v1",
		Mode:            "replicated",
		Replicas:        "3/3",
		RunningReplicas: 3,
		TargetReplicas:  3,
		Ports:           "8080:80/tcp",
	}
	if service.Name != "api-service" || service.RunningReplicas != 3 {
		t.Errorf("unexpected SwarmService: %+v", service)
	}

	summary := SwarmSummary{
		IsActive:      true,
		StatusDisplay: swarmStatusActive,
		NodeID:        "node-12345678",
		IsManager:     false,
		NodesCount:    1,
		ManagersCount: 0,
		Nodes:         []SwarmNode{node},
		Services:      []SwarmService{service},
		Text:          "Swarm Active",
	}
	if !summary.IsActive || len(summary.Nodes) != 1 || len(summary.Services) != 1 {
		t.Errorf("unexpected SwarmSummary: %+v", summary)
	}
}
