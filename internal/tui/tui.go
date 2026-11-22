package tui

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/clipboard"
	"github.com/AndreyJVM/container-ctl/internal/module"
	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/tui/actions"
	"github.com/AndreyJVM/container-ctl/internal/tui/shell"
	tea "github.com/charmbracelet/bubbletea"
)

// TabID represents an active tab in the dashboard.
type TabID int

const (
	// TabDocker displays Docker containers, statuses, metrics, and actions.
	TabDocker TabID = 1
	// TabImages displays Docker images, tags, sizes, and actions.
	TabImages TabID = 2
	// TabCompose displays Docker Compose projects and aggregated service statuses.
	TabCompose TabID = 3
	// TabNetworks displays Docker networks and their configuration.
	TabNetworks TabID = 4
	// TabVolumes displays Docker volumes and their usage details.
	TabVolumes TabID = 5
	// TabSwarm displays Swarm cluster nodes and services.
	TabSwarm TabID = 6
	// TabCount represents the total number of primary tabs.
	TabCount = 6
)

// ModalType represents an active modal dialog.
type ModalType string

type embeddedShellSession interface {
	Read() ([]byte, error)
	Write(data []byte) error
	Feed(data []byte) error
	Render() string
	Resize(width, height int) error
	Wait() error
	Close() error
}

var startEmbeddedShell = func(cmd *exec.Cmd, width, height int) (embeddedShellSession, error) {
	return shell.Start(cmd, width, height)
}

const (
	// ModalNone indicates no modal is active.
	ModalNone ModalType = ""
	// ModalLogs displays container or service logs.
	ModalLogs ModalType = "logs"
	// ModalInspect displays detailed container inspection data.
	ModalInspect ModalType = "inspect"
	// ModalComposeDown displays confirmation for docker compose down.
	ModalComposeDown ModalType = "compose_down"
	// ModalPrune displays Docker system prune options and confirmations.
	ModalPrune ModalType = "prune"
	// ModalRemoveContainer displays confirmation for removing a container.
	ModalRemoveContainer ModalType = "remove_container"
	// ModalRemoveImage displays confirmation for removing a Docker image.
	ModalRemoveImage ModalType = "remove_image"
	// ModalExec displays the interactive container shell launcher.
	ModalExec ModalType = "exec"
)

const (
	SortDefault = iota
	SortCPU
	SortMem
	SortName
	SortStatus
	sortModeCount
)

