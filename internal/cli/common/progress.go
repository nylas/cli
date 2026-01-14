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

// SetFrames sets the spinner animation frames.
func (s *Spinner) SetFrames(frames []string) *Spinner {
	s.frames = frames
	return s
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

// StopWithMessage stops the spinner and prints a final message.
func (s *Spinner) StopWithMessage(message string) {
	s.Stop()
	if !IsQuiet() {
		_, _ = fmt.Fprintln(s.writer, message)
	}
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

// ProgressBar provides a progress bar for determinate operations.
type ProgressBar struct {
	total     int
	current   int
	width     int
	message   string
	writer    io.Writer
	startTime time.Time
	mu        sync.Mutex
}

// NewProgressBar creates a new progress bar.
func NewProgressBar(total int, message string) *ProgressBar {
	return &ProgressBar{
		total:     total,
		current:   0,
		width:     40,
		message:   message,
		writer:    os.Stderr,
		startTime: time.Now(),
	}
}

// SetWidth sets the progress bar width.
func (p *ProgressBar) SetWidth(width int) *ProgressBar {
	p.width = width
	return p
}

// SetWriter sets the output writer.
func (p *ProgressBar) SetWriter(w io.Writer) *ProgressBar {
	p.writer = w
	return p
}

// Increment increments the progress by 1.
func (p *ProgressBar) Increment() {
	p.Add(1)
}

// Add adds n to the current progress.
func (p *ProgressBar) Add(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current += n
	if p.current > p.total {
		p.current = p.total
	}

	p.render()
}

// Set sets the current progress.
func (p *ProgressBar) Set(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = n
	if p.current > p.total {
		p.current = p.total
	}

	p.render()
}

// render draws the progress bar.
func (p *ProgressBar) render() {
	if IsQuiet() {
		return
	}

	percent := float64(p.current) / float64(p.total)
	filled := int(percent * float64(p.width))
	empty := p.width - filled

	// Calculate ETA
	elapsed := time.Since(p.startTime)
	var eta string
	if p.current > 0 {
		remaining := time.Duration(float64(elapsed) / percent * (1 - percent))
		if remaining > time.Second {
			eta = fmt.Sprintf(" ETA: %s", formatDuration(remaining))
		}
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	_, _ = fmt.Fprintf(p.writer, "\r%s %s %s %d/%d (%.0f%%)%s",
		p.message,
		Cyan.Sprint("["),
		bar,
		p.current,
		p.total,
		percent*100,
		eta,
	)

	if p.current >= p.total {
		_, _ = fmt.Fprintln(p.writer)
	}
}

// Finish completes the progress bar.
func (p *ProgressBar) Finish() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = p.total
	p.render()
}

// formatDuration formats a duration for display.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
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

// Count returns the current count.
func (c *Counter) Count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.count
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
