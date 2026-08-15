//go:build !windows

package security

import "fmt"

type unsupportedProtector struct{}

func NewProtector() Protector             { return unsupportedProtector{} }
func (unsupportedProtector) Name() string { return "unsupported" }
func (unsupportedProtector) Protect([]byte) ([]byte, error) {
	return nil, fmt.Errorf("secure session storage is currently implemented for Windows only")
}
func (unsupportedProtector) Unprotect([]byte) ([]byte, error) {
	return nil, fmt.Errorf("secure session storage is currently implemented for Windows only")
}
