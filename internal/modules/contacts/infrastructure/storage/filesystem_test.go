package storage

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestFileSystemRejectsTraversalAndLimitsWrites(t *testing.T) {
	store, err := NewFileSystem(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	key, size, err := store.Save(context.Background(), "workspace/contact", strings.NewReader("hello"), 10)
	if err != nil || size != 5 {
		t.Fatalf("save failed: key=%q size=%d err=%v", key, size, err)
	}
	reader, err := store.Open(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	contents, _ := io.ReadAll(reader)
	_ = reader.Close()
	if string(contents) != "hello" {
		t.Fatalf("unexpected contents %q", contents)
	}
	if _, _, err := store.Save(context.Background(), "workspace/contact", strings.NewReader("too large"), 3); err == nil {
		t.Fatal("oversized upload accepted")
	}
	if _, err := store.Open(context.Background(), "../../etc/passwd"); err == nil {
		t.Fatal("path traversal accepted")
	}
}
