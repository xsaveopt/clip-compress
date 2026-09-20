//go:build !windows

package encoder

import (
	"os/exec"
	"testing"
)

func TestHideWindowIsANoOp(t *testing.T) {
	cmd := exec.Command("true")
	hideWindow(cmd)

	if cmd.SysProcAttr != nil {
		t.Errorf("SysProcAttr = %#v, want nil off Windows", cmd.SysProcAttr)
	}
}

func TestHideWindowLeavesTheCommandRunnable(t *testing.T) {
	cmd := exec.Command("true")
	hideWindow(cmd)

	if err := cmd.Run(); err != nil {
		t.Fatalf("Run after hideWindow: %v", err)
	}
}
