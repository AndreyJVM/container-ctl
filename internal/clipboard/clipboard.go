package clipboard

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/aymanbagabas/go-osc52/v2"
)

var (
	writerMu     sync.RWMutex
	outputWriter io.Writer = os.Stderr
)

// SetOutputWriter sets the writer used for sending OSC 52 sequences thread-safely.
func SetOutputWriter(w io.Writer) {
	writerMu.Lock()
	defer writerMu.Unlock()
	outputWriter = w
}

// GetOutputWriter returns the current writer used for sending OSC 52 sequences thread-safely.
func GetOutputWriter() io.Writer {
	writerMu.RLock()
	defer writerMu.RUnlock()
	return outputWriter
}

// Write copies the given text to the system clipboard via terminal OSC 52 escape sequence.
// This works transparently across local terminals, tmux, and SSH sessions.
func Write(text string) error {
	if text == "" {
		return nil
	}
	seq := osc52.New(text)
	writerMu.RLock()
	w := outputWriter
	writerMu.RUnlock()
	_, err := fmt.Fprint(w, seq)
	return err
}
