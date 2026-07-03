package ebook

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindEPUBsNonRecursive(t *testing.T) {
	root := t.TempDir()
	touch(t, filepath.Join(root, "a.epub"))
	touch(t, filepath.Join(root, "b.EPUB"))
	touch(t, filepath.Join(root, "notes.txt"))
	mkdir(t, filepath.Join(root, "nested"))
	touch(t, filepath.Join(root, "nested", "c.epub"))

	books, err := FindEPUBs(root, false)
	if err != nil {
		t.Fatalf("FindEPUBs returned error: %v", err)
	}

	if got, want := len(books), 2; got != want {
		t.Fatalf("len(books) = %d, want %d", got, want)
	}
	if books[0].Relative != "a.epub" || books[1].Relative != "b.EPUB" {
		t.Fatalf("unexpected books: %#v", books)
	}
}

func TestFindEPUBsRecursive(t *testing.T) {
	root := t.TempDir()
	touch(t, filepath.Join(root, "a.epub"))
	mkdir(t, filepath.Join(root, "nested"))
	touch(t, filepath.Join(root, "nested", "c.epub"))

	books, err := FindEPUBs(root, true)
	if err != nil {
		t.Fatalf("FindEPUBs returned error: %v", err)
	}

	want := []string{"a.epub", filepath.Join("nested", "c.epub")}
	if len(books) != len(want) {
		t.Fatalf("len(books) = %d, want %d", len(books), len(want))
	}
	for i := range want {
		if books[i].Relative != want[i] {
			t.Fatalf("books[%d].Relative = %q, want %q", i, books[i].Relative, want[i])
		}
	}
}

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
		t.Fatalf("touch %s: %v", path, err)
	}
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}
