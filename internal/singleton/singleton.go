package singleton

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
)

func Acquire(name string) (bool, error) {
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	createMutex := kernel32.NewProc("CreateMutexW")

	namePtr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return false, err
	}

	ret, _, callErr := createMutex.Call(0, 0, uintptr(unsafe.Pointer(namePtr)))
	if ret == 0 {
		return false, callErr
	}
	if errors.Is(callErr, windows.ERROR_ALREADY_EXISTS) {
		return false, nil
	}
	return true, nil
}
