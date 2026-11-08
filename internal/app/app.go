package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Command struct {
	Name        string
	Args        []string
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	Interactive bool
}

type Runner interface {
	LookPath(string) (string, error)
	Run(context.Context, Command) error
}

type SystemRunner struct{}

func (SystemRunner) LookPath(name string) (string, error) { return exec.LookPath(name) }
func (SystemRunner) Run(ctx context.Context, command Command) error {
	if command.Interactive {
		terminal, valid := command.Stdin.(*os.File)
		if !valid {
			return fmt.Errorf("a real terminal is required for credential input")
		}
		info, err := terminal.Stat()
		if err != nil || info.Mode()&os.ModeCharDevice == 0 {
			return fmt.Errorf("a real terminal is required for credential input")
		}
	}
	process := exec.CommandContext(ctx, command.Name, command.Args...)
	process.Stdin, process.Stdout, process.Stderr = command.Stdin, command.Stdout, command.Stderr
	return process.Run()
}

type Options struct {
	// Report receives synchronous informational facts; it must not mutate storage
	// or run operations (the operation lock may be held). Nil preserves CLI behavior.
	Report  func(Report)
	target  ReportTarget
	Home    string
	Input   io.Reader
	Output  io.Writer
	Runner  Runner
	Request *Request
	Confirm func(string) (bool, error)
}

func Run(ctx context.Context, options Options) (err error) {
	if options.Report != nil && options.Request != nil {
		sink := options.Report
		var target ReportTarget
		effectsPossible := false
		options.Report = func(report Report) {
			if report.ReportTarget != (ReportTarget{}) {
				target = report.ReportTarget
			}
			if report.Kind == ReportConfirmation && report.Outcome == OutcomeDeclined {
				report.NoChanges = !effectsPossible
			}
			if report.Saved || report.Outcome == OutcomeUnknown {
				effectsPossible = true
			}
			sink(report)
		}
		defer func() {
			if err != nil {
				sink(Report{ReportTarget: target, Kind: ReportExecution, Outcome: OutcomeStopped})
			}
		}()
	}
	if options.Home == "" {
		return fmt.Errorf("Home is required")
	}
	if options.Input == nil {
		options.Input = os.Stdin
	}
	if options.Output == nil {
		options.Output = os.Stdout
	}
	if options.Runner == nil {
		options.Runner = SystemRunner{}
	}
	if options.Request != nil {
		request := *options.Request
		options.Request = &request
		return execute(ctx, options)
	}
	return fmt.Errorf("an explicit operation request is required")
}

func readLine(input io.Reader) (string, error) {
	var line strings.Builder
	buffer := make([]byte, 1)
	for {
		count, err := input.Read(buffer)
		if count > 0 {
			if buffer[0] == '\n' {
				return strings.TrimSpace(line.String()), nil
			}
			line.WriteByte(buffer[0])
		}
		if err != nil {
			return strings.TrimSpace(line.String()), err
		}
	}
}

// ReadConfirmation leaves subsequent credential input for the native process.
func ReadConfirmation(input io.Reader) (bool, error) {
	answer, err := readLine(input)
	return answer == "yes", err
}
