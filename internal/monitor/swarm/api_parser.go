package swarm

import (
	"encoding/json"
	"fmt"
	"strings"
)

func truncateSwarmID(id string) string {
	if len(id) > swarmIDTruncateLen {
		return id[:swarmIDTruncateLen]
	}
	return id
}

func lookupTaskCount(taskMap map[string]int, fullID, shortID string) int {
	if taskMap == nil {
		return 0
	}
	if count, ok := taskMap[fullID]; ok {
		return count
	}
	return taskMap[shortID]
}

func cleanSwarmImage(rawImage string) string {
	return strings.Split(rawImage, "@")[0]
}

// parseSwarmInfo parses Docker Engine API /info response for the Swarm section.
func parseSwarmInfo(body []byte) (SwarmSummary, bool) {
	var infoData struct {
		Swarm struct {
			LocalNodeState   string `json:"LocalNodeState"`
			ControlAvailable bool   `json:"ControlAvailable"`
			NodeID           string `json:"NodeID"`
			Nodes            int    `json:"Nodes"`
			Managers         int    `json:"Managers"`
		} `json:"Swarm"`
	}

	if json.Unmarshal(body, &infoData) != nil || strings.ToLower(infoData.Swarm.LocalNodeState) != "active" {
		return SwarmSummary{}, false
	}

	return SwarmSummary{
		IsActive:      true,
		StatusDisplay: swarmStatusActive,
		NodeID:        truncateSwarmID(infoData.Swarm.NodeID),
		IsManager:     infoData.Swarm.ControlAvailable,
		NodesCount:    infoData.Swarm.Nodes,
		ManagersCount: infoData.Swarm.Managers,
	}, true
}

// parseSwarmTasks parses Docker Engine API /tasks response and returns active task counts grouped by serviceID and nodeID.
func parseSwarmTasks(body []byte) (map[string]int, map[string]int) {
	var rawTasks []struct {
		ID        string `json:"ID"`
		ServiceID string `json:"ServiceID"`
		NodeID    string `json:"NodeID"`
		Status    struct {
			State string `json:"State"`
		} `json:"Status"`
		DesiredState string `json:"DesiredState"`
	}

	svcTasks := make(map[string]int)
	nodeTasks := make(map[string]int)

	if json.Unmarshal(body, &rawTasks) != nil {
		return svcTasks, nodeTasks
	}

	for _, t := range rawTasks {
		if strings.ToLower(t.Status.State) == "running" {
			svcTasks[t.ServiceID]++
			nodeTasks[t.NodeID]++
		}
	}
	return svcTasks, nodeTasks
}

// parseSwarmNodes parses Docker Engine API /nodes response into a slice of SwarmNode models.
func parseSwarmNodes(body []byte, nodeTasks map[string]int) []SwarmNode {
	var rawNodes []struct {
		ID   string `json:"ID"`
		Spec struct {
			Role         string `json:"Role"`
			Availability string `json:"Availability"`
		} `json:"Spec"`
		Description struct {
			Hostname string `json:"Hostname"`
		} `json:"Description"`
		Status struct {
			State string `json:"State"`
		} `json:"Status"`
		ManagerStatus struct {
			Leader bool `json:"Leader"`
		} `json:"ManagerStatus"`
	}

	if json.Unmarshal(body, &rawNodes) != nil {
		return nil
	}

	nodes := make([]SwarmNode, 0, len(rawNodes))
	for _, n := range rawNodes {
		nid := truncateSwarmID(n.ID)
		availability := n.Spec.Availability
		if availability == "" {
			availability = "active"
		}

		nodes = append(nodes, SwarmNode{
			ID:           nid,
			Hostname:     n.Description.Hostname,
			Role:         n.Spec.Role,
			State:        n.Status.State,
			Availability: availability,
			IsLeader:     n.ManagerStatus.Leader,
			ActiveTasks:  lookupTaskCount(nodeTasks, n.ID, nid),
		})
	}
	return nodes
}

type swarmServiceRaw struct {
	ID   string `json:"ID"`
	Spec struct {
		Name         string `json:"Name"`
		TaskTemplate struct {
			ContainerSpec struct {
				Image string `json:"Image"`
			} `json:"ContainerSpec"`
		} `json:"TaskTemplate"`
		Mode struct {
			Replicated *struct {
				Replicas int `json:"Replicas"`
			} `json:"Replicated"`
			Global *struct{} `json:"Global"`
		} `json:"Mode"`
	} `json:"Spec"`
	Endpoint struct {
		Ports []struct {
			Protocol      string `json:"Protocol"`
			TargetPort    int    `json:"TargetPort"`
			PublishedPort int    `json:"PublishedPort"`
		} `json:"Ports"`
	} `json:"Endpoint"`
}

func formatSwarmServicePorts(ports []struct {
	Protocol      string `json:"Protocol"`
	TargetPort    int    `json:"TargetPort"`
	PublishedPort int    `json:"PublishedPort"`
}) string {
	if len(ports) == 0 {
		return ""
	}
	portList := make([]string, 0, len(ports))
	for _, p := range ports {
		if p.PublishedPort > 0 {
			portList = append(portList, fmt.Sprintf("%d:%d/%s", p.PublishedPort, p.TargetPort, p.Protocol))
		} else if p.TargetPort > 0 {
			portList = append(portList, fmt.Sprintf("%d/%s", p.TargetPort, p.Protocol))
		}
	}
	return strings.Join(portList, ", ")
}

// parseSwarmServices parses Docker Engine API /services response into a slice of SwarmService models.
func parseSwarmServices(body []byte, runningTasks map[string]int) []SwarmService {
	var rawSvcs []swarmServiceRaw
	if json.Unmarshal(body, &rawSvcs) != nil {
		return nil
	}

	svcs := make([]SwarmService, 0, len(rawSvcs))
	for _, s := range rawSvcs {
		sid := truncateSwarmID(s.ID)
		running := lookupTaskCount(runningTasks, s.ID, sid)

		mode := "replicated"
		targetReplicas := 0
		var rep string

		if s.Spec.Mode.Global != nil {
			mode = "global"
			targetReplicas = running
			rep = fmt.Sprintf("%d (global)", running)
		} else if s.Spec.Mode.Replicated != nil {
			targetReplicas = s.Spec.Mode.Replicated.Replicas
			rep = fmt.Sprintf("%d/%d", running, targetReplicas)
		} else {
			rep = fmt.Sprintf("%d", running)
		}

		svcs = append(svcs, SwarmService{
			ID:              sid,
			Name:            s.Spec.Name,
			Image:           cleanSwarmImage(s.Spec.TaskTemplate.ContainerSpec.Image),
			Mode:            mode,
			Replicas:        rep,
			RunningReplicas: running,
			TargetReplicas:  targetReplicas,
			Ports:           formatSwarmServicePorts(s.Endpoint.Ports),
		})
	}
	return svcs
}
