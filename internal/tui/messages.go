package tui

import (
	"time"

	"github.com/AndreyJVM/container-ctl/internal/tui/actions"
)

// tickMsg signals a periodic refresh timer tick.
type tickMsg time.Time

// execFinishedMsg is returned when an interactive container exec/attach session ends.
type execFinishedMsg struct {
	containerName string
	err           error
}

// NewExecFinishedMsg constructs an execFinishedMsg for testing.
func NewExecFinishedMsg(containerName string, err error) any {
	return execFinishedMsg{containerName: containerName, err: err}
}

type embeddedShellOutputMsg struct {
	session embeddedShellSession
	data    []byte
	err     error
}

// NewEmbeddedShellOutputMsg constructs an embeddedShellOutputMsg for tests.
func NewEmbeddedShellOutputMsg(session EmbeddedShellSession, data []byte, err error) any {
	return embeddedShellOutputMsg{session: session, data: data, err: err}
}

type embeddedShellExitedMsg struct {
	session embeddedShellSession
	err     error
}

// NewEmbeddedShellExitedMsg constructs an embeddedShellExitedMsg for tests.
func NewEmbeddedShellExitedMsg(session EmbeddedShellSession, err error) any {
	return embeddedShellExitedMsg{session: session, err: err}
}

// dataFetchedMsg delivers freshly gathered metrics and container lists across all domains.
type dataFetchedMsg = actions.DataFetchedMsg

// actionFinishedMsg is returned upon completion of an async container or compose operation.
type actionFinishedMsg = actions.FinishedMsg
