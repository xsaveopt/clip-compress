package singleton

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func uniqueName(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf(`Local\clip-compress-test-%d-%s`, os.Getpid(), strings.ReplaceAll(t.Name(), "/", "-"))
}

func TestAcquireSucceedsForAFreeName(t *testing.T) {
	ok, err := Acquire(uniqueName(t))
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if !ok {
		t.Error("Acquire = false, want true for a name nothing holds")
	}
}

func TestAcquireReportsAnExistingHolder(t *testing.T) {
	name := uniqueName(t)

	ok, err := Acquire(name)
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	if !ok {
		t.Fatal("first Acquire = false, want true")
	}

	ok, err = Acquire(name)
	if err != nil {
		t.Fatalf("second Acquire: %v", err)
	}
	if ok {
		t.Error("second Acquire = true, want false while the name is held")
	}
}

func TestAcquireIndependentNamesDoNotCollide(t *testing.T) {
	for i := range 3 {
		ok, err := Acquire(fmt.Sprintf("%s-%d", uniqueName(t), i))
		if err != nil {
			t.Fatalf("Acquire %d: %v", i, err)
		}
		if !ok {
			t.Errorf("Acquire %d = false, want true", i)
		}
	}
}

func TestAcquireRejectsAnInvalidName(t *testing.T) {
	ok, err := Acquire("clip-compress\x00test")
	if err == nil {
		t.Fatal("Acquire should reject a name holding a NUL")
	}
	if ok {
		t.Error("Acquire = true on an invalid name, want false")
	}
}
