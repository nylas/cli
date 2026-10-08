//go:build !unix && !windows

package filelock

import (
	"errors"
	"os"
)

// errUnsupported fails closed: without a cross-process lock two refreshers
// could replay a rotated token and sign the user out.
var errUnsupported = errors.New("cross-process file locking is not supported on this platform")

func tryLock(*os.File) (bool, error) { return false, errUnsupported }

func unlockFile(*os.File) error { return errUnsupported }
