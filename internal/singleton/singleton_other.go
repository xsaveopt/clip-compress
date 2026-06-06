//go:build !windows

package singleton

func Acquire(string) (bool, error) { return true, nil }
