package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInspectAndCleanAll(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GRADLE_USER_HOME", home)

	if err := os.MkdirAll(filepath.Join(home, "caches", "modules-2"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(home, "wrapper", "dists"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, "caches", "modules-2", "sample.bin"), []byte("data"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	info, err := Inspect()
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if info.Root != home {
		t.Fatalf("Inspect().Root = %q, want %q", info.Root, home)
	}
	if info.TotalSize == 0 {
		t.Fatalf("Inspect().TotalSize = 0, want > 0")
	}

	removed, err := CleanAll()
	if err != nil {
		t.Fatalf("CleanAll() error = %v", err)
	}
	if len(removed) != 2 {
		t.Fatalf("CleanAll() removed = %v, want 2 targets", removed)
	}
	if _, err := os.Stat(filepath.Join(home, "caches")); !os.IsNotExist(err) {
		t.Fatalf("caches dir still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, "wrapper", "dists")); !os.IsNotExist(err) {
		t.Fatalf("wrapper dists dir still exists: %v", err)
	}
}
