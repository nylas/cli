package webhook

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// preflightPrompter abstracts the user-facing prompts used by
// preflightTunnelChoice so prompt errors (EOF, closed stdin, signal) are
// propagated to the caller rather than silently treated as the default
// answer, and so the flow can be unit-tested without a TTY.
//
// All implementations MUST surface io.EOF unchanged when the user closes
// stdin (Ctrl-D) so security-sensitive callers can distinguish "user
// cancelled" from "user accepted the default".
type preflightPrompter interface {
	Confirm(message string, defaultYes bool) (bool, error)
	Password(message string) (string, error)
}

// stdinPrompter is the production preflightPrompter. It reads from
// os.Stdin and writes to os.Stdout, using term.ReadPassword so secrets
// are never echoed to the terminal or shell history.
type stdinPrompter struct {
	in  *bufio.Reader
	out io.Writer
}

func newStdinPrompter() *stdinPrompter {
	return &stdinPrompter{in: bufio.NewReader(os.Stdin), out: os.Stdout}
}

// Confirm reads a y/n response, distinguishing EOF (returned as io.EOF)
// from an empty line (interpreted as defaultYes). Any other read error
// is propagated unchanged.
func (p *stdinPrompter) Confirm(message string, defaultYes bool) (bool, error) {
	suffix := " [y/N]: "
	if defaultYes {
		suffix = " [Y/n]: "
	}
	if _, err := fmt.Fprint(p.out, message+suffix); err != nil {
		return false, err
	}
	line, err := p.in.ReadString('\n')
	if errors.Is(err, io.EOF) && strings.TrimSpace(line) == "" {
		// User pressed Ctrl-D before typing anything — propagate EOF so
		// the caller can decide whether to fall back to a safe default
		// (e.g., NOT auto-accepting an `--allow-unsigned` posture).
		return false, io.EOF
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	response := strings.ToLower(strings.TrimSpace(line))
	if response == "" {
		return defaultYes, nil
	}
	return response == "y" || response == "yes", nil
}

// Password prompts for a secret with terminal echo disabled when stdin
// is a TTY. When stdin is not a TTY (tests, pipes), it reads a line in
// the clear — echo doesn't matter in those contexts.
func (p *stdinPrompter) Password(message string) (string, error) {
	if _, err := fmt.Fprint(p.out, message); err != nil {
		return "", err
	}
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		line, err := p.in.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}
		return strings.TrimSpace(line), nil
	}
	raw, err := term.ReadPassword(fd)
	// term.ReadPassword swallows the trailing newline — emit one so the
	// next line of output isn't glued to the prompt.
	_, _ = fmt.Fprintln(p.out)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(raw)), nil
}
