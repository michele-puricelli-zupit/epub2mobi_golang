package ebook

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Book struct {
	InputPath string
	Relative  string
}

func FindEPUBs(root string, recursive bool) ([]Book, error) {
	var books []Book

	if recursive {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			if !isEPUB(path) {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return fmt.Errorf("percorso relativo per %q: %w", path, err)
			}
			books = append(books, Book{InputPath: path, Relative: rel})
			return nil
		})
		if err != nil {
			return nil, err
		}
		sortBooks(books)
		return books, nil
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !isEPUB(name) {
			continue
		}
		books = append(books, Book{
			InputPath: filepath.Join(root, name),
			Relative:  name,
		})
	}
	sortBooks(books)
	return books, nil
}

func isEPUB(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".epub")
}

func sortBooks(books []Book) {
	sort.Slice(books, func(i, j int) bool {
		return books[i].Relative < books[j].Relative
	})
}
