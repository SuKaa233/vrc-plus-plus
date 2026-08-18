//go:build !windows

package tray

import "errors"

func Start(_ func(), _ func(), _ chan<- struct{}) {}
func Stop()                                       {}
func Notify(_, _ string) error                    { return errors.New("desktop notifications are unavailable") }
