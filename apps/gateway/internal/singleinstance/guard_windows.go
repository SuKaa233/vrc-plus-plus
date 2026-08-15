//go:build windows

package singleinstance

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/sys/windows"
)

const activationRetryCount = 10

type Guard struct {
	mutex       windows.Handle
	activation  windows.Handle
	activations chan struct{}
	stop        chan struct{}
}

func Acquire(name string) (*Guard, bool, error) {
	mutexName, err := windows.UTF16PtrFromString(`Local\` + name + `.mutex`)
	if err != nil {
		return nil, false, err
	}
	mutex, mutexErr := windows.CreateMutex(nil, false, mutexName)
	if mutex == 0 {
		return nil, false, fmt.Errorf("create single-instance mutex: %w", mutexErr)
	}
	if errors.Is(mutexErr, windows.ERROR_ALREADY_EXISTS) {
		_ = windows.CloseHandle(mutex)
		if err := signalExisting(name); err != nil {
			return nil, false, err
		}
		return nil, false, nil
	}
	if mutexErr != nil {
		_ = windows.CloseHandle(mutex)
		return nil, false, fmt.Errorf("create single-instance mutex: %w", mutexErr)
	}

	eventName, err := windows.UTF16PtrFromString(`Local\` + name + `.activate`)
	if err != nil {
		_ = windows.CloseHandle(mutex)
		return nil, false, err
	}
	activation, eventErr := windows.CreateEvent(nil, 0, 0, eventName)
	if activation == 0 || (eventErr != nil && !errors.Is(eventErr, windows.ERROR_ALREADY_EXISTS)) {
		_ = windows.CloseHandle(mutex)
		return nil, false, fmt.Errorf("create activation event: %w", eventErr)
	}

	guard := &Guard{
		mutex:       mutex,
		activation:  activation,
		activations: make(chan struct{}, 1),
		stop:        make(chan struct{}),
	}
	go guard.watch()
	return guard, true, nil
}

func signalExisting(name string) error {
	eventName, err := windows.UTF16PtrFromString(`Local\` + name + `.activate`)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < activationRetryCount; attempt++ {
		event, openErr := windows.OpenEvent(windows.EVENT_MODIFY_STATE, false, eventName)
		if openErr == nil {
			defer windows.CloseHandle(event)
			return windows.SetEvent(event)
		}
		time.Sleep(50 * time.Millisecond)
	}
	return errors.New("VRC++ is already running, but its window could not be activated")
}

func (g *Guard) Activations() <-chan struct{} { return g.activations }

func (g *Guard) watch() {
	defer close(g.activations)
	for {
		select {
		case <-g.stop:
			return
		default:
		}
		result, err := windows.WaitForSingleObject(g.activation, 250)
		if err != nil {
			return
		}
		if result == windows.WAIT_OBJECT_0 {
			select {
			case g.activations <- struct{}{}:
			default:
			}
		}
	}
}

func (g *Guard) Close() {
	select {
	case <-g.stop:
		return
	default:
		close(g.stop)
	}
	_ = windows.SetEvent(g.activation)
	_ = windows.CloseHandle(g.activation)
	_ = windows.CloseHandle(g.mutex)
}
