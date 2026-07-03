package runner

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"epub2mobi/internal/ebook"
)

var (
	ErrAlreadyExists        = errors.New("output gia' presente")
	ErrConverterUnavailable = errors.New("converter non disponibile")
)

type Config struct {
	InputDir      string
	OutputDir     string
	OutputFormat  string
	Recursive     bool
	Overwrite     bool
	DryRun        bool
	Jobs          int
	ConverterPath string
	Verbose       bool
}

type Job struct {
	InputPath  string
	OutputPath string
	Overwrite  bool
}

type Converter interface {
	Available() error
	Convert(context.Context, Job) error
}

type Summary struct {
	Found     int
	Converted int
	Skipped   int
	Failed    int
	Errors    []error
}

func Run(ctx context.Context, cfg Config, converter Converter) (Summary, error) {
	var summary Summary

	if err := converter.Available(); err != nil {
		return summary, err
	}

	books, err := ebook.FindEPUBs(cfg.InputDir, cfg.Recursive)
	if err != nil {
		return summary, fmt.Errorf("ricerca EPUB: %w", err)
	}
	summary.Found = len(books)
	if len(books) == 0 {
		return summary, nil
	}

	jobs := make([]Job, 0, len(books))
	for _, book := range books {
		jobs = append(jobs, Job{
			InputPath:  book.InputPath,
			OutputPath: outputPath(cfg.OutputDir, book.Relative, cfg.OutputFormat),
			Overwrite:  cfg.Overwrite,
		})
	}

	return runJobs(ctx, cfg.Jobs, jobs, converter, summary)
}

func runJobs(ctx context.Context, workers int, jobs []Job, converter Converter, summary Summary) (Summary, error) {
	if workers < 1 {
		workers = 1
	}
	if workers > len(jobs) {
		workers = len(jobs)
	}

	jobCh := make(chan Job)
	resultCh := make(chan error)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobCh {
				resultCh <- converter.Convert(ctx, job)
			}
		}()
	}

	go func() {
		defer close(jobCh)
		for _, job := range jobs {
			select {
			case <-ctx.Done():
				return
			case jobCh <- job:
			}
		}
	}()

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	for err := range resultCh {
		switch {
		case err == nil:
			summary.Converted++
		case errors.Is(err, ErrAlreadyExists):
			summary.Skipped++
		default:
			summary.Failed++
			summary.Errors = append(summary.Errors, err)
		}
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		return summary, ctxErr
	}
	if len(summary.Errors) > 0 {
		return summary, fmt.Errorf("%d conversioni fallite: %w", summary.Failed, errors.Join(summary.Errors...))
	}

	return summary, nil
}

func outputPath(outputDir, relative, format string) string {
	if format == "" {
		format = "mobi"
	}
	ext := filepath.Ext(relative)
	base := strings.TrimSuffix(relative, ext) + "." + strings.ToLower(format)
	return filepath.Join(outputDir, base)
}
