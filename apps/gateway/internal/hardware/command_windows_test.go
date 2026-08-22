//go:build windows

package hardware

import "testing"

func TestHardwareCommandsAreCreatedWithoutAConsoleWindow(t *testing.T) {
	command := hiddenCommand("tasklist.exe", "/?")
	if command.SysProcAttr == nil || !command.SysProcAttr.HideWindow {
		t.Fatal("hardware command must hide its window")
	}
	if command.SysProcAttr.CreationFlags&createNoWindow == 0 {
		t.Fatal("hardware command must use CREATE_NO_WINDOW")
	}
}
