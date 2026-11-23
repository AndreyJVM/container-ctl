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

// GetDockerVolumes returns Docker volumes without changing daemon state.
func GetDockerVolumes() ([]types.DockerVolume, error) {
	if sockPath := engine.DockerSocketFinder(); sockPath != "" {
		var raw struct {
			Volumes []struct {
				Name       string            `json:"Name"`
				Driver     string            `json:"Driver"`
				Scope      string            `json:"Scope"`
				Mountpoint string            `json:"Mountpoint"`
				Labels     map[string]string `json:"Labels"`
				UsageData  *struct {
					Size     int64 `json:"Size"`
					RefCount int   `json:"RefCount"`
				} `json:"UsageData"`
			} `json:"Volumes"`
		}
		if err := engine.FetchDockerResourceJSON(sockPath, "/volumes", &raw); err == nil {
			volumes := make([]types.DockerVolume, 0, len(raw.Volumes))
			for _, item := range raw.Volumes {
				size, refCount := int64(-1), -1
				if item.UsageData != nil {
					size = item.UsageData.Size
					refCount = item.UsageData.RefCount
				}
				volumes = append(volumes, types.DockerVolume{
					Name:           item.Name,
					Driver:         item.Driver,
					Scope:          item.Scope,
					Mountpoint:     item.Mountpoint,
					Size:           size,
					RefCount:       refCount,
					ComposeProject: item.Labels[types.ComposeProjectLabel],
				})
			}
			return volumes, nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), engine.DefaultDockerTimeout)
	defer cancel()
	format := "{{.Name}}|{{.Driver}}|{{.Scope}}|{{.Mountpoint}}|{{.Labels}}"
	out, err := engine.DockerCLIRunner(ctx, "volume", "ls", "--format", format)
	if err != nil {
		return nil, fmt.Errorf("failed to list Docker volumes: %w", err)
	}

	var volumes []types.DockerVolume
	for _, line := range NonEmptyLines(string(out)) {
		parts := strings.SplitN(line, "|", 5)
		if len(parts) != 5 {
			continue
		}
		volumes = append(volumes, types.DockerVolume{
			Name:           parts[0],
			Driver:         parts[1],
			Scope:          parts[2],
			Mountpoint:     parts[3],
			Size:           -1,
			RefCount:       -1,
			ComposeProject: LabelValue(parts[4], types.ComposeProjectLabel),
		})
	}
	return volumes, nil
}

// GetVolumeInspectJSON returns formatted JSON inspection for a Docker volume.
func GetVolumeInspectJSON(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("volume name cannot be empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), engine.DefaultDockerTimeout)
	defer cancel()

	if sockPath := engine.DockerSocketFinder(); sockPath != "" {
		client := engine.SocketClientFactory(sockPath)
		endpoint := fmt.Sprintf("http://localhost/volumes/%s", url.PathEscape(name))
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

	out, err := engine.DockerCLIRunner(ctx, "volume", "inspect", name)
	if err == nil && len(out) > 0 {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, out, "", "  "); err == nil {
			return pretty.String(), nil
		}
		return string(out), nil
	}

	return "", fmt.Errorf("volume not found: %s", name)
}

// RemoveDockerVolume removes a Docker volume by name.
func RemoveDockerVolume(name string, force bool) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("volume name cannot be empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), engine.DefaultDockerTimeout)
	defer cancel()

	if sockPath := engine.DockerSocketFinder(); sockPath != "" {
		client := engine.SocketClientFactory(sockPath)
		endpoint := fmt.Sprintf("http://localhost/volumes/%s?force=%t", url.PathEscape(name), force)
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

	args := []string{"volume", "rm"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, name)

	out, err := engine.DockerCLIRunner(ctx, args...)
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return nil
}

// MatchContainersForVolume finds container names that reference the given volume name in their mounts.
func MatchContainersForVolume(volName string, containers []types.ContainerItem) []string {
	if volName == "" {
		return nil
	}
	var matched []string
	for _, c := range containers {
		for _, mount := range c.Mounts {
			if strings.Contains(mount, volName) {
				matched = append(matched, c.Name)
				break
			}
		}
	}
	sort.Strings(matched)
	return matched
}

// EnrichVolumesWithContainers populates Containers slice on volumes based on container mounts.
func EnrichVolumesWithContainers(volumes []types.DockerVolume, containers []types.ContainerItem) []types.DockerVolume {
	if len(volumes) == 0 {
		return volumes
	}
	enriched := make([]types.DockerVolume, len(volumes))
	copy(enriched, volumes)
	for i := range enriched {
		if len(enriched[i].Containers) == 0 {
			enriched[i].Containers = MatchContainersForVolume(enriched[i].Name, containers)
		}
	}
	return enriched
}
