//go:build windows

package air

import (
	"os/exec"
	"strconv"
)

func configureChildProcessGroup(cmd *exec.Cmd) {}

func killCommandTree(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	taskkill := exec.Command("taskkill", "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F")
	if err := taskkill.Run(); err == nil {
		return nil
	}

	return cmd.Process.Kill()
}
