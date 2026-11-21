package tui

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/clipboard"
	"github.com/AndreyJVM/container-ctl/internal/monitor"
)

func TestNetworksInspectAndCopy(t *testing.T) {
	origWriter := clipboard.OutputWriter
	var buf bytes.Buffer
	clipboard.OutputWriter = &buf
	defer func() { clipboard.OutputWriter = origWriter }()

	m := NewModel(2*time.Second, int(TabNetworks), "v1.0.0")
	m.networks = []monitor.DockerNetwork{
		{
			ID:     "net123",
			Name:   "production-bridge",
			Driver: "bridge",
			Scope:  "local",
		},
	}
	m.networkCursorIdx = 0

	// 1. Inspect network
	mInspect, _ := m.handleNetworksInspect()
	if mInspect.activeModal != ModalInspect {
		t.Errorf("expected activeModal %s, got %s", ModalInspect, mInspect.activeModal)
	}
	if !strings.Contains(mInspect.inspectCustomTitle, "production-bridge") {
		t.Errorf("expected title with network name, got %s", mInspect.inspectCustomTitle)
	}
	if mInspect.inspectCopyID != "net123" {
		t.Errorf("expected inspectCopyID net123, got %s", mInspect.inspectCopyID)
	}

	// 2. Copy network name
	buf.Reset()
	mCopy, _ := m.handleNetworksCopy()
	if !strings.Contains(mCopy.actionMessage, "production-bridge") {
		t.Errorf("expected toast with network name, got %s", mCopy.actionMessage)
	}
}

func TestVolumesInspectAndCopy(t *testing.T) {
	origWriter := clipboard.OutputWriter
	var buf bytes.Buffer
	clipboard.OutputWriter = &buf
	defer func() { clipboard.OutputWriter = origWriter }()

	m := NewModel(2*time.Second, int(TabVolumes), "v1.0.0")
	m.volumes = []monitor.DockerVolume{
		{
			Name:       "db_data",
			Driver:     "local",
			Mountpoint: "/var/lib/docker/volumes/db_data/_data",
		},
	}
	m.volumeCursorIdx = 0

	// 1. Inspect volume
	mInspect, _ := m.handleVolumesInspect()
	if mInspect.activeModal != ModalInspect {
		t.Errorf("expected activeModal %s, got %s", ModalInspect, mInspect.activeModal)
	}
	if !strings.Contains(mInspect.inspectCustomTitle, "db_data") {
		t.Errorf("expected title with volume name, got %s", mInspect.inspectCustomTitle)
	}
	if mInspect.inspectCopyID != "db_data" {
		t.Errorf("expected inspectCopyID db_data, got %s", mInspect.inspectCopyID)
	}

	// 2. Copy volume name
	buf.Reset()
	mCopy, _ := m.handleVolumesCopy()
	if !strings.Contains(mCopy.actionMessage, "db_data") {
		t.Errorf("expected toast with volume name, got %s", mCopy.actionMessage)
	}
}

func TestSwarmLogsAndCopy(t *testing.T) {
	origWriter := clipboard.OutputWriter
	var buf bytes.Buffer
	clipboard.OutputWriter = &buf
	defer func() { clipboard.OutputWriter = origWriter }()

	m := NewModel(2*time.Second, int(TabSwarm), "v1.0.0")
	m.swarm = monitor.SwarmSummary{
		IsActive: true,
		Services: []monitor.SwarmService{
			{
				ID:   "svc123",
				Name: "frontend-web",
				Mode: "replicated",
			},
		},
	}
	m.swarmCursorIdx = 0

	// 1. Swarm logs
	mLogs, _ := m.handleSwarmLogs()
	if mLogs.logsTargetID != "svc123" {
		t.Errorf("expected logsTargetID svc123, got %s", mLogs.logsTargetID)
	}

	// 2. Copy swarm service name
	buf.Reset()
	mCopy, _ := m.handleSwarmCopy()
	if !strings.Contains(mCopy.actionMessage, "frontend-web") {
		t.Errorf("expected toast with service name, got %s", mCopy.actionMessage)
	}
}
