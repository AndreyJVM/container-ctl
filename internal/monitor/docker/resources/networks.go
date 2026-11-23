package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/AndreyJVM/container-ctl/internal/monitor/docker/engine"
	"github.com/AndreyJVM/container-ctl/internal/monitor/docker/types"
)

// GetDockerNetworks returns Docker networks without changing daemon state.
func GetDockerNetworks() ([]types.DockerNetwork, error) {
	if sockPath := engine.DockerSocketFinder(); sockPath != "" {
		var raw []struct {
			ID         string `json:"Id"`
			Name       string `json:"Name"`
			Created    string `json:"Created"`
			Driver     string `json:"Driver"`
			Scope      string `json:"Scope"`
			Internal   bool   `json:"Internal"`
			Attachable bool   `json:"Attachable"`
			Ingress    bool   `json:"Ingress"`
			EnableIPv6 bool   `json:"EnableIPv6"`
			Containers map[string]struct {
				Name        string `json:"Name"`
				EndpointID  string `json:"EndpointID"`
				MacAddress  string `json:"MacAddress"`
				IPv4Address string `json:"IPv4Address"`
				IPv6Address string `json:"IPv6Address"`
			} `json:"Containers"`
			Labels  map[string]string `json:"Labels"`
			Options map[string]string `json:"Options"`
			IPAM    struct {
				Driver string `json:"Driver"`
				Config []struct {
					Subnet  string `json:"Subnet"`
					Gateway string `json:"Gateway"`
					IPRange string `json:"IPRange"`
				} `json:"Config"`
			} `json:"IPAM"`
		}
		if err := engine.FetchDockerResourceJSON(sockPath, "/networks", &raw); err == nil {
			networks := make([]types.DockerNetwork, 0, len(raw))
			for _, item := range raw {
				subnet, gateway, ipRange := "", "", ""
				if len(item.IPAM.Config) > 0 {
					subnet = item.IPAM.Config[0].Subnet
					gateway = item.IPAM.Config[0].Gateway
					ipRange = item.IPAM.Config[0].IPRange
				}
				var connected []string
				var containerDetails []types.NetworkContainerInfo
				for cid, cInfo := range item.Containers {
					cName := cInfo.Name
					if cName == "" {
						continue
					}
					ip := cInfo.IPv4Address
					displayIP := ip
					if idx := strings.IndexByte(displayIP, '/'); idx != -1 {
						displayIP = displayIP[:idx]
					}
					if displayIP != "" {
						connected = append(connected, fmt.Sprintf("%s (%s)", cName, displayIP))
					} else {
						connected = append(connected, cName)
					}
					containerDetails = append(containerDetails, types.NetworkContainerInfo{
						ID:          TruncateResourceID(cid),
						Name:        cName,
						IPv4Address: cInfo.IPv4Address,
						IPv6Address: cInfo.IPv6Address,
						MacAddress:  cInfo.MacAddress,
						EndpointID:  TruncateResourceID(cInfo.EndpointID),
					})
				}
				if len(connected) > 0 {
					sort.Strings(connected)
				}
				if len(containerDetails) > 0 {
					sort.Slice(containerDetails, func(i, j int) bool {
						return containerDetails[i].Name < containerDetails[j].Name
					})
				}

				networks = append(networks, types.DockerNetwork{
					ID:                  TruncateResourceID(item.ID),
					Name:                item.Name,
					Driver:              item.Driver,
					Scope:               item.Scope,
					Internal:            item.Internal,
					Attachable:          item.Attachable,
					Ingress:             item.Ingress,
					EnableIPv6:          item.EnableIPv6,
					Containers:          len(item.Containers),
					ConnectedContainers: connected,
					ContainerDetails:    containerDetails,
					Subnet:              subnet,
					Gateway:             gateway,
					IPRange:             ipRange,
					IPAMDriver:          item.IPAM.Driver,
					ComposeProject:      item.Labels[types.ComposeProjectLabel],
					Created:             item.Created,
					Labels:              item.Labels,
					Options:             item.Options,
				})
			}
			return networks, nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), engine.DefaultDockerTimeout)
	defer cancel()
	format := "{{.ID}}|{{.Name}}|{{.Driver}}|{{.Scope}}|{{.Internal}}|{{.Labels}}"
	out, err := engine.DockerCLIRunner(ctx, "network", "ls", "--no-trunc", "--format", format)
	if err != nil {
		return nil, fmt.Errorf("failed to list Docker networks: %w", err)
	}

	var networks []types.DockerNetwork
	for _, line := range NonEmptyLines(string(out)) {
		parts := strings.SplitN(line, "|", 6)
		if len(parts) != 6 {
			continue
		}
		networks = append(networks, types.DockerNetwork{
			ID:             TruncateResourceID(parts[0]),
			Name:           parts[1],
			Driver:         parts[2],
			Scope:          parts[3],
			Internal:       ParseBool(parts[4]),
			ComposeProject: LabelValue(parts[5], types.ComposeProjectLabel),
		})
	}
	return networks, nil
}

// GetNetworkInspectJSON returns formatted JSON inspection for a Docker network.
func GetNetworkInspectJSON(idOrName string) (string, error) {
	idOrName = strings.TrimSpace(idOrName)
	if idOrName == "" {
		return "", fmt.Errorf("network identifier cannot be empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), engine.DefaultDockerTimeout)
	defer cancel()

	if sockPath := engine.DockerSocketFinder(); sockPath != "" {
		client := engine.SocketClientFactory(sockPath)
		endpoint := fmt.Sprintf("http://localhost/networks/%s", url.PathEscape(idOrName))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err == nil {
			resp, err := client.Do(req)
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
	}

	out, err := engine.DockerCLIRunner(ctx, "network", "inspect", idOrName)
	if err == nil && len(out) > 0 {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, out, "", "  "); err == nil {
			return pretty.String(), nil
		}
		return string(out), nil
	}

	return "", fmt.Errorf("network not found: %s", idOrName)
}

// RemoveDockerNetwork removes a Docker network by ID or name.
func RemoveDockerNetwork(idOrName string) error {
	idOrName = strings.TrimSpace(idOrName)
	if idOrName == "" {
		return fmt.Errorf("network identifier cannot be empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), engine.DefaultDockerTimeout)
	defer cancel()

	if sockPath := engine.DockerSocketFinder(); sockPath != "" {
		client := engine.SocketClientFactory(sockPath)
		endpoint := fmt.Sprintf("http://localhost/networks/%s", url.PathEscape(idOrName))
		req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
		if err == nil {
			resp, err := client.Do(req)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
					return nil
				}
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
			}
		}
	}

	out, err := engine.DockerCLIRunner(ctx, "network", "rm", idOrName)
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return nil
}
