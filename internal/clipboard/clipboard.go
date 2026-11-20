package clipboard

import (
	"fmt"
	"io"
	"os"

	"github.com/aymanbagabas/go-osc52/v2"
)

// OutputWriter is the writer used for sending OSC 52 sequences. Defaults to os.Stderr.
var OutputWriter io.Writer = os.Stderr

// Write copies the given text to the system clipboard via terminal OSC 52 escape sequence.
// This works transparently across local terminals, tmux, and SSH sessions.
func Write(text string) error {
	if text == "" {
		return nil
	}
	seq := osc52.New(text)
	_, err := fmt.Fprint(OutputWriter, seq)
	return err
}
