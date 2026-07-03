package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"epub2mobi/internal/calibre"
	"epub2mobi/internal/runner"
)

func main() {
	var cfg runner.Config
	var timeout time.Duration
	var showVersion bool
	var mobiFileType string
	var extraCalibreArgs repeatedStringFlag

	flag.StringVar(&cfg.InputDir, "in", "", "cartella sorgente con file .epub")
	flag.StringVar(&cfg.OutputDir, "out", "", "cartella di output")
	flag.StringVar(&cfg.OutputFormat, "format", "mobi", "formato output: mobi o azw3")
	flag.BoolVar(&cfg.Recursive, "recursive", false, "cerca file .epub anche nelle sottocartelle")
	flag.BoolVar(&cfg.Overwrite, "overwrite", false, "sovrascrive i file output gia' presenti")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "mostra le conversioni senza eseguirle")
	flag.IntVar(&cfg.Jobs, "jobs", 1, "numero di conversioni parallele")
	flag.StringVar(&cfg.ConverterPath, "converter", "ebook-convert", "comando o percorso di ebook-convert")
	flag.StringVar(&mobiFileType, "mobi-type", "both", "tipo MOBI generato da Calibre: old, new o both")
	flag.Var(&extraCalibreArgs, "calibre-arg", "argomento extra da passare a ebook-convert; ripetibile")
	flag.DurationVar(&timeout, "timeout", 30*time.Minute, "timeout massimo per ogni conversione")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "stampa output dettagliato di ebook-convert")
	flag.BoolVar(&showVersion, "version", false, "stampa la versione ed esce")
	flag.Parse()

	if showVersion {
		fmt.Println("epub2mobi 0.1.0")
		return
	}

	if err := normalizeConfig(&cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		flag.Usage()
		os.Exit(2)
	}
	if err := validateCalibreOptions(mobiFileType); err != nil {
		fmt.Fprintln(os.Stderr, err)
		flag.Usage()
		os.Exit(2)
	}

	converter := calibre.NewConverter(calibre.Config{
		Command:      cfg.ConverterPath,
		Timeout:      timeout,
		DryRun:       cfg.DryRun,
		Verbose:      cfg.Verbose,
		MOBIFileType: mobiFileType,
		ExtraArgs:    []string(extraCalibreArgs),
		Output:       os.Stdout,
		ErrorSink:    os.Stderr,
	})

	summary, err := runner.Run(context.Background(), cfg, converter)
	printSummary(summary)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCode(err))
	}
}

func validateCalibreOptions(mobiFileType string) error {
	switch strings.ToLower(mobiFileType) {
	case "", "old", "new", "both":
	default:
		return fmt.Errorf("mobi-type non valido: %q; usa old, new o both", mobiFileType)
	}

	return nil
}

func normalizeConfig(cfg *runner.Config) error {
	if cfg.InputDir == "" {
		return errors.New("errore: -in e' obbligatorio")
	}
	cfg.OutputFormat = strings.ToLower(strings.TrimSpace(cfg.OutputFormat))
	switch cfg.OutputFormat {
	case "", "mobi":
		cfg.OutputFormat = "mobi"
	case "azw3":
	default:
		return fmt.Errorf("format non valido: %q; usa mobi o azw3", cfg.OutputFormat)
	}

	inputAbs, err := filepath.Abs(cfg.InputDir)
	if err != nil {
		return fmt.Errorf("input non valido: %w", err)
	}
	cfg.InputDir = inputAbs

	info, err := os.Stat(cfg.InputDir)
	if err != nil {
		return fmt.Errorf("input non leggibile: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("input non valido: %s non e' una cartella", cfg.InputDir)
	}

	if cfg.OutputDir == "" {
		cfg.OutputDir = filepath.Join(cfg.InputDir, cfg.OutputFormat)
	}
	outputAbs, err := filepath.Abs(cfg.OutputDir)
	if err != nil {
		return fmt.Errorf("output non valido: %w", err)
	}
	cfg.OutputDir = outputAbs

	if samePath(cfg.InputDir, cfg.OutputDir) {
		return errors.New("output non valido: -out deve essere diverso da -in")
	}

	if cfg.Jobs < 1 {
		return errors.New("jobs non valido: deve essere almeno 1")
	}
	maxJobs := runtime.NumCPU()
	if cfg.Jobs > maxJobs {
		cfg.Jobs = maxJobs
	}

	return nil
}

func samePath(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return equalFold(a, b)
	}
	return a == b
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		ca := a[i]
		cb := b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func printSummary(summary runner.Summary) {
	fmt.Printf(
		"\nTrovati: %d | Convertiti: %d | Saltati: %d | Falliti: %d\n",
		summary.Found,
		summary.Converted,
		summary.Skipped,
		summary.Failed,
	)
}

func exitCode(err error) int {
	if errors.Is(err, runner.ErrConverterUnavailable) {
		return 127
	}
	return 1
}

type repeatedStringFlag []string

func (f *repeatedStringFlag) String() string {
	return strings.Join(*f, " ")
}

func (f *repeatedStringFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}
