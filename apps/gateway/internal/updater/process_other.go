//go:build !windows

package updater

import (
	"os"
	"syscall"
)

func hiddenProcessAttributes() *syscall.SysProcAttr { return nil }
func processExists(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}
