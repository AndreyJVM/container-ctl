package tui

import (
	"errors"
	"testing"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
)

func TestMessages_Instantiations(t *testing.T) {
	now := time.Now()
	tick := tickMsg(now)
	if time.Time(tick) != now {
		t.Errorf("expected tickMsg %v, got %v", now, tick)
	}

	testErr := errors.New("exec error")
	execMsg := execFinishedMsg{
		containerName: "my-container",
		err:           testErr,
	}
	if execMsg.containerName != "my-container" || execMsg.err != testErr {
		t.Errorf("unexpected execFinishedMsg: %+v", execMsg)
	}

	shellOutMsg := embeddedShellOutputMsg{
		session: nil,
		data:    []byte("output-line\n"),
		err:     nil,
	}
	if string(shellOutMsg.data) != "output-line\n" {
		t.Errorf("unexpected embeddedShellOutputMsg data: %s", shellOutMsg.data)
	}

	shellExitMsg := embeddedShellExitedMsg{
		session: nil,
		err:     testErr,
	}
	if shellExitMsg.err != testErr {
		t.Errorf("unexpected embeddedShellExitedMsg err: %v", shellExitMsg.err)
	}

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

	actionMsg := actionFinishedMsg{
		message: "Action succeeded",
		err:     nil,
		refresh: true,
	}
	if actionMsg.message != "Action succeeded" || !actionMsg.refresh || actionMsg.err != nil {
		t.Errorf("unexpected actionFinishedMsg: %+v", actionMsg)
	}
}
