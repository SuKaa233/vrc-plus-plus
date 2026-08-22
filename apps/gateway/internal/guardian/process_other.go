//go:build !windows

package guardian

func vrchatProcessRunning() bool { return false }
