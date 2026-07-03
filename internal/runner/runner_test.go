package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestRunBuildsJobsAndCountsResults(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(root, "out")
	touch(t, filepath.Join(root, "one.epub"))
	touch(t, filepath.Join(root, "two.epub"))
	touch(t, filepath.Join(root, "skip.epub"))

	converter := &fakeConverter{
		errs: map[string]error{
			filepath.Join(root, "skip.epub"): ErrAlreadyExists,
		},
	}

	summary, err := Run(context.Background(), Config{
		InputDir:  root,
		OutputDir: out,
		Jobs:      2,
	}, converter)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if summary.Found != 3 || summary.Converted != 2 || summary.Skipped != 1 || summary.Failed != 0 {
		t.Fatalf("unexpected summary: %#v", summary)
	}

	expected := map[string]bool{
		filepath.Join(out, "one.mobi"):  true,
		filepath.Join(out, "two.mobi"):  true,
		filepath.Join(out, "skip.mobi"): true,
	}
	for _, job := range converter.jobs {
		if !expected[job.OutputPath] {
			t.Fatalf("unexpected output path: %s", job.OutputPath)
		}
		delete(expected, job.OutputPath)
	}
	if len(expected) > 0 {
		t.Fatalf("missing expected jobs: %#v", expected)
	}
}

func TestRunBuildsAZW3Jobs(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(root, "out")
	touch(t, filepath.Join(root, "one.epub"))

	converter := &fakeConverter{}
	summary, err := Run(context.Background(), Config{
		InputDir:      root,
		OutputDir:     out,
		OutputFormat:  "azw3",
		Jobs:          1,
		ConverterPath: "ebook-convert",
	}, converter)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if summary.Converted != 1 {
		t.Fatalf("summary.Converted = %d, want 1", summary.Converted)
	}
	if got, want := converter.jobs[0].OutputPath, filepath.Join(out, "one.azw3"); got != want {
		t.Fatalf("OutputPath = %q, want %q", got, want)
	}
}

func TestRunReturnsUnavailableConverter(t *testing.T) {
	want := ErrConverterUnavailable
	converter := &fakeConverter{availableErr: want}

	summary, err := Run(context.Background(), Config{InputDir: t.TempDir(), OutputDir: t.TempDir(), Jobs: 1}, converter)
	if !errors.Is(err, want) {
		t.Fatalf("Run error = %v, want %v", err, want)
	}
	if summary.Found != 0 {
		t.Fatalf("summary.Found = %d, want 0", summary.Found)
	}
}

type fakeConverter struct {
	availableErr error
	errs         map[string]error
	mu           sync.Mutex
	jobs         []Job
}

func (f *fakeConverter) Available() error {
	return f.availableErr
}

func (f *fakeConverter) Convert(_ context.Context, job Job) error {
	f.mu.Lock()
	f.jobs = append(f.jobs, job)
	f.mu.Unlock()
	if f.errs != nil {
		return f.errs[job.InputPath]
	}
	return nil
}

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
		t.Fatalf("touch %s: %v", path, err)
	}
}
