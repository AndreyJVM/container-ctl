package tui

import (
	"fmt"
	"os/exec"
	"sync"
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

// String returns the human-readable display name of the tab.
func (t TabID) String() string {
	switch t {
	case TabDocker:
		return "Containers"
	case TabImages:
		return "Images"
	case TabCompose:
		return "Compose"
	case TabNetworks:
		return "Networks"
	case TabVolumes:
		return "Volumes"
	case TabSwarm:
		return "Swarm"
	default:
		return fmt.Sprintf("Tab(%d)", int(t))
	}
}

// IsValid reports whether the tab ID corresponds to a known dashboard tab.
func (t TabID) IsValid() bool {
	return t >= TabDocker && t <= TabSwarm
}

// ModalType represents an active modal dialog.
type ModalType string

// String returns the string representation of the modal type.
func (m ModalType) String() string {
	return string(m)
}

// IsOpen reports whether an overlay modal is currently active.
func (m ModalType) IsOpen() bool {
	return m != ModalNone
}

// EmbeddedShellSession defines the interface for an active PTY shell session.
type EmbeddedShellSession interface {
	Read() ([]byte, error)
	Write(data []byte) error
	Feed(data []byte) error
	Render() string
	Resize(width, height int) error
	Wait() error
	Close() error
}

type embeddedShellSession = EmbeddedShellSession

var (
	embeddedShellMu    sync.RWMutex
	startEmbeddedShell = func(cmd *exec.Cmd, width, height int) (embeddedShellSession, error) {
		return shell.Start(cmd, width, height)
	}
)

func runStartEmbeddedShell(cmd *exec.Cmd, width, height int) (embeddedShellSession, error) {
	embeddedShellMu.RLock()
	fn := startEmbeddedShell
	embeddedShellMu.RUnlock()
	return fn(cmd, width, height)
}

// SetStartEmbeddedShellForTesting overrides the shell launcher for testing and returns a cleanup func.
func SetStartEmbeddedShellForTesting(fn func(cmd *exec.Cmd, width, height int) (EmbeddedShellSession, error)) func() {
	embeddedShellMu.Lock()
	prev := startEmbeddedShell
	startEmbeddedShell = fn
	embeddedShellMu.Unlock()
	return func() {
		embeddedShellMu.Lock()
		startEmbeddedShell = prev
		embeddedShellMu.Unlock()
	}
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
	// ModalRemoveNetwork displays confirmation for removing a Docker network.
	ModalRemoveNetwork ModalType = "remove_network"
	// ModalRemoveVolume displays confirmation for removing a Docker volume.
	ModalRemoveVolume ModalType = "remove_volume"
	// ModalExec displays the interactive container shell launcher.
	ModalExec ModalType = "exec"
	// ModalSwarmScale displays interactive service replica scaling.
	ModalSwarmScale ModalType = "swarm_scale"
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

// CommonState holds general application lifecycle and window dimensions.
type CommonState struct {
	version             string
	activeTab           TabID
	refreshInterval     time.Duration
	showHelp            bool
	width               int
	height              int
	actionMessage       string
	actionMessageExpiry time.Time
	registry            *module.Registry
}

// NavState holds cursor positions, scroll offsets, sorting, and search filter queries per tab.
type NavState struct {
	cursorIdx          int
	imageCursorIdx     int
	composeCursorIdx   int
	networkCursorIdx   int
	volumeCursorIdx    int
	swarmCursorIdx     int
	containerSortMode  int
	isFiltering        bool
	filterQuery        string
	showSwarm          bool
	hideDetails        bool
	onlyUnusedImages   bool
	selectedContainers map[string]struct{}
	selectedImages     map[string]struct{}
	selectedNetworks   map[string]struct{}
	selectedVolumes    map[string]struct{}
}

// ShellState holds the embedded PTY shell terminal session.
type ShellState struct {
	embeddedShell      embeddedShellSession
	shellFocused       bool
	shellContainerName string
}

// ModalState holds state for active overlays and confirmation dialogs.
type ModalState struct {
	activeModal ModalType

	// Exec modal
	execTargetID      string
	execTargetName    string
	execCursorIdx     int
	execIsCustomInput bool
	execCustomCmd     string

	// Logs modal
	logsTargetName  string
	logsTargetID    string
	logsLines       []string
	logsScroll      int
	logsFilterQuery string
	logsFiltering   bool

	// Inspect modal
	inspectTarget      monitor.ContainerDetail
	inspectCustomTitle string
	inspectCustomLines []string
	inspectCopyID      string
	inspectScroll      int

	// Compose down modal
	composeDownTarget string

	// Remove container modal
	removeContainerTargetID   string
	removeContainerTargetName string
	removeContainerBatch      []monitor.ContainerItem

	// Remove image modal
	removeImageTargetID    string
	removeImageTargetName  string
	removeImageTargetInUse bool
	removeImageBatch       []monitor.DockerImage

	// Remove network modal
	removeNetworkTargetID        string
	removeNetworkTargetName      string
	removeNetworkContainersCount int
	removeNetworkBatch           []monitor.DockerNetwork

	// Remove volume modal
	removeVolumeTargetName  string
	removeVolumeTargetInUse bool
	removeVolumeBatch       []monitor.DockerVolume

	// Prune modal
	pruneUsage    monitor.DockerDiskUsage
	pruneUsageErr string
	pruneConfirm  int // 0 - menu, 1 - containers, 2 - images, 3 - volumes, 4 - all

	// Swarm scaling modal
	swarmScaleTargetID   string
	swarmScaleTargetName string
	swarmScaleCurrent    int
	swarmScaleDesired    int
	swarmScaleTyped      bool
}

// DataState holds cached domain data fetched asynchronously from the Docker daemon.
type DataState struct {
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
}

// Model contains the complete TUI application state composed of dedicated state domains.
type Model struct {
	CommonState
	NavState
	ShellState
	ModalState
	DataState
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

// ModelOption configures a TUI Model instance during construction.
type ModelOption func(*Model)

// WithShowSwarm sets whether the Swarm tab should be forced to be visible.
func WithShowSwarm(show bool) ModelOption {
	return func(m *Model) {
		m.showSwarm = show
	}
}

// WithRegistry sets a custom module registry for the dashboard.
func WithRegistry(r *module.Registry) ModelOption {
	return func(m *Model) {
		m.SetRegistry(r)
	}
}

// WithOnlyUnusedImages configures the initial image filtering to show only unused images.
func WithOnlyUnusedImages(val bool) ModelOption {
	return func(m *Model) {
		m.onlyUnusedImages = val
	}
}

// WithSortMode sets the initial container sort mode.
func WithSortMode(mode int) ModelOption {
	return func(m *Model) {
		m.containerSortMode = mode
	}
}

// WithHideDetails sets whether the lower details pane should be hidden initially.
func WithHideDetails(hide bool) ModelOption {
	return func(m *Model) {
		m.hideDetails = hide
	}
}

// NewModelWithOptions creates a TUI model with a refresh interval, initial tab, version, and optional functional options.
// Startup is completely non-blocking: data is fetched asynchronously via Bubble Tea's Init() command.
func NewModelWithOptions(refreshInterval time.Duration, initialTab int, version string, opts ...ModelOption) Model {
	tab := TabID(initialTab)
	if tab < TabDocker || tab > TabCount {
		tab = TabDocker
	}
	if refreshInterval <= 0 {
		refreshInterval = 2 * time.Second
	}
	var m Model
	m.version = version
	m.activeTab = tab
	m.refreshInterval = refreshInterval
	m.showSwarm = tab == TabSwarm
	m.selectedContainers = make(map[string]struct{})
	m.selectedImages = make(map[string]struct{})
	m.selectedNetworks = make(map[string]struct{})
	m.selectedVolumes = make(map[string]struct{})
	m.registry = module.DefaultRegistry()

	for _, opt := range opts {
		if opt != nil {
			opt(&m)
		}
	}

	return m
}

// NewModel creates a TUI model with a refresh interval and initial tab.
// Preserves backward compatibility while delegating to non-blocking NewModelWithOptions.
func NewModel(refreshInterval time.Duration, initialTab int, version string, showSwarm ...bool) Model {
	var opts []ModelOption
	if len(showSwarm) > 0 && showSwarm[0] {
		opts = append(opts, WithShowSwarm(true))
	}
	return NewModelWithOptions(refreshInterval, initialTab, version, opts...)
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
	m.pruneSelections()

	if m.registry != nil {
		currID, hasID := ModuleIDFromTabID(m.activeTab)
		if hasID && !m.registry.IsModuleVisible(currID, m.moduleContext()) {
			visible := m.registry.VisibleModules(m.moduleContext())
			if len(visible) > 0 {
				if tabID, ok := TabIDFromModuleID(visible[0].ID); ok {
					m.activeTab = tabID
				}
			}
			if currID == module.IDSwarm {
				m.showToast("Host is not part of an active Swarm cluster.")
			}
		}
	} else if !m.IsSwarmVisible() && m.activeTab == TabSwarm {
		m.activeTab = TabDocker
		m.showToast("Host is not part of an active Swarm cluster.")
	}
}

func (m *Model) applyFetchedData(d dataFetchedMsg) {
	m.applyActionsData(d)
}

var fetchDockerData = fetchAllData

func (m *Model) fetchAllData() {
	if m.registry != nil {
		ctx := m.moduleContext()
		includeImages := m.registry.IsModuleVisible(module.IDImages, ctx)
		includeNetworks := m.registry.IsModuleVisible(module.IDNetworks, ctx)
		includeVolumes := m.registry.IsModuleVisible(module.IDVolumes, ctx)
		includeSwarm := m.registry.IsModuleVisible(module.IDSwarm, ctx)
		m.applyActionsData(actions.FetchAllDataFiltered(includeImages, includeNetworks, includeVolumes, includeSwarm))
		return
	}
	m.applyFetchedData(fetchDockerData())
}

// FetchDataCmd returns a tea.Cmd that queries Docker resources for only the active and visible modules.
func (m Model) FetchDataCmd() tea.Cmd {
	ctx := m.moduleContext()
	reg := m.Registry()
	includeImages := reg.IsModuleVisible(module.IDImages, ctx)
	includeNetworks := reg.IsModuleVisible(module.IDNetworks, ctx)
	includeVolumes := reg.IsModuleVisible(module.IDVolumes, ctx)
	includeSwarm := reg.IsModuleVisible(module.IDSwarm, ctx)

	return func() tea.Msg {
		return actions.FetchAllDataFiltered(includeImages, includeNetworks, includeVolumes, includeSwarm)
	}
}

// Init starts the periodic ticker and asynchronously triggers the initial data fetch.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(m.refreshInterval),
		m.FetchDataCmd(),
	)
}

func (m *Model) showToast(msg string) {
	m.actionMessage = msg
	m.actionMessageExpiry = time.Now().Add(4 * time.Second)
}

// ShowToast displays an ephemeral notification message in the dashboard.
func (m *Model) ShowToast(msg string) {
	m.showToast(msg)
}

func (m *Model) copyToClipboard(text string) {
	if err := clipboard.Write(text); err != nil {
		m.showToast(fmt.Sprintf("Clipboard error: %v", err))
	} else {
		m.showToast(fmt.Sprintf("Copied to clipboard: %s", text))
	}
}

// CopyToClipboard writes text to the system clipboard and notifies the user via a toast.
func (m *Model) CopyToClipboard(text string) {
	m.copyToClipboard(text)
}

func formatFetchError(err error) string {
	return actions.FormatFetchError(err)
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

// closeModal dismisses the active modal dialog and cleans ephemeral input state and allocations.
func (m *Model) closeModal() {
	m.activeModal = ModalNone
	m.execIsCustomInput = false
	m.execCustomCmd = ""
	m.execTargetID = ""
	m.execTargetName = ""
	m.execCursorIdx = 0

	m.logsFiltering = false
	m.logsFilterQuery = ""
	m.logsTargetName = ""
	m.logsTargetID = ""
	m.logsLines = nil
	m.logsScroll = 0

	m.inspectTarget = monitor.ContainerDetail{}
	m.inspectCustomTitle = ""
	m.inspectCustomLines = nil
	m.inspectCopyID = ""
	m.inspectScroll = 0

	m.composeDownTarget = ""

	m.removeContainerTargetID = ""
	m.removeContainerTargetName = ""
	m.removeContainerBatch = nil

	m.removeImageTargetID = ""
	m.removeImageTargetName = ""
	m.removeImageTargetInUse = false
	m.removeImageBatch = nil

	m.removeNetworkTargetID = ""
	m.removeNetworkTargetName = ""
	m.removeNetworkContainersCount = 0
	m.removeNetworkBatch = nil

	m.removeVolumeTargetName = ""
	m.removeVolumeTargetInUse = false
	m.removeVolumeBatch = nil

	m.pruneUsage = monitor.DockerDiskUsage{}
	m.pruneUsageErr = ""
	m.pruneConfirm = 0

	m.swarmScaleTargetID = ""
	m.swarmScaleTargetName = ""
	m.swarmScaleCurrent = 0
	m.swarmScaleDesired = 0
	m.swarmScaleTyped = false
}

// OpenModal activates the specified modal dialog.
func (m *Model) OpenModal(modal ModalType) {
	m.openModal(modal)
}

// CloseModal dismisses the active modal dialog.
func (m *Model) CloseModal() {
	m.closeModal()
}

// IsModalOpen reports whether an overlay modal dialog is currently active.
func (m Model) IsModalOpen() bool {
	return m.activeModal.IsOpen()
}

// IsHelpOpen reports whether the help overlay window is currently active.
func (m Model) IsHelpOpen() bool {
	return m.showHelp
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

// ToggleDetails toggles visibility of the lower details card.
func (m *Model) ToggleDetails() {
	m.hideDetails = !m.hideDetails
}

// SetHideDetails explicitly sets visibility of the lower details card.
func (m *Model) SetHideDetails(hide bool) {
	m.hideDetails = hide
}
