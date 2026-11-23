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

type embeddedShellOutputMsg struct {
	session embeddedShellSession
	data    []byte
	err     error
}

type embeddedShellExitedMsg struct {
	session embeddedShellSession
	err     error
}

// dataFetchedMsg delivers freshly gathered metrics and container lists across all domains.
type dataFetchedMsg = actions.DataFetchedMsg

// actionFinishedMsg is returned upon completion of an async container or compose operation.
type actionFinishedMsg = actions.FinishedMsg
