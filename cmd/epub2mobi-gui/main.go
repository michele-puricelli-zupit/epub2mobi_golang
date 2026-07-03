package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"epub2mobi/internal/calibre"
	"epub2mobi/internal/runner"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

const ebookConvert = "ebook-convert"

func main() {

	a := app.NewWithID("epub2mobi.gui")
	w := a.NewWindow("EPUB to Kindle Converter")
	w.Resize(fyne.NewSize(760, 560))

	inputEntry := widget.NewEntry()
	inputEntry.SetPlaceHolder("Cartella con file .epub")
	outputEntry := widget.NewEntry()
	outputEntry.SetPlaceHolder("Cartella di output; vuota = <input>/<format>")
	converterEntry := widget.NewEntry()
	converterEntry.SetText(ebookConvert)
	converterEntry.SetPlaceHolder(ebookConvert)

	formatSelect := widget.NewSelect([]string{"mobi", "azw3"}, nil)
	formatSelect.SetSelected("azw3")

	mobiTypeSelect := widget.NewSelect([]string{"old", "new", "both"}, nil)
	mobiTypeSelect.SetSelected("both")

	jobsEntry := widget.NewEntry()
	jobsEntry.SetText(strconv.Itoa(runtime.NumCPU()))
	jobsEntry.SetPlaceHolder("1")

	timeoutEntry := widget.NewEntry()
	timeoutEntry.SetText("30m")
	timeoutEntry.SetPlaceHolder("30m")

	recursiveCheck := widget.NewCheck("Recursive", nil)
	overwriteCheck := widget.NewCheck("Overwrite", nil)
	verboseCheck := widget.NewCheck("Verbose", nil)
	dryRunCheck := widget.NewCheck("Dry-run", nil)

	logEntry := widget.NewMultiLineEntry()
	logEntry.SetPlaceHolder("Log conversione")
	logEntry.Wrapping = fyne.TextWrapWord
	logEntry.SetMinRowsVisible(12)

	statusLabel := widget.NewLabel("Pronto")
	statusLabel.Truncation = fyne.TextTruncateEllipsis

	var cancel context.CancelFunc
	var runningMu sync.Mutex
	running := false

	pickFolder := func(target *widget.Entry) {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if uri == nil {
				return
			}
			target.SetText(uri.Path())
		}, w)
	}

	inputButton := widget.NewButton("Scegli...", func() { pickFolder(inputEntry) })
	outputButton := widget.NewButton("Scegli...", func() { pickFolder(outputEntry) })

	startButton := widget.NewButton("Converti", nil)
	cancelButton := widget.NewButton("Annulla", func() {
		runningMu.Lock()
		defer runningMu.Unlock()
		if cancel != nil {
			cancel()
			statusLabel.SetText("Annullamento in corso...")
		}
	})
	cancelButton.Disable()

	setRunning := func(value bool) {
		runningMu.Lock()
		running = value
		runningMu.Unlock()

		if value {
			startButton.Disable()
			cancelButton.Enable()
			return
		}
		startButton.Enable()
		cancelButton.Disable()
	}

	startButton.OnTapped = func() {
		runningMu.Lock()
		if running {
			runningMu.Unlock()
			return
		}
		runningMu.Unlock()

		cfg, converterCfg, err := buildConfigs(uiValues{
			inputDir:      inputEntry.Text,
			outputDir:     outputEntry.Text,
			outputFormat:  formatSelect.Selected,
			converterPath: converterEntry.Text,
			mobiFileType:  mobiTypeSelect.Selected,
			jobs:          jobsEntry.Text,
			timeout:       timeoutEntry.Text,
			recursive:     recursiveCheck.Checked,
			overwrite:     overwriteCheck.Checked,
			verbose:       verboseCheck.Checked,
			dryRun:        dryRunCheck.Checked,
		})
		if err != nil {
			dialog.ShowError(err, w)
			return
		}

		logEntry.SetText("")
		statusLabel.SetText("Conversione in corso...")
		setRunning(true)

		ctx, stop := context.WithCancel(context.Background())
		runningMu.Lock()
		cancel = stop
		runningMu.Unlock()

		go func() {
			defer func() {
				stop()
				runningMu.Lock()
				if cancel != nil {
					cancel = nil
				}
				runningMu.Unlock()
				fyne.Do(func() {
					setRunning(false)
				})
			}()

			logWriter := newUILogWriter(logEntry)
			converterCfg.Output = logWriter
			converterCfg.ErrorSink = logWriter
			converter := loggingConverter{
				inner: calibre.NewConverter(converterCfg),
				log:   logWriter,
			}

			summary, runErr := runner.Run(ctx, cfg, converter)
			logWriter.Printf(
				"\nTrovati: %d | Convertiti: %d | Saltati: %d | Falliti: %d\n",
				summary.Found,
				summary.Converted,
				summary.Skipped,
				summary.Failed,
			)

			fyne.Do(func() {
				if runErr != nil {
					statusLabel.SetText("Conversione terminata con errori")
					dialog.ShowError(runErr, w)
					return
				}
				statusLabel.SetText("Conversione completata")
			})
		}()
	}

	formatSelect.OnChanged = func(value string) {
		if value == "mobi" {
			mobiTypeSelect.Enable()
			return
		}
		mobiTypeSelect.Disable()
	}
	mobiTypeSelect.Disable()

	form := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Cartella EPUB", container.NewBorder(nil, nil, nil, inputButton, inputEntry)),
			widget.NewFormItem("Cartella output", container.NewBorder(nil, nil, nil, outputButton, outputEntry)),
			widget.NewFormItem("Formato", formatSelect),
			widget.NewFormItem("Tipo MOBI", mobiTypeSelect),
			widget.NewFormItem("Jobs", jobsEntry),
			widget.NewFormItem("Timeout", timeoutEntry),
			widget.NewFormItem("Converter", converterEntry),
		),
		container.NewHBox(recursiveCheck, overwriteCheck, verboseCheck, dryRunCheck),
		container.NewHBox(layout.NewSpacer(), cancelButton, startButton),
	)

	content := container.NewBorder(
		form,
		container.NewVBox(widget.NewSeparator(), statusLabel),
		nil,
		nil,
		container.NewScroll(logEntry),
	)

	w.SetContent(content)
	w.ShowAndRun()
}

