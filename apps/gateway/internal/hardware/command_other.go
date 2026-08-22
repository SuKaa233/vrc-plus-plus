//go:build !windows

package hardware

import (
	"context"
	"os/exec"
)

func hiddenCommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

func hiddenCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
