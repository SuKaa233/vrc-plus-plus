//go:build windows

package updater

import (
	"syscall"

	"golang.org/x/sys/windows"
)

func hiddenProcessAttributes() *syscall.SysProcAttr { return &syscall.SysProcAttr{HideWindow: true} }
func processExists(pid int) bool {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(handle)
	var code uint32
	if windows.GetExitCodeProcess(handle, &code) != nil {
		return false
	}
	return code == 259
}