type uiValues struct {
	inputDir      string
	outputDir     string
	outputFormat  string
	converterPath string
	mobiFileType  string
	jobs          string
	timeout       string
	recursive     bool
	overwrite     bool
	verbose       bool
	dryRun        bool
}

func buildConfigs(values uiValues) (runner.Config, calibre.Config, error) {
	inputDir := strings.TrimSpace(values.inputDir)
	if inputDir == "" {
		return runner.Config{}, calibre.Config{}, errors.New("seleziona una cartella EPUB")
	}

	inputAbs, err := filepath.Abs(inputDir)
	if err != nil {
		return runner.Config{}, calibre.Config{}, fmt.Errorf("input non valido: %w", err)
	}
	info, err := os.Stat(inputAbs)
	if err != nil {
		return runner.Config{}, calibre.Config{}, fmt.Errorf("input non leggibile: %w", err)
	}
	if !info.IsDir() {
		return runner.Config{}, calibre.Config{}, fmt.Errorf("input non valido: %s non e' una cartella", inputAbs)
	}

	outputFormat := strings.ToLower(strings.TrimSpace(values.outputFormat))
	switch outputFormat {
	case "", "mobi":
		outputFormat = "mobi"
	case "azw3":
	default:
		return runner.Config{}, calibre.Config{}, fmt.Errorf("formato non valido: %q", values.outputFormat)
	}

	outputDir := strings.TrimSpace(values.outputDir)
	if outputDir == "" {
		outputDir = filepath.Join(inputAbs, outputFormat)
	}
	outputAbs, err := filepath.Abs(outputDir)
	if err != nil {
		return runner.Config{}, calibre.Config{}, fmt.Errorf("output non valido: %w", err)
	}
	if filepath.Clean(inputAbs) == filepath.Clean(outputAbs) {
		return runner.Config{}, calibre.Config{}, errors.New("la cartella output deve essere diversa dalla cartella input")
	}

	jobs, err := strconv.Atoi(strings.TrimSpace(values.jobs))
	if err != nil || jobs < 1 {
		return runner.Config{}, calibre.Config{}, errors.New("jobs deve essere un numero intero maggiore di zero")
	}
	if jobs > runtime.NumCPU() {
		jobs = runtime.NumCPU()
	}

	timeout, err := time.ParseDuration(strings.TrimSpace(values.timeout))
	if err != nil || timeout <= 0 {
		return runner.Config{}, calibre.Config{}, errors.New("timeout non valido; usa valori come 30m, 10m o 1h")
	}

	mobiFileType := strings.ToLower(strings.TrimSpace(values.mobiFileType))
	switch mobiFileType {
	case "", "old", "new", "both":
		if mobiFileType == "" {
			mobiFileType = "both"
		}
	default:
		return runner.Config{}, calibre.Config{}, fmt.Errorf("tipo MOBI non valido: %q", values.mobiFileType)
	}

	converterPath := strings.TrimSpace(values.converterPath)
	if converterPath == "" {
		converterPath = ebookConvert
	}

	cfg := runner.Config{
		InputDir:      inputAbs,
		OutputDir:     outputAbs,
		OutputFormat:  outputFormat,
		Recursive:     values.recursive,
		Overwrite:     values.overwrite,
		DryRun:        values.dryRun,
		Jobs:          jobs,
		ConverterPath: converterPath,
		Verbose:       values.verbose,
	}
	converterCfg := calibre.Config{
		Command:      converterPath,
		Timeout:      timeout,
		DryRun:       values.dryRun,
		Verbose:      values.verbose,
		MOBIFileType: mobiFileType,
	}

	return cfg, converterCfg, nil
}

type loggingConverter struct {
	inner runner.Converter
	log   *uiLogWriter
}

func (c loggingConverter) Available() error {
	return c.inner.Available()
}

func (c loggingConverter) Convert(ctx context.Context, job runner.Job) error {
	c.log.Printf("Converto: %s\n", job.InputPath)
	err := c.inner.Convert(ctx, job)
	switch {
	case err == nil:
		c.log.Printf("OK: %s\n", job.OutputPath)
	case errors.Is(err, runner.ErrAlreadyExists):
		c.log.Printf("Skip, output gia' presente: %s\n", job.OutputPath)
	default:
		c.log.Printf("Errore: %s\n", err)
	}
	return err
}

type uiLogWriter struct {
	mu    sync.Mutex
	entry *widget.Entry
	buf   strings.Builder
}

func newUILogWriter(entry *widget.Entry) *uiLogWriter {
	return &uiLogWriter{entry: entry}
}

func (w *uiLogWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	w.buf.Write(p)
	text := w.buf.String()
	w.mu.Unlock()

	fyne.Do(func() {
		w.entry.SetText(text)
	})
	return len(p), nil
}

func (w *uiLogWriter) Printf(format string, args ...any) {
	_, _ = w.Write([]byte(fmt.Sprintf(format, args...)))
}
