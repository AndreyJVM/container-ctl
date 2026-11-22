package tui

import (
	"strings"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/tui/modals"
	"github.com/AndreyJVM/container-ctl/internal/tui/theme"
	"github.com/AndreyJVM/container-ctl/internal/tui/views"
	"github.com/charmbracelet/lipgloss"
)

// View renders the complete TUI interface for the current model state.
func (m Model) View() string {
	var b strings.Builder

	// 1. Header
	b.WriteString(views.RenderHeader(views.HeaderContext{
		Version:               m.version,
		LastUpdate:            m.lastUpdate,
		DockerInfo:            m.dockerInfo,
		FormattedDockerStatus: m.formatDockerStatus(m.dockerInfo.Status),
	}) + "\n\n")

	// 2. Tabs
	visible := m.Registry().VisibleModules(m.moduleContext())
	tabs := make([]string, len(visible))
	for i, v := range visible {
		tabs[i] = v.TabLabel
	}
	b.WriteString(views.RenderTabs(tabs, int(m.activeTab), m.width) + "\n")

	// Action message toast
	if m.actionMessage != "" && time.Now().Before(m.actionMessageExpiry) {
		b.WriteString(theme.YellowStyle.Render("[!] "+m.actionMessage) + "\n\n")
	}

	// 3. Tab contents or active modal
	var content string
	switch m.activeModal {
	case ModalLogs:
		content = m.renderLogsModal()
	case ModalInspect:
		content = m.renderInspectModal()
	case ModalComposeDown:
		content = m.renderComposeDownModal()
	case ModalRemoveContainer:
		content = m.renderRemoveContainerModal()
	case ModalRemoveImage:
		content = m.renderRemoveImageModal()
	case ModalExec:
		content = m.renderExecModal()
	case ModalPrune:
		content = m.renderPruneModal()
	default:
		if m.showHelp {
			content = "\n" + m.renderHelpBox() + "\n"
		} else {
			switch m.activeTab {
			case TabDocker:
				content = m.renderDockerTab()
			case TabImages:
				content = m.renderImagesTab()
			case TabCompose:
				content = m.renderComposeTab()
			case TabNetworks:
				content = m.renderNetworksTab()
			case TabVolumes:
				content = m.renderVolumesTab()
			case TabSwarm:
				content = m.renderSwarmTab()
			}
		}
	}
	if m.embeddedShell != nil && m.activeModal == ModalNone && !m.showHelp {
		content = m.renderEmbeddedShellLayout(content)
	}
	b.WriteString(content)

	// 4. Footer
	b.WriteString(views.RenderFooter(views.FooterContext{
		ActiveTab:    int(m.activeTab),
		ActiveModal:  string(m.activeModal),
		PruneConfirm: m.pruneConfirm,
		IsFiltering:  m.isFiltering,
		FilterQuery:  m.filterQuery,
		Width:        m.width,
		ShellActive:  m.embeddedShell != nil,
		ShellFocused: m.shellFocused,
		SwarmVisible: m.IsSwarmVisible(),
	}))

	return b.String()
}

func (m Model) renderEmbeddedShellLayout(content string) string {
	panelWidth := max(embeddedShellMinWidth, m.width)
	panelHeight := max(embeddedShellMinHeight, m.height-8)
	if m.width >= embeddedShellWideLayoutMin {
		panelWidth = m.width / 2
	}

	terminal := views.RenderEmbeddedShell(views.EmbeddedShellContext{
		ContainerName: m.shellContainerName,
		Output:        m.embeddedShell.Render(),
		Width:         panelWidth,
		Height:        panelHeight,
		Focused:       m.shellFocused,
	})
	if m.width < embeddedShellWideLayoutMin {
		return terminal
	}

	leftWidth := max(1, m.width-panelWidth-1)
	left := lipgloss.NewStyle().
		Width(leftWidth).
		MaxWidth(leftWidth).
		MaxHeight(panelHeight).
		Render(content)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, " ", terminal)
}

