package app

import "ssh-keyflow/internal/state"

// ReportKind identifies an independent operation fact, not a workflow state.
type ReportKind string

const (
	ReportExecution      ReportKind = "execution"
	ReportConfirmation   ReportKind = "confirmation"
	ReportCreation       ReportKind = "creation"
	ReportImport         ReportKind = "import"
	ReportDestination    ReportKind = "destination"
	ReportResolvedTarget ReportKind = "target"
)

// ReportOutcome describes one fact; it is not the overall result of Run.
type ReportOutcome string

const (
	OutcomeStopped        ReportOutcome = "stopped"
	OutcomeDeclined       ReportOutcome = "declined"
	OutcomeRequiresReview ReportOutcome = "requires-review"
	OutcomeApplied        ReportOutcome = "applied"
	OutcomePending        ReportOutcome = "pending"
	OutcomeUnknown        ReportOutcome = "unknown"
	OutcomeResolved       ReportOutcome = "resolved"
)

// ReportTarget contains fixed IDs, including allocated IDs on failed saves.
// Confirm persisted identities with Saved and Snapshot before offering recovery.
// Empty IDs mean no identity is known; manual guidance never promises a task.
type ReportTarget struct {
	HostID     string
	KeyID      string
	CreationID string
}

// Report is a current-operation fact. Saved is true only after persistence
// succeeds; a saved pending checkpoint is not completion. Observations and manual
// guidance do not replace saved evidence.
// Reports contain no credentials, native output, or error text. Keep Run's
// error and diagnostics and reload Snapshot separately; absent facts are unknown.
type Report struct {
	ReportTarget
	Kind    ReportKind
	Outcome ReportOutcome
	Saved   bool
	// NoChanges is asserted only for a structured confirmation denial before
	// any saved checkpoint or uncertain effect in this Run. False is no guarantee.
	NoChanges bool
	Stage     string
	Path      string
}

func (options Options) report(report Report) {
	if options.Report == nil {
		return
	}
	report.ReportTarget = options.target
	options.Report(report)
}

// saveReport never turns an attempted checkpoint into a committed fact.
func (options Options) saveReport(store *state.LockedStore, document state.State, report Report) error {
	if err := store.Save(document); err != nil {
		options.report(Report{Kind: report.Kind, Outcome: OutcomeUnknown, Path: report.Path})
		return err
	}
	report.Saved = true
	options.report(report)
	return nil
}
