package actions

import (
	"errors"
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
)

func TestBatchActions(t *testing.T) {
	// Backup original funcs and restore at end
	origStart := startContainerFunc
	origStop := stopContainerFunc
	origRestart := restartContainerFunc
	origKill := killContainerFunc
	origRemove := removeContainerFunc
	defer func() {
		startContainerFunc = origStart
		stopContainerFunc = origStop
		restartContainerFunc = origRestart
		killContainerFunc = origKill
		removeContainerFunc = origRemove
	}()

	c1 := monitor.ContainerItem{ID: "c1", Name: "app1", State: "stopped"}
	c2 := monitor.ContainerItem{ID: "c2", Name: "app2", State: "running"}
	c3 := monitor.ContainerItem{ID: "c3", Name: "app3", State: "running"}

	t.Run("empty containers return No containers selected", func(t *testing.T) {
		msg := BatchStartContainers(nil)().(FinishedMsg)
		if msg.Message != "No containers selected" {
			t.Errorf("expected No containers selected, got %q", msg.Message)
		}
	})

	t.Run("BatchStartContainers successfully starts stopped containers", func(t *testing.T) {
		startedIDs := []string{}
		startContainerFunc = func(id string) error {
			startedIDs = append(startedIDs, id)
			return nil
		}

		msg := BatchStartContainers([]monitor.ContainerItem{c1, c2})().(FinishedMsg)
		if msg.Err != nil {
			t.Errorf("unexpected error: %v", msg.Err)
		}
		if len(startedIDs) != 1 || startedIDs[0] != "c1" {
			t.Errorf("expected c1 started, got %v", startedIDs)
		}
		if !strings.Contains(msg.Message, "Started 1 container(s) successfully") {
			t.Errorf("unexpected msg: %s", msg.Message)
		}
	})

	t.Run("BatchStartContainers handles errors", func(t *testing.T) {
		startContainerFunc = func(id string) error {
			return errors.New("cannot start")
		}

		msg := BatchStartContainers([]monitor.ContainerItem{c1})().(FinishedMsg)
		if msg.Err == nil {
			t.Errorf("expected error, got nil")
		}
		if !strings.Contains(msg.Message, "0 container(s), 1 failed") {
			t.Errorf("unexpected message: %s", msg.Message)
		}
	})

	t.Run("BatchStopContainers successfully stops running containers", func(t *testing.T) {
		stoppedIDs := []string{}
		stopContainerFunc = func(id string) error {
			stoppedIDs = append(stoppedIDs, id)
			return nil
		}

		msg := BatchStopContainers([]monitor.ContainerItem{c1, c2, c3})().(FinishedMsg)
		if msg.Err != nil {
			t.Errorf("unexpected error: %v", msg.Err)
		}
		if len(stoppedIDs) != 2 {
			t.Errorf("expected 2 stopped, got %v", stoppedIDs)
		}
		if !strings.Contains(msg.Message, "Stopped 2 container(s) successfully") {
			t.Errorf("unexpected msg: %s", msg.Message)
		}
	})

	t.Run("BatchRestartContainers restarts all", func(t *testing.T) {
		restartedIDs := []string{}
		restartContainerFunc = func(id string) error {
			restartedIDs = append(restartedIDs, id)
			return nil
		}

		msg := BatchRestartContainers([]monitor.ContainerItem{c1, c2})().(FinishedMsg)
		if msg.Err != nil {
			t.Errorf("unexpected error: %v", msg.Err)
		}
		if len(restartedIDs) != 2 {
			t.Errorf("expected 2 restarted, got %v", restartedIDs)
		}
		if !strings.Contains(msg.Message, "Restarted 2 container(s) successfully") {
			t.Errorf("unexpected msg: %s", msg.Message)
		}
	})

	t.Run("BatchKillContainers kills only running", func(t *testing.T) {
		killedIDs := []string{}
		killContainerFunc = func(id string) error {
			killedIDs = append(killedIDs, id)
			return nil
		}

		msg := BatchKillContainers([]monitor.ContainerItem{c1, c2})().(FinishedMsg)
		if msg.Err != nil {
			t.Errorf("unexpected error: %v", msg.Err)
		}
		if len(killedIDs) != 1 || killedIDs[0] != "c2" {
			t.Errorf("expected c2 killed, got %v", killedIDs)
		}
		if !strings.Contains(msg.Message, "Force-killed 1 container(s) successfully") {
			t.Errorf("unexpected msg: %s", msg.Message)
		}
	})

	t.Run("BatchRemoveContainers removes all", func(t *testing.T) {
		removedIDs := []string{}
		removeContainerFunc = func(id string) error {
			removedIDs = append(removedIDs, id)
			return nil
		}

		msg := BatchRemoveContainers([]monitor.ContainerItem{c1, c2})().(FinishedMsg)
		if msg.Err != nil {
			t.Errorf("unexpected error: %v", msg.Err)
		}
		if len(removedIDs) != 2 {
			t.Errorf("expected 2 removed, got %v", removedIDs)
		}
		if !strings.Contains(msg.Message, "Removed 2 container(s) successfully") {
			t.Errorf("unexpected msg: %s", msg.Message)
		}
	})
}