func (m Model) renderDockerTab() string {
	containers := m.getFilteredContainers()
	sortNames := []string{
		"Default",
		"CPU ↓",
		"Memory ↓",
		"Name ↑",
		"Status",
	}

	return views.RenderDocker(views.DockerContext{
		DockerInfo:           m.dockerInfo,
		Containers:           containers,
		TotalContainersCount: len(m.dockerInfo.Containers),
		CursorIdx:            m.cursorIdx,
		IsFiltering:          m.isFiltering,
		FilterQuery:          m.filterQuery,
		SortMode:             m.containerSortMode,
		SortNames:            sortNames,
		Width:                m.width,
	})
}

func (m Model) renderImagesTab() string {
	images := m.getFilteredImages()
	return views.RenderImages(views.ImagesContext{
		Images:           images,
		AllImages:        m.images,
		TotalImagesCount: len(m.images),
		CursorIdx:        m.imageCursorIdx,
		Error:            m.imageErr,
		IsFiltering:      m.isFiltering,
		FilterQuery:      m.filterQuery,
		OnlyUnused:       m.onlyUnusedImages,
		Width:            m.width,
	})
}

func (m Model) renderComposeTab() string {
	compose := m.getFilteredCompose()
	return views.RenderCompose(views.ComposeContext{
		Projects:           compose,
		TotalProjectsCount: len(m.compose),
		CursorIdx:          m.composeCursorIdx,
		FormatStatus:       m.formatComposeStatus,
		IsFiltering:        m.isFiltering,
		FilterQuery:        m.filterQuery,
		Width:              m.width,
	})
}

func (m Model) renderNetworksTab() string {
	networks := m.getFilteredNetworks()
	return views.RenderNetworks(views.NetworksContext{
		Networks:           networks,
		TotalNetworksCount: len(m.networks),
		CursorIdx:          m.networkCursorIdx,
		Error:              m.networkErr,
		IsFiltering:        m.isFiltering,
		FilterQuery:        m.filterQuery,
		Width:              m.width,
	})
}

func (m Model) renderVolumesTab() string {
	volumes := m.getFilteredVolumes()
	return views.RenderVolumes(views.VolumesContext{
		Volumes:           volumes,
		TotalVolumesCount: len(m.volumes),
		CursorIdx:         m.volumeCursorIdx,
		Error:             m.volumeErr,
		IsFiltering:       m.isFiltering,
		FilterQuery:       m.filterQuery,
		Width:             m.width,
	})
}

func (m Model) renderSwarmTab() string {
	return views.RenderSwarm(views.SwarmContext{
		Swarm:        m.swarm,
		CursorIdx:    m.swarmCursorIdx,
		FormatStatus: m.formatSwarmStatus,
		Width:        m.width,
	})
}

func (m Model) renderLogsModal() string {
	filterActive := "false"
	if m.logsFiltering {
		filterActive = "true"
	}
	return modals.RenderLogs(m.logsLines, m.logsScroll, m.logsTargetName, m.logsTargetID, m.width, m.logsFilterQuery, filterActive)
}

func (m Model) renderInspectModal() string {
	if len(m.inspectCustomLines) > 0 {
		return modals.RenderInspectLines(m.inspectCustomTitle, m.inspectCustomLines, m.inspectScroll)
	}
	return modals.RenderInspect(m.inspectTarget, m.inspectScroll)
}

func (m Model) renderPruneModal() string {
	return modals.RenderPrune(m.pruneUsage, m.pruneUsageErr, m.pruneConfirm)
}

func (m Model) renderComposeDownModal() string {
	return modals.RenderComposeDown(m.composeDownTarget)
}

func (m Model) renderRemoveContainerModal() string {
	return modals.RenderRemoveContainer(m.removeContainerTargetName, m.removeContainerTargetID)
}

func (m Model) renderRemoveImageModal() string {
	return modals.RenderRemoveImage(m.removeImageTargetName, m.removeImageTargetID, m.removeImageTargetInUse)
}

func (m Model) renderExecModal() string {
	return modals.RenderExec(modals.ExecModalContext{
		ContainerName: m.execTargetName,
		ContainerID:   m.execTargetID,
		CursorIdx:     m.execCursorIdx,
		IsCustomInput: m.execIsCustomInput,
		CustomCmd:     m.execCustomCmd,
	})
}

func (m Model) renderHelpBox() string {
	return modals.RenderHelp(m.refreshInterval, m.IsSwarmVisible())
}
