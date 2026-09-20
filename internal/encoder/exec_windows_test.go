package encoder

import (
	"os/exec"
	"testing"
)

func TestHideWindowSetsSysProcAttr(t *testing.T) {
	cmd := exec.Command("cmd.exe", "/c", "echo")
	hideWindow(cmd)

	attr := cmd.SysProcAttr
	if attr == nil {
		t.Fatal("SysProcAttr = nil, want a *syscall.SysProcAttr")
	}
	if !attr.HideWindow {
		t.Error("HideWindow = false, want true")
	}
	if attr.CreationFlags&createNoWindow == 0 {
		t.Errorf("CreationFlags = %#x, want CREATE_NO_WINDOW set", attr.CreationFlags)
	}
}

func TestCreateNoWindowConstant(t *testing.T) {
	if createNoWindow != 0x08000000 {
		t.Errorf("createNoWindow = %#x, want 0x08000000", createNoWindow)
	}
}

func TestHideWindowIsIdempotent(t *testing.T) {
	cmd := exec.Command("cmd.exe", "/c", "echo")
	hideWindow(cmd)
	hideWindow(cmd)

	if cmd.SysProcAttr == nil {
		t.Fatal("SysProcAttr was cleared by a second call")
	}
	if !cmd.SysProcAttr.HideWindow {
		t.Error("HideWindow = false after a second call, want true")
	}
}