func tickCmd(d time.Duration) tea.Cmd {
	if d <= 0 {
		d = 2 * time.Second
	}
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Model contains the TUI state.
type Model struct {
	version         string
	activeTab       TabID
	refreshInterval time.Duration
	showHelp        bool
	width           int
	height          int

	// Navigation and modal state.
	cursorIdx           int
	imageCursorIdx      int
	composeCursorIdx    int
	networkCursorIdx    int
	volumeCursorIdx     int
	swarmCursorIdx      int
	activeModal         ModalType
	execTargetID        string
	execTargetName      string
	execCursorIdx       int
	execIsCustomInput   bool
	execCustomCmd       string
	embeddedShell       embeddedShellSession
	shellFocused        bool
	shellContainerName  string
	logsTargetName      string
	logsTargetID        string
	logsLines           []string
	logsScroll          int
	logsFilterQuery     string
	logsFiltering       bool
	inspectTarget       monitor.ContainerDetail
	inspectCustomTitle  string
	inspectCustomLines  []string
	inspectCopyID       string
	inspectScroll       int
	actionMessage       string
	actionMessageExpiry time.Time

	// Compose down modal.
	composeDownTarget string

	// Remove container modal.
	removeContainerTargetID   string
	removeContainerTargetName string

	// Remove image modal.
	removeImageTargetID    string
	removeImageTargetName  string
	removeImageTargetInUse bool

	// Prune modal.
	pruneUsage    monitor.DockerDiskUsage
	pruneUsageErr string
	pruneConfirm  int // 0 - menu, 1 - containers, 2 - images, 3 - volumes, 4 - all

	// Container sorting: default, CPU desc, memory desc, name asc, or status.
	containerSortMode int

	// Container search and filtering.
	isFiltering bool
	filterQuery string

	dockerInfo monitor.DockerSummary
	images     []monitor.DockerImage
	compose    []monitor.ComposeProject
	networks   []monitor.DockerNetwork
	volumes    []monitor.DockerVolume
	swarm      monitor.SwarmSummary
	imageErr   string
	networkErr string
	volumeErr  string
	lastUpdate string

	// showSwarm forces the Swarm tab to remain visible even when inactive.
	showSwarm bool

	// onlyUnusedImages filters the image list to only unreferenced/reclaimable images.
	onlyUnusedImages bool

	// registry manages active modules and custom tab ordering.
	registry *module.Registry
}

// SetRegistry configures the module registry for the TUI model.
func (m *Model) SetRegistry(r *module.Registry) {
	if r != nil {
		m.registry = r
	}
}

// Registry returns the active module registry, initializing a default if nil.
func (m *Model) Registry() *module.Registry {
	if m.registry == nil {
		m.registry = module.DefaultRegistry()
	}
	return m.registry
}

func (m Model) moduleContext() module.Context {
	return module.Context{
		ShowSwarm:     m.showSwarm,
		IsSwarmActive: m.swarm.IsActive,
	}
}

// NewModel creates a TUI model with a refresh interval and initial tab.
func NewModel(refreshInterval time.Duration, initialTab int, version string, showSwarm ...bool) Model {
	tab := TabID(initialTab)
	if tab < TabDocker || tab > TabCount {
		tab = TabDocker
	}
	if refreshInterval <= 0 {
		refreshInterval = 2 * time.Second
	}
	forceSwarm := tab == TabSwarm
	if len(showSwarm) > 0 && showSwarm[0] {
		forceSwarm = true
	}
	m := Model{
		version:         version,
		activeTab:       tab,
		refreshInterval: refreshInterval,
		showSwarm:       forceSwarm,
		registry:        module.DefaultRegistry(),
	}
	m.fetchAllData()
	return m
}

func (m *Model) applyActionsData(d actions.DataFetchedMsg) {
	m.dockerInfo = d.DockerInfo
	m.images = d.Images
	m.compose = d.Compose
	m.networks = d.Networks
	m.volumes = d.Volumes
	m.swarm = d.Swarm
	m.imageErr = d.ImageErr
	m.networkErr = d.NetworkErr
	m.volumeErr = d.VolumeErr
	m.lastUpdate = d.Timestamp

	m.clampCursor()
	if m.composeCursorIdx >= len(m.compose) {
		m.composeCursorIdx = max(0, len(m.compose)-1)
	}
	if m.swarmCursorIdx >= len(m.swarm.Services) {
		m.swarmCursorIdx = max(0, len(m.swarm.Services)-1)
	}

	if !m.IsSwarmVisible() && m.activeTab == TabSwarm {
		m.activeTab = TabDocker
		m.showToast("Host is not part of an active Swarm cluster.")
	}
}

func (m *Model) applyFetchedData(d dataFetchedMsg) {
	m.applyActionsData(actions.DataFetchedMsg{
		DockerInfo: d.dockerInfo,
		Images:     d.images,
		Compose:    d.compose,
		Networks:   d.networks,
		Volumes:    d.volumes,
		Swarm:      d.swarm,
		ImageErr:   d.imageErr,
		NetworkErr: d.networkErr,
		VolumeErr:  d.volumeErr,
		Timestamp:  d.timestamp,
	})
}

var fetchDockerData = fetchAllData

func (m *Model) fetchAllData() {
	m.applyFetchedData(fetchDockerData())
}

func (m Model) Init() tea.Cmd {
	return tickCmd(m.refreshInterval)
}

func (m *Model) showToast(msg string) {
	m.actionMessage = msg
	m.actionMessageExpiry = time.Now().Add(4 * time.Second)
}

func (m *Model) copyToClipboard(text string) {
	if err := clipboard.Write(text); err != nil {
		m.showToast(fmt.Sprintf("Clipboard error: %v", err))
	} else {
		m.showToast(fmt.Sprintf("Copied to clipboard: %s", text))
	}
}

func formatFetchError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// openModal activates the specified modal dialog.
func (m *Model) openModal(modal ModalType) {
	m.activeModal = modal
	if modal == ModalLogs {
		m.logsFiltering = false
		m.logsFilterQuery = ""
	}
	if modal == ModalInspect {
		m.inspectScroll = 0
	}
}

// closeModal dismisses the active modal dialog and cleans ephemeral input state.
func (m *Model) closeModal() {
	m.activeModal = ModalNone
	m.execIsCustomInput = false
	m.execCustomCmd = ""
	m.logsFiltering = false
	m.logsFilterQuery = ""
	m.inspectScroll = 0
}

// OpenModal activates the specified modal dialog.
func (m *Model) OpenModal(modal ModalType) {
	m.openModal(modal)
}

// CloseModal dismisses the active modal dialog.
func (m *Model) CloseModal() {
	m.closeModal()
}

// ActiveModal returns the current active modal dialog.
func (m Model) ActiveModal() ModalType {
	return m.activeModal
}

// ActiveTab returns the current active tab.
func (m Model) ActiveTab() TabID {
	return m.activeTab
}

// SetTab switches the active tab.
func (m *Model) SetTab(tab TabID) {
	m.setTab(tab)
}

// SetLogsForTesting sets logs content on the model for testing purposes.
func (m *Model) SetLogsForTesting(targetName, targetID string, lines []string) {
	m.logsTargetName = targetName
	m.logsTargetID = targetID
	m.logsLines = lines
}

// SetInspectForTesting sets inspect target on the model for testing purposes.
func (m *Model) SetInspectForTesting(detail monitor.ContainerDetail) {
	m.inspectTarget = detail
}

// SetCursorForTesting sets cursor indices across tabs for testing purposes.
func (m *Model) SetCursorForTesting(idx int) {
	m.cursorIdx = idx
	m.imageCursorIdx = idx
	m.composeCursorIdx = idx
	m.networkCursorIdx = idx
	m.volumeCursorIdx = idx
	m.swarmCursorIdx = idx
}
