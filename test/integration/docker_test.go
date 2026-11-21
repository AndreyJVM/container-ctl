package integration

import (
	"os"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor/docker"
)

// TestLiveDockerDaemonIntegration runs integration tests against the live Docker daemon if available.
func TestLiveDockerDaemonIntegration(t *testing.T) {
	sock := docker.FindDockerSocket()
	if sock == "" {
		t.Skip("Skipping live integration test: No Docker socket found")
	}
	if _, err := os.Stat(sock); err != nil {
		t.Skipf("Skipping live integration test: Socket %s inaccessible: %v", sock, err)
	}

	summary := docker.GetDockerSummary()
	if summary.Status != docker.DockerStatusRunning {
		t.Skipf("Skipping live integration test: Docker daemon is not running (status: %s)", summary.Status)
	}

	t.Logf("Live Docker Engine detected: version %s, %d containers, %d images",
		summary.Version, summary.RunningContainers+summary.StoppedContainers, summary.ImagesCount)

	// 1. Containers from summary
	containers := summary.Containers
	t.Logf("Successfully fetched %d containers from live daemon", len(containers))
	for _, c := range containers {
		if c.ID == "" {
			t.Errorf("Container has empty ID")
		}
	}

	// 2. Fetch live images
	images, imgErr := docker.GetDockerImages()
	if imgErr != nil {
		t.Errorf("GetDockerImages() failed: %v", imgErr)
	} else {
		t.Logf("Successfully fetched %d images from live daemon", len(images))
	}

	// 3. Fetch live networks
	networks, netErr := docker.GetDockerNetworks()
	if netErr != nil {
		t.Errorf("GetDockerNetworks() failed: %v", netErr)
	} else {
		t.Logf("Successfully fetched %d networks from live daemon", len(networks))
	}

	// 4. Fetch live volumes
	volumes, volErr := docker.GetDockerVolumes()
	if volErr != nil {
		t.Errorf("GetDockerVolumes() failed: %v", volErr)
	} else {
		t.Logf("Successfully fetched %d volumes from live daemon", len(volumes))
	}

	// 5. Fetch disk usage
	usage, usageErr := docker.GetDockerDiskUsage()
	if usageErr != nil {
		t.Logf("GetDockerDiskUsage() returned: %v (may require elevated permissions)", usageErr)
	} else {
		t.Logf("Disk usage: Total Reclaimable = %d bytes", usage.TotalRecl)
	}

	// 6. Detailed inspection of first container (if any)
	if len(containers) > 0 {
		first := containers[0]
		detail, detailErr := docker.GetContainerDetail(first.ID)
		if detailErr != nil {
			t.Errorf("GetContainerDetail(%q) failed: %v", first.ID, detailErr)
		} else if detail.ID == "" {
			t.Errorf("GetContainerDetail(%q) returned empty detail ID", first.ID)
		}

		logs, logErr := docker.GetContainerLogs(first.ID, 10)
		if logErr != nil {
			t.Logf("GetContainerLogs(%q) error: %v", first.ID, logErr)
		} else {
			t.Logf("Fetched logs for container %s (len=%d bytes)", first.Name, len(logs))
		}
	}
}
