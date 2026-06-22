package common

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Spinner provides an animated spinner for indeterminate operations.
type Spinner struct {
	message  string
	frames   []string
	interval time.Duration
	writer   io.Writer
	stop     chan struct{}
	done     chan struct{}
	mu       sync.Mutex
	active   bool
}

// SpinnerFrames defines available spinner styles.
var SpinnerFrames = struct {
	Dots    []string
	Line    []string
	Circle  []string
	Arrow   []string
	Default []string
}{
	Dots:    []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	Line:    []string{"-", "\\", "|", "/"},
	Circle:  []string{"◐", "◓", "◑", "◒"},
	Arrow:   []string{"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"},
	Default: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
}

// NewSpinner creates a new spinner with the given message.
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message:  message,
		frames:   SpinnerFrames.Default,
		interval: 80 * time.Millisecond,
		writer:   os.Stderr,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// SetWriter sets the output writer.
func (s *Spinner) SetWriter(w io.Writer) *Spinner {
	s.writer = w
	return s
}

// Start starts the spinner animation.
func (s *Spinner) Start() {
	if IsQuiet() {
		return
	}

	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.stop = make(chan struct{})
	s.done = make(chan struct{})
	s.mu.Unlock()

	go func() {
		defer close(s.done)
		frameIdx := 0

		for {
			select {
			case <-s.stop:
				// Clear the spinner line
				_, _ = fmt.Fprintf(s.writer, "\r%s\r", strings.Repeat(" ", len(s.message)+4))
				return
			default:
				frame := s.frames[frameIdx%len(s.frames)]
				_, _ = fmt.Fprintf(s.writer, "\r%s %s", Cyan.Sprint(frame), s.message)
				frameIdx++
				time.Sleep(s.interval)
			}
		}
	}()
}

// Stop stops the spinner animation.
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	s.mu.Unlock()

	close(s.stop)
	<-s.done
}

// StopWithSuccess stops the spinner with a success message.
func (s *Spinner) StopWithSuccess(message string) {
	s.Stop()
	if !IsQuiet() {
		_, _ = fmt.Fprintf(s.writer, "%s %s\n", Green.Sprint("✓"), message)
	}
}

// StopWithError stops the spinner with an error message.
func (s *Spinner) StopWithError(message string) {
	s.Stop()
	if !IsQuiet() {
		_, _ = fmt.Fprintf(s.writer, "%s %s\n", Red.Sprint("✗"), message)
	}
}

// Counter provides a simple counter display.
type Counter struct {
	message string
	count   int
	writer  io.Writer
	mu      sync.Mutex
}

// NewCounter creates a new counter.
func NewCounter(message string) *Counter {
	return &Counter{
		message: message,
		count:   0,
		writer:  os.Stderr,
	}
}

// Increment increments the counter.
func (c *Counter) Increment() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.count++
	if !IsQuiet() {
		_, _ = fmt.Fprintf(c.writer, "\r%s: %d", c.message, c.count)
	}
}

// Finish completes the counter display.
func (c *Counter) Finish() {
	if !IsQuiet() {
		_, _ = fmt.Fprintln(c.writer)
	}
}

// RunWithSpinner executes a function while displaying a spinner.
// It handles spinner start/stop and error propagation.
func RunWithSpinner(message string, fn func() error) error {
	spinner := NewSpinner(message)
	spinner.Start()
	err := fn()
	spinner.Stop()
	return err
}

// RunWithSpinnerResult executes a function while displaying a spinner.
// Returns the result and any error from the function.
func RunWithSpinnerResult[T any](message string, fn func() (T, error)) (T, error) {
	spinner := NewSpinner(message)
	spinner.Start()
	result, err := fn()
	spinner.Stop()
	return result, err
}
