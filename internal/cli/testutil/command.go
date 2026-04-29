package testutil

import (
	"bytes"
	"os"

	"github.com/spf13/cobra"
)

// NewTestRoot returns a minimal root command that carries the global persistent
// flags (--json, --format, --quiet) so that subcommands added to it can inherit
// them, matching the real nylas root. This is needed in unit tests where commands
// are constructed in isolation without the real root.
func NewTestRoot(sub *cobra.Command) *cobra.Command {
	root := &cobra.Command{Use: "nylas", Short: "test root"}
	root.PersistentFlags().Bool("json", false, "Output as JSON")
	root.PersistentFlags().String("format", "", "Output format")
	root.PersistentFlags().BoolP("quiet", "q", false, "Quiet output")
	root.AddCommand(sub)
	return root
}

// ExecuteSubCommand wraps sub under a test root with global persistent flags and
// runs it with the given args. This is the preferred helper for tests that need
// to pass --json or other global flags to an isolated subcommand.
func ExecuteSubCommand(sub *cobra.Command, args ...string) (stdout string, stderr string, err error) {
	root := NewTestRoot(sub)
	allArgs := append([]string{sub.Name()}, args...)
	return ExecuteCommand(root, allArgs...)
}

// ExecuteCommand runs a Cobra command while capturing both command output and direct stdout/stderr writes.
func ExecuteCommand(cmd *cobra.Command, args ...string) (stdout string, stderr string, err error) {
	stdoutBuf := new(bytes.Buffer)
	stderrBuf := new(bytes.Buffer)

	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rOut, wOut, pipeErr := os.Pipe()
	if pipeErr != nil {
		cmd.SetOut(stdoutBuf)
		cmd.SetErr(stderrBuf)
		cmd.SetArgs(args)
		err = cmd.Execute()
		return stdoutBuf.String(), stderrBuf.String(), err
	}

	rErr, wErr, pipeErr := os.Pipe()
	if pipeErr != nil {
		_ = rOut.Close()
		_ = wOut.Close()
		cmd.SetOut(stdoutBuf)
		cmd.SetErr(stderrBuf)
		cmd.SetArgs(args)
		err = cmd.Execute()
		return stdoutBuf.String(), stderrBuf.String(), err
	}

	os.Stdout = wOut
	os.Stderr = wErr
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	cmd.SetOut(stdoutBuf)
	cmd.SetErr(stderrBuf)
	cmd.SetArgs(args)
	err = cmd.Execute()

	_ = wOut.Close()
	_ = wErr.Close()

	var pipeStdout bytes.Buffer
	var pipeStderr bytes.Buffer
	_, _ = pipeStdout.ReadFrom(rOut)
	_, _ = pipeStderr.ReadFrom(rErr)
	_ = rOut.Close()
	_ = rErr.Close()

	return stdoutBuf.String() + pipeStdout.String(), stderrBuf.String() + pipeStderr.String(), err
}
