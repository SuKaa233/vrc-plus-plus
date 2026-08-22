//go:build windows

package hardware

import (
	"context"
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

func hideCommandWindow(command *exec.Cmd) *exec.Cmd {
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	return command
}

func hiddenCommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return hideCommandWindow(exec.CommandContext(ctx, name, args...))
}

func hiddenCommand(name string, args ...string) *exec.Cmd {
	return hideCommandWindow(exec.Command(name, args...))
}
