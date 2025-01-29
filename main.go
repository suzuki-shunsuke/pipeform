package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/magodo/pipeform/internal/log"
	"github.com/magodo/pipeform/pipeform"
	"github.com/urfave/cli/v3"
)

var fset pipeform.FlagSet

func main() {
	cmd := &cli.Command{
		Name:  "pipeform",
		Usage: "Terraform UI by running like: `terraform ... -json | pipeform`",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "log-level",
				Usage:       "The log level",
				Sources:     cli.EnvVars("PF_LOG"),
				Value:       string(log.LevelDebug),
				Destination: &fset.LogLevel,
				Validator: func(input string) error {
					if !slices.Contains(log.PossibleLevels(), log.Level(strings.ToLower(input))) {
						return fmt.Errorf("invalid log level: %s", input)
					}
					return nil
				},
			},
			&cli.StringFlag{
				Name:        "log-path",
				Usage:       "The log path",
				Sources:     cli.EnvVars("PF_LOG_PATH"),
				Destination: &fset.LogPath,
			},
			&cli.StringFlag{
				Name:        "tee",
				Usage:       `Equivalent to "terraform ... -json | tee <value> | pipeform"`,
				Sources:     cli.EnvVars("PF_TEE"),
				Destination: &fset.TeePath,
			},
			&cli.StringFlag{
				Name:        "time-csv",
				Usage:       "The csv file that records the timing of each operation of each resource",
				Sources:     cli.EnvVars("PF_TIME_CSV"),
				Destination: &fset.TimeCsv,
			},
			&cli.BoolFlag{
				Name:        "plain-ui",
				Usage:       "Simply print each log line by line, that expect to use in systems only support plain output",
				Sources:     cli.EnvVars("PF_PLAIN_UI"),
				Destination: &fset.PlainUI,
			},
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			// If this program starts in standalone, its stdin is the same as the terminal.
			// bubbletea will change the terminal into raw mode and read ansi events from it,
			// which conflicts with the stdin reading for terraform JSON streams.
			// In this case, user's input (e.g. ctrl-c keypress) will most likely be accidently read by
			// the stream reader, instead of the ansi read loop (by bubbletea), causing a lost of event.
			if term.IsTerminal(os.Stdin.Fd()) {
				return ctx, errors.New("Must be followed by a pipe")
			}
			return ctx, nil
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			return action(ctx)
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func action(ctx context.Context) error {
	r := pipeform.NewRunner(&fset)
	return r.Run(ctx)
}
