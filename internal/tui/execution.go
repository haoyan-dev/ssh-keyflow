package tui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"ssh-keyflow/internal/app"
)

type operationResult struct {
	request        app.Request
	reports        []app.Report
	err, reloadErr error
	diagnostics    string
}
type completed struct{ result operationResult }

// Only native processes receive stdin; credentials never enter TUI text fields.
type execution struct {
	ctx     context.Context
	options app.Options
	request app.Request
	reports []app.Report
	output  diagnosticBuffer
}

func (operation *execution) SetStdin(input io.Reader)   { operation.options.Input = input }
func (operation *execution) SetStdout(output io.Writer) { operation.options.Output = output }
func (operation *execution) SetStderr(io.Writer)        {}
func (operation *execution) Run() error {
	options := operation.options
	options.Request = &operation.request
	if options.Output == nil {
		options.Output = io.Discard
	}
	options.Output = io.MultiWriter(options.Output, &operation.output)
	options.Confirm = func(string) (bool, error) { return readNativeConfirmation(operation.ctx, options.Input) }
	options.Report = func(report app.Report) {
		operation.reports = append(operation.reports, report)
		if operation.options.Report != nil {
			operation.options.Report(report)
		}
	}
	if err := operation.ctx.Err(); err != nil {
		return err
	}
	return app.Run(operation.ctx, options)
}
func (operation *execution) complete(err error) completed {
	return completed{operationResult{request: operation.request, reports: operation.reports, err: err, diagnostics: operation.output.String()}}
}

type diagnosticBuffer struct {
	bytes.Buffer
	truncated bool
}

func (buffer *diagnosticBuffer) Write(data []byte) (int, error) {
	count := len(data)
	remaining := max(0, 32768-buffer.Len())
	if len(data) > remaining {
		buffer.truncated = true
		data = data[:remaining]
	}
	_, err := buffer.Buffer.Write(data)
	return count, err
}
func (buffer *diagnosticBuffer) String() string {
	text := safeText(buffer.Buffer.String())
	if buffer.truncated {
		text += "\n[diagnostics truncated]"
	}
	return text
}
func (result operationResult) text() string {
	title := "Local operation completed"
	if errors.Is(result.err, app.ErrCancelled) {
		title = "Cancelled before further changes"
	} else if result.err != nil {
		title = "Operation stopped; saved checkpoints remain"
	}
	var text strings.Builder
	fmt.Fprintf(&text, "%s\n%s\n", title, actionTitle(result.request.Operation))
	if result.request.Key != "" {
		fmt.Fprintf(&text, "Key: %s\n", result.request.Key)
	}
	if result.request.Host != "" {
		fmt.Fprintf(&text, "Destination: %s\n", result.request.Host)
	}
	for _, report := range result.reports {
		fmt.Fprintf(&text, "\n%s: %s", report.Kind, report.Outcome)
		if report.Stage != "" {
			fmt.Fprintf(&text, " · %s", report.Stage)
		}
		if report.Saved {
			text.WriteString(" (saved)")
		}
		if report.KeyID != "" {
			fmt.Fprintf(&text, "\nKey: %s", report.KeyID)
		}
		if report.HostID != "" {
			fmt.Fprintf(&text, "\nDestination: %s", report.HostID)
		}
		if report.CreationID != "" {
			fmt.Fprintf(&text, "\nCreation: %s", report.CreationID)
		}
	}
	if result.err != nil {
		fmt.Fprintf(&text, "\n\nError: %v", result.err)
	}
	if result.reloadErr != nil {
		fmt.Fprintf(&text, "\n\nReload failed: %v\nSaved state is stale. F5 refresh before continuing.", result.reloadErr)
	}
	text.WriteString("\n\nRemote placement and connection are unverified.\nEsc: Keys / Destinations. Select unfinished local creation and press s to resume.")
	if result.diagnostics != "" {
		text.WriteString("\n\nNative diagnostics:\n" + result.diagnostics)
	}
	return text.String()
}
