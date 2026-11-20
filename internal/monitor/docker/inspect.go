package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type socketInspectData struct {
	Name    string `json:"Name"`
	Created string `json:"Created"`
	State   struct {
		Status   string `json:"Status"`
		ExitCode int    `json:"ExitCode"`
		Health   *struct {
			Status        string `json:"Status"`
			FailingStreak int    `json:"FailingStreak"`
		} `json:"Health"`
	} `json:"State"`
	Config struct {
		Image string   `json:"Image"`
		Cmd   []string `json:"Cmd"`
		Env   []string `json:"Env"`
	} `json:"Config"`
	HostConfig struct {
		RestartPolicy struct {
			Name string `json:"Name"`
		} `json:"RestartPolicy"`
	} `json:"HostConfig"`
	NetworkSettings struct {
		IPAddress  string `json:"IPAddress"`
		Gateway    string `json:"Gateway"`
		MacAddress string `json:"MacAddress"`
		Networks   map[string]struct {
			IPAddress  string `json:"IPAddress"`
			Gateway    string `json:"Gateway"`
			MacAddress string `json:"MacAddress"`
		} `json:"Networks"`
	} `json:"NetworkSettings"`
	Mounts []struct {
		Source      string `json:"Source"`
		Destination string `json:"Destination"`
		Mode        string `json:"Mode"`
	} `json:"Mounts"`
}

func getContainerDetailViaSocket(sockPath, containerID string) (ContainerDetail, error) {
	var detail ContainerDetail
	detail.ID = containerID

	client := SocketClientFactory(sockPath)
	endpoint := fmt.Sprintf("http://localhost/containers/%s/json", url.PathEscape(containerID))
	resp, err := client.Get(endpoint)
	if err != nil {
		return detail, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return detail, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return detail, err
	}

	var data socketInspectData
	if err := json.Unmarshal(body, &data); err != nil {
		return detail, err
	}

	detail.Name = strings.TrimPrefix(data.Name, "/")
	detail.Image = data.Config.Image
	detail.Created = data.Created
	detail.State = data.State.Status
	detail.Status = fmt.Sprintf("%s (ExitCode: %d)", data.State.Status, data.State.ExitCode)
	if data.State.Health != nil && data.State.Health.Status != "" {
		detail.Health = ParseHealthStatus(data.State.Health.Status)
	} else {
		detail.Health = ParseHealthStatus(detail.Status)
	}
	detail.RestartPolicy = data.HostConfig.RestartPolicy.Name
	detail.IP = data.NetworkSettings.IPAddress
	detail.Gateway = data.NetworkSettings.Gateway
	detail.MACAddress = data.NetworkSettings.MacAddress
	if detail.IP == "" {
		for _, net := range data.NetworkSettings.Networks {
			if net.IPAddress != "" {
				detail.IP = net.IPAddress
				if detail.Gateway == "" {
					detail.Gateway = net.Gateway
				}
				if detail.MACAddress == "" {
					detail.MACAddress = net.MacAddress
				}
				break
			}
		}
	}
	detail.Command = strings.Join(data.Config.Cmd, " ")

	for _, m := range data.Mounts {
		detail.Mounts = append(detail.Mounts, fmt.Sprintf("%s -> %s (%s)", m.Source, m.Destination, m.Mode))
	}
	detail.Env = data.Config.Env

	if detail.State == "running" {
		stats := fetchContainerStats(client, containerID)
		detail.CPUPerc = stats.CPUPerc
		detail.MemUsageMB = stats.MemUsageMB
		detail.MemPerc = stats.MemPerc
		detail.NetRxBytes = stats.NetRxBytes
		detail.NetTxBytes = stats.NetTxBytes
		detail.BlockReadBytes = stats.BlockReadBytes
		detail.BlockWriteBytes = stats.BlockWriteBytes
	}

	return detail, nil
}

