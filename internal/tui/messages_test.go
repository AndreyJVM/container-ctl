package tui

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
)

func TestMessages_Instantiations(t *testing.T) {
	t.Parallel()

	t.Run("tickMsg conversion", func(t *testing.T) {
		t.Parallel()
		now := time.Now()
		tick := tickMsg(now)
		if time.Time(tick) != now {
			t.Errorf("expected tickMsg %v, got %v", now, tick)
		}
	})

	t.Run("execFinishedMsg fields", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("exec error")
		execMsg := execFinishedMsg{
			containerName: "my-container",
			err:           testErr,
		}
		if execMsg.containerName != "my-container" || !errors.Is(execMsg.err, testErr) {
			t.Errorf("unexpected execFinishedMsg: %+v", execMsg)
		}
	})

	t.Run("embeddedShellOutputMsg fields", func(t *testing.T) {
		t.Parallel()
		shellOutMsg := embeddedShellOutputMsg{
			session: nil,
			data:    []byte("output-line\n"),
			err:     nil,
		}
		if string(shellOutMsg.data) != "output-line\n" {
			t.Errorf("unexpected embeddedShellOutputMsg data: %s", shellOutMsg.data)
		}
	})

	t.Run("embeddedShellExitedMsg fields", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("exit error")
		shellExitMsg := embeddedShellExitedMsg{
			session: nil,
			err:     testErr,
		}
		if !errors.Is(shellExitMsg.err, testErr) {
			t.Errorf("unexpected embeddedShellExitedMsg err: %v", shellExitMsg.err)
		}
	})

	t.Run("dataFetchedMsg fields", func(t *testing.T) {
		t.Parallel()
		dataMsg := dataFetchedMsg{
			dockerInfo: monitor.DockerSummary{RunningContainers: 5},
			images:     []monitor.DockerImage{{ID: "img-1"}},
			compose:    []monitor.ComposeProject{{Name: "proj-1"}},
			networks:   []monitor.DockerNetwork{{Name: "bridge"}},
			volumes:    []monitor.DockerVolume{{Name: "vol-1"}},
			swarm:      monitor.SwarmSummary{IsActive: false},
			imageErr:   "img err",
			networkErr: "net err",
			volumeErr:  "vol err",
			timestamp:  "12:00:00",
		}
		if dataMsg.dockerInfo.RunningContainers != 5 || len(dataMsg.images) != 1 || dataMsg.timestamp != "12:00:00" {
			t.Errorf("unexpected dataFetchedMsg: %+v", dataMsg)
		}
		if dataMsg.imageErr != "img err" || dataMsg.networkErr != "net err" || dataMsg.volumeErr != "vol err" {
			t.Errorf("unexpected error messages in dataFetchedMsg: %+v", dataMsg)
		}
	})

	t.Run("actionFinishedMsg fields", func(t *testing.T) {
		t.Parallel()
		actionMsg := actionFinishedMsg{
			message: "Action succeeded",
			err:     nil,
			refresh: true,
		}
		if actionMsg.message != "Action succeeded" || !actionMsg.refresh || actionMsg.err != nil {
			t.Errorf("unexpected actionFinishedMsg: %+v", actionMsg)
		}
	})
}

func TestDataFetchedMsg_StateApplicationAndSwarmFallback(t *testing.T) {
	t.Parallel()

	t.Run("applies fresh state across all domains", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.composeCursorIdx = 5
		m.swarmCursorIdx = 10

		data := dataFetchedMsg{
			dockerInfo: monitor.DockerSummary{
				Status:            monitor.DockerStatusRunning,
				RunningContainers: 3,
			},
			images:   []monitor.DockerImage{{ID: "img1"}, {ID: "img2"}},
			compose:  []monitor.ComposeProject{{Name: "stack1"}, {Name: "stack2"}},
			networks: []monitor.DockerNetwork{{Name: "net1"}},
			volumes:  []monitor.DockerVolume{{Name: "vol1"}},
			swarm: monitor.SwarmSummary{
				IsActive: true,
				Services: []monitor.SwarmService{{Name: "svc1"}},
			},
			imageErr:   "warn image",
			networkErr: "warn net",
			volumeErr:  "warn vol",
			timestamp:  "15:30:00",
		}

		res, cmd := m.Update(data)
		updated := res.(Model)
		if cmd != nil {
			t.Errorf("expected nil cmd after dataFetchedMsg")
		}
		if updated.lastUpdate != "15:30:00" {
			t.Errorf("expected lastUpdate '15:30:00', got %q", updated.lastUpdate)
		}
		if len(updated.images) != 2 || len(updated.compose) != 2 || len(updated.networks) != 1 || len(updated.volumes) != 1 {
			t.Errorf("unexpected resource lengths: %+v", updated)
		}
		// Verify cursor clamping
		if updated.composeCursorIdx != 1 {
			t.Errorf("expected composeCursorIdx clamped to 1, got %d", updated.composeCursorIdx)
		}
		if updated.swarmCursorIdx != 0 {
			t.Errorf("expected swarmCursorIdx clamped to 0, got %d", updated.swarmCursorIdx)
		}
		if updated.imageErr != "warn image" || updated.networkErr != "warn net" || updated.volumeErr != "warn vol" {
			t.Errorf("unexpected error messages populated")
		}
	})

	t.Run("switches from TabSwarm to TabDocker when Swarm becomes inactive", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeTab = TabSwarm
		m.showSwarm = false

		data := dataFetchedMsg{
			swarm: monitor.SwarmSummary{
				IsActive: false,
			},
			timestamp: "16:00:00",
		}

		res, _ := m.Update(data)
		updated := res.(Model)
		if updated.activeTab != TabDocker {
			t.Errorf("expected activeTab reset to TabDocker, got %v", updated.activeTab)
		}
		if !strings.Contains(updated.actionMessage, "Host is not part of an active Swarm cluster") {
			t.Errorf("expected swarm inactive toast, got %q", updated.actionMessage)
		}
	})
}

