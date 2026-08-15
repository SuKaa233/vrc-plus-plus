//go:build !windows

package tray

func Start(_ func(), _ func(), _ chan<- struct{}) {}
func Stop()                                       {}
