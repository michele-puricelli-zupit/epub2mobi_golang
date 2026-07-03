package calibre

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"epub2mobi/internal/runner"
)

func TestDryRunIncludesMOBIOptions(t *testing.T) {
	var out bytes.Buffer
	converter := NewConverter(Config{
		Command:      "ebook-convert",
		DryRun:       true,
		MOBIFileType: "both",
		ExtraArgs:    []string{"--disable-font-rescaling"},
		Output:       &out,
	})

	err := converter.Convert(context.Background(), runner.Job{
		InputPath:  "book.epub",
		OutputPath: filepath.Join(t.TempDir(), "book.mobi"),
		Overwrite:  true,
	})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"--mobi-file-type",
		"both",
		"--disable-font-rescaling",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("dry-run output %q does not contain %q", got, want)
		}
	}

	if strings.Contains(got, "--jpeg-quality") {
		t.Fatalf("dry-run output contains unsupported option: %q", got)
	}
}

func TestDryRunDoesNotPassMOBIOptionsForAZW3(t *testing.T) {
	var out bytes.Buffer
	converter := NewConverter(Config{
		Command:      "ebook-convert",
		DryRun:       true,
		MOBIFileType: "both",
		Output:       &out,
	})

	err := converter.Convert(context.Background(), runner.Job{
		InputPath:  "book.epub",
		OutputPath: filepath.Join(t.TempDir(), "book.azw3"),
		Overwrite:  true,
	})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, ".azw3") {
		t.Fatalf("dry-run output %q does not contain azw3 output", got)
	}
	if strings.Contains(got, "--mobi-file-type") {
		t.Fatalf("dry-run output contains MOBI-only option: %q", got)
	}
}
