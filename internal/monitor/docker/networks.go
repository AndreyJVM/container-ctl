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

// DockerNetwork describes a Docker network shown in the dashboard.
type DockerNetwork struct {
	ID             string
	Name           string
	Driver         string
	Scope          string
	Internal       bool
	Attachable     bool
	Containers     int
	Subnet         string
	Gateway        string
	ComposeProject string
}

// GetDockerNetworks returns Docker networks without changing daemon state.
func GetDockerNetworks() ([]DockerNetwork, error) {
	if sockPath := DockerSocketFinder(); sockPath != "" {
		var raw []struct {
			ID         string                 `json:"Id"`
			Name       string                 `json:"Name"`
			Driver     string                 `json:"Driver"`
			Scope      string                 `json:"Scope"`
			Internal   bool                   `json:"Internal"`
			Attachable bool                   `json:"Attachable"`
			Containers map[string]interface{} `json:"Containers"`
			Labels     map[string]string      `json:"Labels"`
			IPAM       struct {
				Config []struct {
					Subnet  string `json:"Subnet"`
					Gateway string `json:"Gateway"`
				} `json:"Config"`
			} `json:"IPAM"`
		}
		if err := fetchDockerResourceJSON(sockPath, "/networks", &raw); err == nil {
			networks := make([]DockerNetwork, 0, len(raw))
			for _, item := range raw {
				subnet, gateway := "", ""
				if len(item.IPAM.Config) > 0 {
					subnet = item.IPAM.Config[0].Subnet
					gateway = item.IPAM.Config[0].Gateway
				}
				networks = append(networks, DockerNetwork{
					ID:             truncateResourceID(item.ID),
					Name:           item.Name,
					Driver:         item.Driver,
					Scope:          item.Scope,
					Internal:       item.Internal,
					Attachable:     item.Attachable,
					Containers:     len(item.Containers),
					Subnet:         subnet,
					Gateway:        gateway,
					ComposeProject: item.Labels[composeProjectLabel],
				})
			}
			return networks, nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultDockerTimeout)
	defer cancel()
	format := "{{.ID}}|{{.Name}}|{{.Driver}}|{{.Scope}}|{{.Internal}}|{{.Labels}}"
	out, err := DockerCLIRunner(ctx, "network", "ls", "--no-trunc", "--format", format)
	if err != nil {
		return nil, fmt.Errorf("failed to list Docker networks: %w", err)
	}

	var networks []DockerNetwork
	for _, line := range nonEmptyLines(string(out)) {
		parts := strings.SplitN(line, "|", 6)
		if len(parts) != 6 {
			continue
		}
		networks = append(networks, DockerNetwork{
			ID:             truncateResourceID(parts[0]),
			Name:           parts[1],
			Driver:         parts[2],
			Scope:          parts[3],
			Internal:       parseBool(parts[4]),
			ComposeProject: labelValue(parts[5], composeProjectLabel),
		})
	}
	return networks, nil
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
