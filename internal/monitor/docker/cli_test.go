package docker

import (
	"context"
	"errors"
	"testing"
)

func TestParseCLIPsOutput(t *testing.T) {
	t.Run("empty output", func(t *testing.T) {
		res := parseCLIPsOutput("")
		if len(res) != 0 {
			t.Errorf("expected empty slice, got %d items", len(res))
		}
	})

	t.Run("malformed lines with insufficient fields", func(t *testing.T) {
		input := "c1\tname1\timage1\n\n  \nc2\tname2\n"
		res := parseCLIPsOutput(input)
		if len(res) != 0 {
			t.Errorf("expected empty slice for malformed lines, got %d items", len(res))
		}
	})

	t.Run("standard 11 fields output with mounts and compose labels", func(t *testing.T) {
		input := "1234567890ab\tweb-app\tnginx:alpine\trunning\tUp 2 hours\t80->80/tcp\t/data,/app/log\tcom.docker.compose.project=myproj,com.docker.compose.service=web\tbridge\t2026-01-01 12:00:00\t2 hours ago\n"
		res := parseCLIPsOutput(input)
		if len(res) != 1 {
			t.Fatalf("expected 1 container, got %d", len(res))
		}
		c := res[0]
		if c.ID != "1234567890ab" || c.Name != "web-app" || c.Image != "nginx:alpine" {
			t.Errorf("unexpected container identity: %+v", c)
		}
		if c.State != "running" || c.Status != "Up 2 hours" || c.Ports != "80->80/tcp" {
			t.Errorf("unexpected state/status/ports: %+v", c)
		}
		if len(c.Mounts) != 2 || c.Mounts[0] != "/data" || c.Mounts[1] != "/app/log" {
			t.Errorf("unexpected mounts: %v", c.Mounts)
		}
		if c.ComposeProject != "myproj" || c.ComposeService != "web" {
			t.Errorf("unexpected compose info: proj=%q svc=%q", c.ComposeProject, c.ComposeService)
		}
		if c.NetworkName != "bridge" || c.Created != "2026-01-01 12:00:00" || c.StartedAt != "2 hours ago" {
			t.Errorf("unexpected network/created/started: %+v", c)
		}
	})
}

func TestEnrichContainersWithCLIInspect(t *testing.T) {
	origRunner := DockerCLIRunner
	defer func() { DockerCLIRunner = origRunner }()

	t.Run("empty containers slice returns immediately", func(t *testing.T) {
		called := false
		DockerCLIRunner = func(_ context.Context, _ ...string) ([]byte, error) {
			called = true
			return nil, nil
		}
		enrichContainersWithCLIInspect(nil)
		if called {
			t.Errorf("DockerCLIRunner should not be called for empty containers")
		}
	})

	t.Run("cli error gracefully ignored", func(t *testing.T) {
		DockerCLIRunner = func(_ context.Context, _ ...string) ([]byte, error) {
			return nil, errors.New("command failed")
		}
		containers := []ContainerItem{{ID: "c1", Status: "running"}}
		enrichContainersWithCLIInspect(containers)
		if containers[0].RestartPolicy != "" {
			t.Errorf("expected empty restart policy on error")
		}
	})

	t.Run("successful enrichment with restart policy, ip address, and logs on failure", func(t *testing.T) {
		DockerCLIRunner = func(_ context.Context, args ...string) ([]byte, error) {
			if len(args) > 0 && args[0] == "inspect" {
				return []byte("c1|always|172.18.0.2 \nc2|no|172.18.0.3 \n"), nil
			}
			if len(args) > 0 && args[0] == "logs" {
				return []byte("Error: fatal connection dropped\nProcess exited with status 1"), nil
			}
			return nil, nil
		}

		containers := []ContainerItem{
			{ID: "c1", State: "running", Status: "Up 1 hour"},
			{ID: "c2", State: "exited", Status: "Exited (1) 2 minutes ago"},
		}

		enrichContainersWithCLIInspect(containers)

		if containers[0].RestartPolicy != "always" || containers[0].IPAddress != "172.18.0.2" {
			t.Errorf("c1 enrichment failed: %+v", containers[0])
		}
		if containers[1].RestartPolicy != "no" || containers[1].IPAddress != "172.18.0.3" {
			t.Errorf("c2 enrichment failed: %+v", containers[1])
		}
		if containers[1].ErrorSnippet == "" {
			t.Errorf("expected error snippet for failed container c2")
		}
	})
}
