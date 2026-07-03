package calibre

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"epub2mobi/internal/runner"
)

type Config struct {
	Command      string
	Timeout      time.Duration
	DryRun       bool
	Verbose      bool
	MOBIFileType string
	ExtraArgs    []string
	Output       io.Writer
	ErrorSink    io.Writer
}

type Converter struct {
	command      string
	timeout      time.Duration
	dryRun       bool
	verbose      bool
	mobiFileType string
	extraArgs    []string
	output       io.Writer
	errorSink    io.Writer
}

func NewConverter(cfg Config) *Converter {
	if cfg.Command == "" {
		cfg.Command = "ebook-convert"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Minute
	}
	if cfg.MOBIFileType == "" {
		cfg.MOBIFileType = "both"
	}
	if cfg.Output == nil {
		cfg.Output = io.Discard
	}
	if cfg.ErrorSink == nil {
		cfg.ErrorSink = io.Discard
	}

	return &Converter{
		command:      cfg.Command,
		timeout:      cfg.Timeout,
		dryRun:       cfg.DryRun,
		verbose:      cfg.Verbose,
		mobiFileType: cfg.MOBIFileType,
		extraArgs:    append([]string(nil), cfg.ExtraArgs...),
		output:       cfg.Output,
		errorSink:    cfg.ErrorSink,
	}
}

func (c *Converter) Available() error {
	if c.dryRun {
		return nil
	}
	if _, err := exec.LookPath(c.command); err != nil {
		return fmt.Errorf("%w: comando %q non trovato; installa Calibre o usa -converter", runner.ErrConverterUnavailable, c.command)
	}
	return nil
}

func (c *Converter) Convert(ctx context.Context, job runner.Job) error {
	if !job.Overwrite {
		if _, err := os.Stat(job.OutputPath); err == nil {
			return runner.ErrAlreadyExists
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("controllo output %q: %w", job.OutputPath, err)
		}
	}

	if err := os.MkdirAll(filepath.Dir(job.OutputPath), 0o755); err != nil {
		return fmt.Errorf("creazione cartella output: %w", err)
	}

	if c.dryRun {
		fmt.Fprintf(c.output, "DRY-RUN %s\n", formatCommand(c.command, c.argsFor(job)))
		return nil
	}

	commandCtx := ctx
	var cancel context.CancelFunc
	if c.timeout > 0 {
		commandCtx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(commandCtx, c.command, c.argsFor(job)...)
	combined, err := cmd.CombinedOutput()
	if len(combined) > 0 && c.verbose {
		fmt.Fprintln(c.output, strings.TrimSpace(string(combined)))
	}
	if errors.Is(commandCtx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("timeout dopo %s: %s", c.timeout, job.InputPath)
	}
	if err != nil {
		message := strings.TrimSpace(string(combined))
		if message == "" {
			message = err.Error()
		}
		fmt.Fprintln(c.errorSink, message)
		return fmt.Errorf("conversione fallita per %q: %w", job.InputPath, err)
	}

	return nil
}

func (c *Converter) argsFor(job runner.Job) []string {
	args := []string{job.InputPath, job.OutputPath}
	if c.mobiFileType != "" && strings.EqualFold(filepath.Ext(job.OutputPath), ".mobi") {
		args = append(args, "--mobi-file-type", c.mobiFileType)
	}
	args = append(args, c.extraArgs...)
	return args
}

func formatCommand(command string, args []string) string {
	all := append([]string{command}, args...)
	quoted := make([]string, len(all))
	for i, arg := range all {
		quoted[i] = strconv.Quote(arg)
	}
	return strings.Join(quoted, " ")
}
