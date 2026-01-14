package calendar

import (
	"bytes"

	"github.com/spf13/cobra"
)

// executeCommand is a test helper function to execute cobra commands and capture output
func executeCommand(root *cobra.Command, args ...string) (string, string, error) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(errBuf)
	root.SetArgs(args)

	err := root.Execute()

	return buf.String(), errBuf.String(), err
}
