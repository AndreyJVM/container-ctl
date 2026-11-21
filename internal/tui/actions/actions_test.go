package actions

import (
	"errors"
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
)

func TestMakeActionCmd(t *testing.T) {
	// Success
	cmd := MakeActionCmd(func() error { return nil }, "err: ", "Success")
	msg := cmd().(FinishedMsg)
	if msg.Err != nil || msg.Message != "Success" || !msg.Refresh {
		t.Errorf("unexpected success message: %+v", msg)
	}

	// Error
	cmdErr := MakeActionCmd(func() error { return errors.New("failed") }, "err: ", "Success")
	msgErr := cmdErr().(FinishedMsg)
	if msgErr.Err == nil || !strings.Contains(msgErr.Err.Error(), "err: failed") {
		t.Errorf("unexpected error message: %+v", msgErr)
	}
}

func TestContainerAndComposeCommands(t *testing.T) {
	// StartContainer
	msg := StartContainer("c1", "web")().(FinishedMsg)
	if msg.Err == nil && !strings.Contains(msg.Message, "web started") {
		t.Errorf("unexpected StartContainer msg: %+v", msg)
	}

	// StopContainer
	msg = StopContainer("c1", "web")().(FinishedMsg)
	if msg.Err == nil && !strings.Contains(msg.Message, "web stopped") {
		t.Errorf("unexpected StopContainer msg: %+v", msg)
	}

	// RestartContainer
	msg = RestartContainer("c1", "web")().(FinishedMsg)
	if msg.Err == nil && !strings.Contains(msg.Message, "web restarted") {
		t.Errorf("unexpected RestartContainer msg: %+v", msg)
	}

	// KillContainer
	msg = KillContainer("c1", "web")().(FinishedMsg)
	if msg.Err == nil && !strings.Contains(msg.Message, "web force-killed") {
		t.Errorf("unexpected KillContainer msg: %+v", msg)
	}

	// RemoveContainer
	msg = RemoveContainer("c1", "web")().(FinishedMsg)
	if msg.Err == nil && !strings.Contains(msg.Message, "web removed") {
		t.Errorf("unexpected RemoveContainer msg: %+v", msg)
	}

	// RemoveImage
	msg = RemoveImage("img1", "alpine:latest", false)().(FinishedMsg)
	if msg.Err == nil && !strings.Contains(msg.Message, "alpine:latest removed") {
		t.Errorf("unexpected RemoveImage msg: %+v", msg)
	}

	containers := []monitor.ContainerItem{{ID: "c1", ComposeProject: "proj"}}

	// StartCompose
	msg = StartCompose("proj", containers)().(FinishedMsg)
	if msg.Err == nil && !strings.Contains(msg.Message, "proj started") {
		t.Errorf("unexpected StartCompose msg: %+v", msg)
	}

	// StopCompose
	msg = StopCompose("proj", containers)().(FinishedMsg)
	if msg.Err == nil && !strings.Contains(msg.Message, "proj stopped") {
		t.Errorf("unexpected StopCompose msg: %+v", msg)
	}

	// RestartCompose
	msg = RestartCompose("proj", containers)().(FinishedMsg)
	if msg.Err == nil && !strings.Contains(msg.Message, "proj restarted") {
		t.Errorf("unexpected RestartCompose msg: %+v", msg)
	}

	// DownCompose
	msg = DownCompose("proj", containers)().(FinishedMsg)
	if msg.Err == nil && !strings.Contains(msg.Message, "proj downed") {
		t.Errorf("unexpected DownCompose msg: %+v", msg)
	}
}

func TestPruneActions(t *testing.T) {
	opts := []int{PruneOptionContainers, PruneOptionImages, PruneOptionVolumes, PruneOptionAll}
	for _, opt := range opts {
		cmd := Prune(opt)
		if cmd == nil {
			t.Errorf("expected non-nil cmd for option %d", opt)
		}
		_ = cmd()
	}

	// Invalid option
	_, err := ExecutePrune(99)
	if err == nil {
		t.Errorf("expected error for invalid prune option")
	}
}

func TestFetchAllDataAndCmd(t *testing.T) {
	data := FetchAllData()
	if len(data.Timestamp) != 8 {
		t.Errorf("expected valid timestamp, got %s", data.Timestamp)
	}

	cmd := FetchData()
	msg := cmd().(DataFetchedMsg)
	if len(msg.Timestamp) != 8 {
		t.Errorf("expected valid timestamp from FetchData, got %s", msg.Timestamp)
	}
}
