//go:build !windows

package encoder

import "os/exec"

func hideWindow(*exec.Cmd) {}
