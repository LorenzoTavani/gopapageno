package generator

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
)

func TestEmitDirectoryCreatesMissingDirectories(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "a", "b")
	opts := &Options{OutputDirectory: dir, TypesOnly: true, Logger: log.New(io.Discard, "", 0)}

	name, err := emitDirectory(opts)
	if err != nil {
		t.Fatal(err)
	}
	if name != "b" {
		t.Errorf("package name = %q, want %q", name, "b")
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("the directory was not created: %v", err)
	}
}
