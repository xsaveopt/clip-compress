//go:build !windows

package startup

func Enabled(string) bool     { return false }
func Enable(string) error     { return nil }
func Disable(string) error    { return nil }
func Sync(string, bool) error { return nil }
