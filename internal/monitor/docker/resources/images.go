package resources

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/monitor/docker/engine"
	"github.com/AndreyJVM/container-ctl/internal/monitor/docker/types"
)

// GetDockerImages returns Docker images without changing daemon state.
func GetDockerImages() ([]types.DockerImage, error) {
	if sockPath := engine.DockerSocketFinder(); sockPath != "" {
		var raw []struct {
			ID          string            `json:"Id"`
			RepoTags    []string          `json:"RepoTags"`
			RepoDigests []string          `json:"RepoDigests"`
			Created     int64             `json:"Created"`
			Size        int64             `json:"Size"`
			Containers  int               `json:"Containers"`
			Labels      map[string]string `json:"Labels"`
		}
		if err := engine.FetchDockerResourceJSON(sockPath, "/images/json", &raw); err == nil {
			images := make([]types.DockerImage, 0, len(raw))
			for _, item := range raw {
				repo, tag, dangling := parseRepoTags(item.RepoTags)
				digest := "-"
				if len(item.RepoDigests) > 0 {
					digest = cleanImageDigest(item.RepoDigests[0])
				}
				createdStr := ""
				if item.Created > 0 {
					createdStr = time.Unix(item.Created, 0).Format("2006-01-02 15:04")
				}
				images = append(images, types.DockerImage{
					ID:         cleanImageID(item.ID),
					FullID:     item.ID,
					Repository: repo,
					Tag:        tag,
					AllTags:    item.RepoTags,
					Size:       item.Size,
					Created:    createdStr,
					Containers: item.Containers,
					Digest:     digest,
					InUse:      item.Containers > 0,
					Dangling:   dangling,
				})
			}
			sortDockerImages(images)
			return images, nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), engine.DefaultDockerTimeout)
	defer cancel()
	format := "{{.ID}}|{{.Repository}}|{{.Tag}}|{{.Size}}|{{.CreatedAt}}|{{.Digest}}"
	out, err := engine.DockerCLIRunner(ctx, "image", "ls", "-a", "--no-trunc", "--format", format)
	if err != nil {
		return nil, fmt.Errorf("failed to list Docker images: %w", err)
	}

	var images []types.DockerImage
	for _, line := range NonEmptyLines(string(out)) {
		parts := strings.SplitN(line, "|", 6)
		if len(parts) != 6 {
			continue
		}
		repo := parts[1]
		tag := parts[2]
		dangling := repo == "<none>" && tag == "<none>"
		digest := cleanImageDigest(parts[5])
		created := formatCLICreatedAt(parts[4])

		images = append(images, types.DockerImage{
			ID:         cleanImageID(parts[0]),
			FullID:     parts[0],
			Repository: repo,
			Tag:        tag,
			AllTags:    []string{fmt.Sprintf("%s:%s", repo, tag)},
			Size:       parseSizeToBytes(parts[3]),
			Created:    created,
			Containers: -1,
			Digest:     digest,
			InUse:      false,
			Dangling:   dangling,
		})
	}
	sortDockerImages(images)
	return images, nil
}

// RemoveDockerImage removes a Docker image by ID or name, with optional force.
func RemoveDockerImage(imageID string, force bool) error {
	if strings.TrimSpace(imageID) == "" {
		return fmt.Errorf("image ID cannot be empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sockPath := engine.DockerSocketFinder()
	if sockPath != "" {
		client := engine.SocketClientFactory(sockPath)
		endpoint := fmt.Sprintf("http://localhost/images/%s?force=%t", url.PathEscape(imageID), force)
		req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
		if err == nil {
			resp, err := client.Do(req)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
					return nil
				}
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
			}
		}
	}
	args := []string{"rmi"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, imageID)
	out, err := engine.DockerCLIRunner(ctx, args...)
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return nil
}

func parseRepoTags(tags []string) (repo, tag string, dangling bool) {
	if len(tags) == 0 || (len(tags) == 1 && tags[0] == "<none>:<none>") {
		return "<none>", "<none>", true
	}
	first := tags[0]
	idx := strings.LastIndex(first, ":")
	slashIdx := strings.LastIndex(first, "/")
	var baseTag string
	if idx == -1 || idx < slashIdx {
		repo = first
		baseTag = "<none>"
	} else {
		repo = first[:idx]
		baseTag = first[idx+1:]
	}
	if len(tags) > 1 {
		tag = fmt.Sprintf("%s (+%d)", baseTag, len(tags)-1)
	} else {
		tag = baseTag
	}
	return repo, tag, false
}

func cleanImageID(id string) string {
	id = strings.TrimSpace(id)
	id = strings.TrimPrefix(id, "sha256:")
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func cleanImageDigest(digest string) string {
	digest = strings.TrimSpace(digest)
	if digest == "" || digest == "<none>" {
		return "-"
	}
	if strings.Contains(digest, "@") {
		parts := strings.SplitN(digest, "@", 2)
		digest = parts[1]
	}
	digest = strings.TrimPrefix(digest, "sha256:")
	if len(digest) > 12 {
		return digest[:12]
	}
	return digest
}

func formatCLICreatedAt(created string) string {
	created = strings.TrimSpace(created)
	if len(created) >= 16 {
		return created[:16]
	}
	return created
}

func parseSizeToBytes(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" {
		return -1
	}
	unit := int64(1)
	upper := strings.ToUpper(s)
	var numStr string
	switch {
	case strings.HasSuffix(upper, "TB") || strings.HasSuffix(upper, "TIB"):
		unit = 1024 * 1024 * 1024 * 1024
		numStr = strings.TrimSuffix(strings.TrimSuffix(upper, "TIB"), "TB")
	case strings.HasSuffix(upper, "GB") || strings.HasSuffix(upper, "GIB"):
		unit = 1024 * 1024 * 1024
		numStr = strings.TrimSuffix(strings.TrimSuffix(upper, "GIB"), "GB")
	case strings.HasSuffix(upper, "MB") || strings.HasSuffix(upper, "MIB"):
		unit = 1024 * 1024
		numStr = strings.TrimSuffix(strings.TrimSuffix(upper, "MIB"), "MB")
	case strings.HasSuffix(upper, "KB") || strings.HasSuffix(upper, "KIB"):
		unit = 1024
		numStr = strings.TrimSuffix(strings.TrimSuffix(upper, "KIB"), "KB")
	case strings.HasSuffix(upper, "B"):
		unit = 1
		numStr = strings.TrimSuffix(upper, "B")
	default:
		numStr = upper
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(numStr), 64)
	if err != nil {
		return -1
	}
	return int64(f * float64(unit))
}

func sortDockerImages(images []types.DockerImage) {
	sort.Slice(images, func(i, j int) bool {
		if images[i].Dangling != images[j].Dangling {
			return !images[i].Dangling
		}
		if images[i].Repository != images[j].Repository {
			return images[i].Repository < images[j].Repository
		}
		return images[i].Tag < images[j].Tag
	})
}
