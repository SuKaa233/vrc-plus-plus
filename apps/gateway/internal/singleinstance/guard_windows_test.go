//go:build windows

package singleinstance

import (
	"fmt"
	"testing"
	"time"
)

func TestSecondAcquireSignalsPrimaryInstance(t *testing.T) {
	name := fmt.Sprintf("VRCPlusPlusTest.%d", time.Now().UnixNano())
	guard, primary, err := Acquire(name)
	if err != nil {
		t.Fatal(err)
	}
	if !primary {
		t.Fatal("first acquire should own the instance")
	}
	defer guard.Close()

	second, secondPrimary, err := Acquire(name)
	if err != nil {
		t.Fatal(err)
	}
	if second != nil || secondPrimary {
		t.Fatal("second acquire should only signal the primary instance")
	}

	select {
	case <-guard.Activations():
	case <-time.After(2 * time.Second):
		t.Fatal("primary instance did not receive an activation signal")
	}
}
