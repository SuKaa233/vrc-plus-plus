//go:build !windows

package desktop

import (
	"context"
	"errors"
)

var ErrUnavailable = errors.New("embedded desktop window is only available on Windows")

type Manager struct{}

func New() *Manager                                        { return &Manager{} }
func (*Manager) Show()                                     {}
func (*Manager) Run(context.Context, string, string) error { return ErrUnavailable }
