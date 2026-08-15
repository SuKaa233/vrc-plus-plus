//go:build !windows

package singleinstance

type Guard struct {
	activations chan struct{}
}

func Acquire(string) (*Guard, bool, error) {
	return &Guard{activations: make(chan struct{})}, true, nil
}

func (g *Guard) Activations() <-chan struct{} { return g.activations }
func (g *Guard) Close()                       { close(g.activations) }
