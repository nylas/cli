package testutil

import (
	"bytes"
	"os"

	"github.com/spf13/cobra"
)

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