func TestActionFinishedMsg_Dispatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		msg           actionFinishedMsg
		expectedToast string
		expectCmd     bool
	}{
		{
			name:          "success without refresh",
			msg:           actionFinishedMsg{message: "Service scaled", refresh: false},
			expectedToast: "Service scaled",
			expectCmd:     false,
		},
		{
			name:          "success with refresh returns fetchDataCmd",
			msg:           actionFinishedMsg{message: "Container removed", refresh: true},
			expectedToast: "Container removed",
			expectCmd:     true,
		},
		{
			name:          "error message takes precedence",
			msg:           actionFinishedMsg{message: "ignored", err: errors.New("daemon timeout"), refresh: false},
			expectedToast: "daemon timeout",
			expectCmd:     false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := newFastTestModel()
			res, cmd := m.Update(tc.msg)
			updated := res.(Model)

			if (cmd != nil) != tc.expectCmd {
				t.Errorf("expected cmd %v, got %v", tc.expectCmd, cmd != nil)
			}
			if !strings.Contains(updated.actionMessage, tc.expectedToast) {
				t.Errorf("expected toast %q, got %q", tc.expectedToast, updated.actionMessage)
			}
		})
	}
}

func TestTickMsg_TimerReschedule(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()
	now := time.Now()

	_, cmd := m.Update(tickMsg(now))
	if cmd == nil {
		t.Fatalf("expected non-nil batch cmd on tickMsg")
	}
}

func TestExecFinishedMsg_HandlingVariants(t *testing.T) {
	t.Parallel()

	t.Run("nil error (clean exit)", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		res, _ := m.Update(execFinishedMsg{containerName: "web-srv", err: nil})
		updated := res.(Model)
		if !strings.Contains(updated.actionMessage, "Shell session in web-srv ended") {
			t.Errorf("expected clean exit toast, got %q", updated.actionMessage)
		}
	})

	t.Run("exit code 130 (SIGINT / Ctrl+C)", func(t *testing.T) {
		t.Parallel()
		exit130 := exec.Command("bash", "-c", "exit 130").Run()
		if exit130 == nil {
			t.Skip("unable to generate exit 130")
		}

		m := newFastTestModel()
		res, _ := m.Update(execFinishedMsg{containerName: "db-srv", err: exit130})
		updated := res.(Model)
		if !strings.Contains(updated.actionMessage, "Shell session in db-srv ended") ||
			strings.Contains(updated.actionMessage, "code") {
			t.Errorf("expected clean exit toast without code, got %q", updated.actionMessage)
		}
	})

	t.Run("exit code 1 (general error)", func(t *testing.T) {
		t.Parallel()
		exit1 := exec.Command("bash", "-c", "exit 1").Run()
		if exit1 == nil {
			t.Skip("unable to generate exit 1")
		}

		m := newFastTestModel()
		res, _ := m.Update(execFinishedMsg{containerName: "cache-srv", err: exit1})
		updated := res.(Model)
		if !strings.Contains(updated.actionMessage, "Shell session in cache-srv ended (code 1)") {
			t.Errorf("expected code 1 toast, got %q", updated.actionMessage)
		}
	})

	t.Run("generic non-ExitError error", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		res, _ := m.Update(execFinishedMsg{containerName: "api-srv", err: errors.New("exec stream failed")})
		updated := res.(Model)
		if !strings.Contains(updated.actionMessage, "Shell error in api-srv: exec stream failed") {
			t.Errorf("expected shell error toast, got %q", updated.actionMessage)
		}
	})
}
