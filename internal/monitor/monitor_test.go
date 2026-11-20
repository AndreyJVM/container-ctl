package monitor_test

import (
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
)

func TestFacadeExports(t *testing.T) {
	// Verify that key types are accessible through the monitor facade
	var (
		_ monitor.ContainerItem
		_ monitor.ContainerDetail
		_ monitor.DockerSummary
		_ monitor.DockerDiskUsage
		_ monitor.ComposeProject
		_ monitor.SwarmSummary
	)

	// Verify constants
	if monitor.StatusRunning != "running" {
		t.Errorf("expected StatusRunning == 'running', got %s", monitor.StatusRunning)
	}
	if monitor.StatusPartial != "partial" || monitor.StatusStopped != "stopped" {
		t.Errorf("unexpected compose statuses")
	}
	if monitor.ActionRestart != "restart" || monitor.ActionStop != "stop" || monitor.ActionStart != "start" || monitor.ActionDown != "down" {
		t.Errorf("unexpected compose actions")
	}

	// Verify health status parser
	if monitor.ParseHealthStatus("healthy") != monitor.HealthStatusHealthy {
		t.Errorf("expected HealthStatusHealthy")
	}
	if monitor.ParseHealthStatus("unhealthy") != monitor.HealthStatusUnhealthy {
		t.Errorf("expected HealthStatusUnhealthy")
	}
	if monitor.ParseHealthStatus("starting") != monitor.HealthStatusStarting {
		t.Errorf("expected HealthStatusStarting")
	}
	if monitor.ParseHealthStatus("unknown") != "" {
		t.Errorf("expected empty for unknown health")
	}

	// Verify exit code parser and container failed
	failed, code := monitor.ParseExitCode("Exited (137) 2 hours ago")
	if !failed || code != 137 {
		t.Errorf("expected failed=true, code=137, got %v, %d", failed, code)
	}
	failed, code = monitor.IsContainerFailed("exited", "Exited (1) 5 mins ago")
	if !failed || code != 1 {
		t.Errorf("expected failed=true, code=1, got %v, %d", failed, code)
	}

	// Verify format bytes
	if formatted := monitor.FormatBytes(1024 * 1024); formatted != "1.0 MB" {
		t.Errorf("expected '1.0 MB', got %s", formatted)
	}
	if formatted := monitor.FormatUintBytes(2 * 1024 * 1024); formatted != "2.0 MB" {
		t.Errorf("expected '2.0 MB', got %s", formatted)
	}

	// Verify socket & CLI checker
	_ = monitor.FindDockerSocket()
	restore := monitor.SetDockerCLIAvailableCheckerForTesting(func() bool { return true })
	if !monitor.IsDockerCLIAvailable() {
		t.Errorf("expected CLI to be available via mocked checker")
	}
	restore()

	// Verify exec command builders
	cmd := monitor.GetContainerExecCommand("c123")
	if cmd == nil || len(cmd.Args) == 0 {
		t.Errorf("expected valid exec command")
	}
	cmdOpt := monitor.GetContainerExecCommandWithOptions("c123", monitor.ExecOptions{
		Mode:      "custom",
		User:      "root",
		CustomCmd: "sh",
	})
	if cmdOpt == nil || len(cmdOpt.Args) == 0 {
		t.Errorf("expected valid exec command with options")
	}

	// Verify compose facade functions
	containers := []monitor.ContainerItem{
		{ID: "c1", Name: "proj_web_1", ComposeProject: "proj", ComposeService: "web", State: "running"},
	}
	projects := monitor.GetComposeProjects(containers)
	if len(projects) != 1 || projects[0].Name != "proj" {
		t.Errorf("expected 1 compose project 'proj'")
	}
	_, _ = monitor.GetComposeProjectLogs("proj", containers, 10)
	_ = monitor.RestartComposeProject("proj", containers)
	_ = monitor.StopComposeProject("proj", containers)
	_ = monitor.StartComposeProject("proj", containers)
	_ = monitor.DownComposeProject("proj", containers)

	// Verify swarm facade functions
	_ = monitor.GetSwarmSummary()
	_, _ = monitor.GetSwarmServiceLogs("srv1", 10)

	// Verify prune facade functions
	_, _ = monitor.PruneContainers()
	_, _ = monitor.PruneImages()
	_, _ = monitor.PruneVolumes()
	_, _ = monitor.PruneAll()

	// Verify docker client facade functions (safely invoked without panic)
	_ = monitor.GetDockerSummary()
	_, _ = monitor.GetContainerDetail("non-existent-id")
	_, _ = monitor.GetContainerLogs("non-existent-id", 5)
	_, _ = monitor.GetDockerImages()
	_ = monitor.RemoveDockerImage("non-existent-id", false)
	_, _ = monitor.GetDockerNetworks()
	_, _ = monitor.GetDockerVolumes()
	_, _ = monitor.GetDockerDiskUsage()
	_ = monitor.StartContainer("c1")
	_ = monitor.StopContainer("c1")
	_ = monitor.RestartContainer("c1")
	_ = monitor.KillContainer("c1")
	_ = monitor.RemoveContainer("c1")
}

func TestInspectAndCustomSocketFacade(t *testing.T) {
	_, _ = monitor.GetNetworkInspectJSON("bridge")
	_, _ = monitor.GetVolumeInspectJSON("my-vol")
}
