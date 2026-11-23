package clipboard

import (
	"errors"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/aymanbagabas/go-osc52/v2"
)

// DefaultMaxClipboardSize defines a safe upper bound (100 KB) for terminal OSC 52 sequences
// to prevent overflowing terminal/SSH buffers.
const DefaultMaxClipboardSize = 100 * 1024

// ErrTextTooLarge is returned when the payload exceeds the configured maximum size.
var ErrTextTooLarge = errors.New("clipboard: text exceeds maximum allowed size")

// Service represents a clipboard provider capable of copying text.
type Service interface {
	Write(text string) error
}

// Clipboard manages writing text to system or primary clipboards via OSC 52 sequences.
type Clipboard struct {
	mu      sync.RWMutex
	out     io.Writer
	maxSize int
	mode    osc52.Mode
}

// New creates a new Clipboard instance writing to w.
// If w is nil, io.Discard is used to prevent nil pointer dereferences.
func New(w io.Writer) *Clipboard {
	if w == nil {
		w = io.Discard
	}
	return &Clipboard{
		out:     w,
		maxSize: DefaultMaxClipboardSize,
		mode:    osc52.DefaultMode,
	}
}

// SetWriter sets the destination writer thread-safely.
// If w is nil, io.Discard is used.
func (c *Clipboard) SetWriter(w io.Writer) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if w == nil {
		w = io.Discard
	}
	c.out = w
}

// GetWriter returns the current destination writer thread-safely.
func (c *Clipboard) GetWriter() io.Writer {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.out
}

// SetMaxSize configures the maximum text size in bytes (0 or negative disables the limit).
func (c *Clipboard) SetMaxSize(size int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.maxSize = size
}

// GetMaxSize returns the current maximum text size in bytes.
func (c *Clipboard) GetMaxSize() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.maxSize
}

// SetMode sets the escape sequence wrapping mode (e.g. DefaultMode, TmuxMode, ScreenMode).
func (c *Clipboard) SetMode(mode osc52.Mode) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mode = mode
}

// GetMode returns the current escape sequence mode.
func (c *Clipboard) GetMode() osc52.Mode {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.mode
}

// AutoDetect configures the clipboard mode based on environment variables ($TMUX, $TERM).
func (c *Clipboard) AutoDetect() *Clipboard {
	c.mu.Lock()
	defer c.mu.Unlock()
	if os.Getenv("TMUX") != "" {
		c.mode = osc52.TmuxMode
	} else if strings.HasPrefix(os.Getenv("TERM"), "screen") {
		c.mode = osc52.ScreenMode
	} else {
		c.mode = osc52.DefaultMode
	}
	return c
}

// Write copies the given text to the system clipboard via terminal OSC 52 escape sequence.
func (c *Clipboard) Write(text string) error {
	if text == "" {
		return nil
	}

	c.mu.RLock()
	maxSize := c.maxSize
	mode := c.mode
	c.mu.RUnlock()

	if maxSize > 0 && len(text) > maxSize {
		return ErrTextTooLarge
	}

	seq := osc52.New(text)
	if mode != osc52.DefaultMode {
		seq = seq.Mode(mode)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := seq.WriteTo(c.out)
	return err
}

// WritePrimary copies the given text to the primary selection (used for middle-click on X11/Wayland).
func (c *Clipboard) WritePrimary(text string) error {
	if text == "" {
		return nil
	}

	c.mu.RLock()
	maxSize := c.maxSize
	mode := c.mode
	c.mu.RUnlock()

	if maxSize > 0 && len(text) > maxSize {
		return ErrTextTooLarge
	}

	seq := osc52.New(text).Primary()
	if mode != osc52.DefaultMode {
		seq = seq.Mode(mode)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := seq.WriteTo(c.out)
	return err
}

// WriteAll copies the given text to both the system clipboard and primary selection.
func (c *Clipboard) WriteAll(text string) error {
	if err := c.Write(text); err != nil {
		return err
	}
	return c.WritePrimary(text)
}

// -----------------------------------------------------------------------------
// Package-level helpers maintaining full backward compatibility.
// -----------------------------------------------------------------------------

var (
	defaultClipboard = New(os.Stderr)
	testLockMu       sync.Mutex
)

// Default returns the package-level default Clipboard instance.
func Default() *Clipboard {
	return defaultClipboard
}

// SetOutputWriter sets the writer used for sending OSC 52 sequences thread-safely.
func SetOutputWriter(w io.Writer) {
	defaultClipboard.SetWriter(w)
}

// GetOutputWriter returns the current writer used for sending OSC 52 sequences thread-safely.
func GetOutputWriter() io.Writer {
	return defaultClipboard.GetWriter()
}

// TestLock serializes tests that mutate the package-level outputWriter.
func TestLock() func() {
	testLockMu.Lock()
	return testLockMu.Unlock
}

// Write copies the given text to the system clipboard via terminal OSC 52 escape sequence.
// This works transparently across local terminals, tmux, and SSH sessions.
func Write(text string) error {
	return defaultClipboard.Write(text)
}

// WritePrimary copies text to the primary selection via terminal OSC 52 escape sequence.
func WritePrimary(text string) error {
	return defaultClipboard.WritePrimary(text)
}

// WriteAll copies text to both the system clipboard and primary selection.
func WriteAll(text string) error {
	return defaultClipboard.WriteAll(text)
}
