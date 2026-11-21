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

// DockerVolume describes a Docker volume shown in the dashboard.
type DockerVolume struct {
	Name           string
	Driver         string
	Scope          string
	Mountpoint     string
	Size           int64
	RefCount       int
	ComposeProject string
}

// GetDockerVolumes returns Docker volumes without changing daemon state.
func GetDockerVolumes() ([]DockerVolume, error) {
	if sockPath := DockerSocketFinder(); sockPath != "" {
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
		if err := fetchDockerResourceJSON(sockPath, "/volumes", &raw); err == nil {
			volumes := make([]DockerVolume, 0, len(raw.Volumes))
			for _, item := range raw.Volumes {
				size, refCount := int64(-1), -1
				if item.UsageData != nil {
					size = item.UsageData.Size
					refCount = item.UsageData.RefCount
				}
				volumes = append(volumes, DockerVolume{
					Name:           item.Name,
					Driver:         item.Driver,
					Scope:          item.Scope,
					Mountpoint:     item.Mountpoint,
					Size:           size,
					RefCount:       refCount,
					ComposeProject: item.Labels[composeProjectLabel],
				})
			}
			return volumes, nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultDockerTimeout)
	defer cancel()
	format := "{{.Name}}|{{.Driver}}|{{.Scope}}|{{.Mountpoint}}|{{.Labels}}"
	out, err := DockerCLIRunner(ctx, "volume", "ls", "--format", format)
	if err != nil {
		return nil, fmt.Errorf("failed to list Docker volumes: %w", err)
	}

	var volumes []DockerVolume
	for _, line := range nonEmptyLines(string(out)) {
		parts := strings.SplitN(line, "|", 5)
		if len(parts) != 5 {
			continue
		}
		volumes = append(volumes, DockerVolume{
			Name:           parts[0],
			Driver:         parts[1],
			Scope:          parts[2],
			Mountpoint:     parts[3],
			Size:           -1,
			RefCount:       -1,
			ComposeProject: labelValue(parts[4], composeProjectLabel),
		})
	}
	return volumes, nil
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