func getContainerDetailViaCLI(containerID string) (ContainerDetail, error) {
	var detail ContainerDetail
	detail.ID = containerID

	ctx, cancel := context.WithTimeout(context.Background(), defaultDockerTimeout)
	defer cancel()
	out, err := DockerCLIRunner(ctx, "inspect", containerID)
	if err != nil {
		return detail, fmt.Errorf("container not found: %s", containerID)
	}

	var raw []map[string]interface{}
	if err := json.Unmarshal(out, &raw); err != nil || len(raw) == 0 {
		return detail, fmt.Errorf("container not found: %s", containerID)
	}

	first := raw[0]
	if name, ok := first["Name"].(string); ok {
		detail.Name = strings.TrimPrefix(name, "/")
	}
	if state, ok := first["State"].(map[string]interface{}); ok {
		if status, ok := state["Status"].(string); ok {
			detail.State = status
			detail.Status = status
		}
		if health, ok := state["Health"].(map[string]interface{}); ok {
			if hStatus, ok := health["Status"].(string); ok {
				detail.Health = ParseHealthStatus(hStatus)
			}
		}
	}
	if hostConfig, ok := first["HostConfig"].(map[string]interface{}); ok {
		if rp, ok := hostConfig["RestartPolicy"].(map[string]interface{}); ok {
			if name, ok := rp["Name"].(string); ok {
				detail.RestartPolicy = name
			}
		}
	}
	if detail.IP == "" {
		if netSettings, ok := first["NetworkSettings"].(map[string]interface{}); ok {
			if nets, ok := netSettings["Networks"].(map[string]interface{}); ok {
				for _, n := range nets {
					if nMap, ok := n.(map[string]interface{}); ok {
						if ip, ok := nMap["IPAddress"].(string); ok && ip != "" {
							detail.IP = ip
							if gw, ok := nMap["Gateway"].(string); ok && detail.Gateway == "" {
								detail.Gateway = gw
							}
							if mac, ok := nMap["MacAddress"].(string); ok && detail.MACAddress == "" {
								detail.MACAddress = mac
							}
							break
						}
					}
				}
			}
		}
	}
	if detail.Health == "" {
		detail.Health = ParseHealthStatus(detail.Status)
	}
	if created, ok := first["Created"].(string); ok {
		detail.Created = created
	}
	if config, ok := first["Config"].(map[string]interface{}); ok {
		if img, ok := config["Image"].(string); ok {
			detail.Image = img
		}
		if cmd, ok := config["Cmd"].([]interface{}); ok {
			var cmdParts []string
			for _, part := range cmd {
				if s, ok := part.(string); ok {
					cmdParts = append(cmdParts, s)
				}
			}
			detail.Command = strings.Join(cmdParts, " ")
		}
		if env, ok := config["Env"].([]interface{}); ok {
			for _, e := range env {
				if s, ok := e.(string); ok {
					detail.Env = append(detail.Env, s)
				}
			}
		}
	}
	if mounts, ok := first["Mounts"].([]interface{}); ok {
		for _, m := range mounts {
			if mMap, ok := m.(map[string]interface{}); ok {
				src, _ := mMap["Source"].(string)
				dst, _ := mMap["Destination"].(string)
				mode, _ := mMap["Mode"].(string)
				if src != "" && dst != "" {
					detail.Mounts = append(detail.Mounts, fmt.Sprintf("%s -> %s (%s)", src, dst, mode))
				}
			}
		}
	}
	return detail, nil
}

// GetContainerDetail returns detailed inspection data for a container.
func GetContainerDetail(containerID string) (ContainerDetail, error) {
	var detail ContainerDetail
	if strings.TrimSpace(containerID) == "" {
		return detail, fmt.Errorf("container ID cannot be empty")
	}
	detail.ID = containerID

	sockPath := DockerSocketFinder()
	if sockPath != "" {
		if d, err := getContainerDetailViaSocket(sockPath, containerID); err == nil {
			return d, nil
		}
	}

	return getContainerDetailViaCLI(containerID)
}

// GetNetworkInspectJSON returns formatted JSON inspection for a Docker network.
func GetNetworkInspectJSON(idOrName string) (string, error) {
	if strings.TrimSpace(idOrName) == "" {
		return "", fmt.Errorf("network identifier cannot be empty")
	}

	if sockPath := DockerSocketFinder(); sockPath != "" {
		client := SocketClientFactory(sockPath)
		endpoint := fmt.Sprintf("http://localhost/networks/%s", url.PathEscape(idOrName))
		resp, err := client.Get(endpoint)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				if err == nil && len(body) > 0 {
					var pretty bytes.Buffer
					if err := json.Indent(&pretty, body, "", "  "); err == nil {
						return pretty.String(), nil
					}
					return string(body), nil
				}
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultDockerTimeout)
	defer cancel()
	out, err := DockerCLIRunner(ctx, "network", "inspect", idOrName)
	if err == nil && len(out) > 0 {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, out, "", "  "); err == nil {
			return pretty.String(), nil
		}
		return string(out), nil
	}

	return "", fmt.Errorf("network not found: %s", idOrName)
}

// GetVolumeInspectJSON returns formatted JSON inspection for a Docker volume.
func GetVolumeInspectJSON(name string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("volume name cannot be empty")
	}

	if sockPath := DockerSocketFinder(); sockPath != "" {
		client := SocketClientFactory(sockPath)
		endpoint := fmt.Sprintf("http://localhost/volumes/%s", url.PathEscape(name))
		resp, err := client.Get(endpoint)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				if err == nil && len(body) > 0 {
					var pretty bytes.Buffer
					if err := json.Indent(&pretty, body, "", "  "); err == nil {
						return pretty.String(), nil
					}
					return string(body), nil
				}
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultDockerTimeout)
	defer cancel()
	out, err := DockerCLIRunner(ctx, "volume", "inspect", name)
	if err == nil && len(out) > 0 {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, out, "", "  "); err == nil {
			return pretty.String(), nil
		}
		return string(out), nil
	}

	return "", fmt.Errorf("volume not found: %s", name)
}
