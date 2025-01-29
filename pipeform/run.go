package pipeform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/magodo/pipeform/internal/log"
	"github.com/magodo/pipeform/internal/plainui"
	"github.com/magodo/pipeform/internal/reader"
	"github.com/magodo/pipeform/internal/ui"
)

type FlagSet struct {
	LogLevel string
	LogPath  string
	TeePath  string
	TimeCsv  string
	PlainUI  bool
}

type Runner struct {
	Options   *FlagSet
	StartTime time.Time
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
}

func NewRunner(opts *FlagSet) *Runner {
	return &Runner{
		Options:   opts,
		StartTime: time.Now(),
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
	}
}

func (r *Runner) Run(_ context.Context) error {
	logger, err := log.NewLogger(log.Level(r.Options.LogLevel), r.Options.LogPath)
	if err != nil {
		return err
	}
	defer logger.Close()
	teeWriter := io.Discard
	if path := r.Options.TeePath; path != "" {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
		if err != nil {
			return fmt.Errorf("open for tee: %v", err)
		}
		teeWriter = f
		defer f.Close()
	}

	reader := reader.NewReader(r.Stdin, teeWriter)

	type Model interface {
		ToCsv() []byte
		IsEOF() bool
	}

	var model Model

	if r.Options.PlainUI {
		m := plainui.NewRuntimeModel(logger, reader, r.Stdout, r.StartTime)
		if err := m.Run(); err != nil {
			return fmt.Errorf("Error running program: %v\n", err)
		}

		model = m
	} else {
		m := ui.NewRuntimeModel(logger, reader, r.StartTime)
		tm, err := tea.NewProgram(m, tea.WithInputTTY(), tea.WithAltScreen()).Run()
		if err != nil {
			return fmt.Errorf("Error running program: %v\n", err)
		}

		m = tm.(ui.UIModel)

		// Print diags
		for _, diag := range m.Diags() {
			if b, err := json.MarshalIndent(diag, "", "  "); err == nil {
				fmt.Fprintln(r.Stderr, string(b))
			}
		}

		model = m
	}

	if path := r.Options.TimeCsv; path != "" {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
		if err != nil {
			return fmt.Errorf("open time csv file: %v", err)
		}
		defer f.Close()

		if _, err := f.Write(model.ToCsv()); err != nil {
			fmt.Fprintf(r.Stderr, "writing time csv file: %v", err)
		}
	}

	if !model.IsEOF() {
		return errors.New("Interrupted!")
	}

	return nil
}
