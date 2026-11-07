package tui

import (
	"bytes"
	"context"
	"io"

	"ssh-keyflow/internal/app"
)

// operationResult keeps independent facts separate from Run's error and Snapshot.
// Allocated target IDs are not proof of persistence; use reports.Saved and view.
type operationResult struct {
	request     app.Request
	target      app.ReportTarget
	reports     []app.Report
	err         error
	diagnostics string
	reloadErr   error
}
type completed struct{ result operationResult }

// execution keeps Bubble Tea's original terminal input for native credentials.
type execution struct {
	ctx     context.Context
	options app.Options
	request app.Request
	target  app.ReportTarget
	reports []app.Report
	output  bytes.Buffer
}

func (operation *execution) SetStdin(input io.Reader)   { operation.options.Input = input }
func (operation *execution) SetStdout(output io.Writer) { operation.options.Output = output }
func (operation *execution) SetStderr(io.Writer)        {}

func (operation *execution) Run() error {
	options := operation.options
	options.Request = &operation.request
	options.Output = io.MultiWriter(options.Output, &operation.output)
	options.Confirm = func(string) (bool, error) { return readNativeConfirmation(operation.ctx, options.Input) }
	options.Report = func(report app.Report) {
		operation.reports = append(operation.reports, report)
		operation.target = mergeTarget(operation.target, report.ReportTarget)
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
	return completed{operationResult{request: operation.request, target: operation.target, reports: operation.reports, err: err, diagnostics: operation.output.String()}}
}

func mergeTarget(old, current app.ReportTarget) app.ReportTarget {
	if current.HostID != "" {
		old.HostID = current.HostID
	}
	if current.KeyID != "" {
		old.KeyID = current.KeyID
	}
	if current.RegistrationID != "" {
		old.RegistrationID = current.RegistrationID
	}
	if current.OnboardingID != "" {
		old.OnboardingID = current.OnboardingID
	}
	if current.CreationID != "" {
		old.CreationID = current.CreationID
	}
	if current.DeletionID != "" {
		old.DeletionID = current.DeletionID
	}
	return old
}

func (m model) handoffTarget() app.ReportTarget {
	target := app.ReportTarget{HostID: m.request.Host, KeyID: m.request.Key, RegistrationID: m.request.Registration}
	for _, r := range m.view.State.Registrations {
		if r.ID == m.request.Registration || r.ID == m.request.Resume {
			target = mergeTarget(target, app.ReportTarget{HostID: r.HostID, KeyID: r.KeyID, RegistrationID: r.ID})
		}
	}
	for _, j := range m.view.State.Onboardings {
		if j.ID == m.request.Resume {
			target = mergeTarget(target, app.ReportTarget{HostID: j.HostID, KeyID: j.KeyID, OnboardingID: j.ID, CreationID: j.CreationID})
		}
	}
	for _, c := range m.view.State.Creations {
		if c.ID == m.request.Resume {
			target = mergeTarget(target, app.ReportTarget{KeyID: c.KeyID, CreationID: c.ID})
		}
	}
	for _, d := range m.view.State.Deletions {
		if d.ID == m.request.Resume {
			target = mergeTarget(target, app.ReportTarget{KeyID: d.KeyID, DeletionID: d.ID})
		}
	}
	return target
}
