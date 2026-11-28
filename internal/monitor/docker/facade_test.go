package docker

import (
	"strings"
	"testing"
)

func TestFacade_ResourcesForwarding(t *testing.T) {
	// FormatBytes / FormatUintBytes
	if s := FormatBytes(1024 * 1024); !strings.Contains(s, "1.00 MB") && !strings.Contains(s, "1 MB") && !strings.Contains(s, "1.0") {
		t.Errorf("unexpected FormatBytes result: %s", s)
	}
	if s := FormatUintBytes(1024 * 1024 * 1024); !strings.Contains(s, "1.00 GB") && !strings.Contains(s, "1 GB") && !strings.Contains(s, "1.0") {
		t.Errorf("unexpected FormatUintBytes result: %s", s)
	}

	// MatchContainersForVolume & EnrichVolumesWithContainers
	containers := []ContainerItem{
		{
			Name:   "app-web",
			Mounts: []string{"data-vol:/data"},
		},
	}
	matched := MatchContainersForVolume("data-vol", containers)
	if len(matched) != 1 || matched[0] != "app-web" {
		t.Errorf("expected ['app-web'], got %v", matched)
	}

	vols := []DockerVolume{
		{Name: "data-vol"},
		{Name: "orphan-vol"},
	}
	enriched := EnrichVolumesWithContainers(vols, containers)
	if len(enriched) != 2 || len(enriched[0].Containers) != 1 {
		t.Errorf("expected volume enriched with 1 container, got: %+v", enriched)
	}
}

func TestFacade_ExecAndLogsForwarding(t *testing.T) {
	// DemuxDockerLogs
	header := []byte{1, 0, 0, 0, 0, 0, 0, 5, 'h', 'e', 'l', 'l', 'o'}
	demuxed := DemuxDockerLogs(strings.NewReader(string(header)))
	if !strings.Contains(demuxed, "hello") {
		t.Errorf("expected 'hello' in demuxed output, got %q", demuxed)
	}

	// CleanLogSnippet
	snippet := CleanLogSnippet("line 1\nline 2\nline 3\n\n", 2)
	if !strings.Contains(snippet, "line 2") || !strings.Contains(snippet, "line 3") {
		t.Errorf("unexpected snippet: %q", snippet)
	}

	// GetContainerExecCommand & GetContainerExecCommandWithOptions
	cmdDefault := GetContainerExecCommand("c123")
	if cmdDefault == nil || len(cmdDefault.Args) < 3 {
		t.Fatalf("expected valid exec.Cmd, got %v", cmdDefault)
	}

	cmdCustom := GetContainerExecCommandWithOptions("c123", ExecOptions{
		Mode:      "custom",
		CustomCmd: "ls -la /tmp",
	})
	if cmdCustom == nil || len(cmdCustom.Args) < 3 {
		t.Fatalf("expected valid custom exec.Cmd, got %v", cmdCustom)
	}
}
