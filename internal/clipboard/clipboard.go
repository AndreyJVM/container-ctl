package clipboard

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/aymanbagabas/go-osc52/v2"
)

var (
	writerMu     sync.Mutex
	outputWriter io.Writer = os.Stderr
	testLockMu   sync.Mutex
)

// SetOutputWriter sets the writer used for sending OSC 52 sequences thread-safely.
func SetOutputWriter(w io.Writer) {
	writerMu.Lock()
	defer writerMu.Unlock()
	outputWriter = w
}

// GetOutputWriter returns the current writer used for sending OSC 52 sequences thread-safely.
func GetOutputWriter() io.Writer {
	writerMu.Lock()
	defer writerMu.Unlock()
	return outputWriter
}

// TestLock serializes tests that mutate the package-level outputWriter.
func TestLock() func() {
	testLockMu.Lock()
	return testLockMu.Unlock
}

// Write copies the given text to the system clipboard via terminal OSC 52 escape sequence.
// This works transparently across local terminals, tmux, and SSH sessions.
func Write(text string) error {
	if text == "" {
		return nil
	}
	seq := osc52.New(text)
	writerMu.Lock()
	defer writerMu.Unlock()
	_, err := fmt.Fprint(outputWriter, seq)
	return err
}
