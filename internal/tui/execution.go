package tui

import (
	"bytes"
	"context"
	"io"

	"ssh-keyflow/internal/app"
)

type completed struct {
	err    error
	output string
}

// execution implements Bubble Tea's terminal handoff interface.
type execution struct {
	ctx     context.Context
	options app.Options
	request app.Request
	output  bytes.Buffer
}

func (operation *execution) SetStdin(input io.Reader)   { operation.options.Input = input }
func (operation *execution) SetStdout(output io.Writer) { operation.options.Output = output }
func (operation *execution) SetStderr(io.Writer)        {}

func (operation *execution) Run() error {
	options := operation.options
	options.Request = &operation.request
	options.Output = io.MultiWriter(options.Output, &operation.output)
	options.Confirm = func(string) (bool, error) { return app.ReadConfirmation(options.Input) }
	return app.Run(operation.ctx, options)
}
