package app

import (
	"errors"

	"ssh-keyflow/internal/state"
)

// ReportKind identifies an independent operation fact, not a workflow state.
type ReportKind string

const (
	ReportExecution      ReportKind = "execution"
	ReportConfirmation   ReportKind = "confirmation"
	ReportDeletion       ReportKind = "deletion"
	ReportCreation       ReportKind = "creation"
	ReportOnboarding     ReportKind = "onboarding"
	ReportImport         ReportKind = "import"
	ReportRegistration   ReportKind = "registration"
	ReportAlias          ReportKind = "alias"
	ReportInclude        ReportKind = "include"
	ReportResolvedTarget ReportKind = "target"
	ReportManual         ReportKind = "manual"
	ReportVerification   ReportKind = "verification"
)

// ReportOutcome describes one fact; it is not the overall result of Run.
type ReportOutcome string

const (
	OutcomeStopped        ReportOutcome = "stopped"
	OutcomeDeclined       ReportOutcome = "declined"
	OutcomeRequiresReview ReportOutcome = "requires-review"
	OutcomeReconciled     ReportOutcome = "reconciled"
	OutcomeApplied        ReportOutcome = "applied"
	OutcomeFailed         ReportOutcome = "failed"
	OutcomeUnavailable    ReportOutcome = "unavailable"
	OutcomeSkipped        ReportOutcome = "skipped"
	OutcomePending        ReportOutcome = "pending"
	OutcomeUnknown        ReportOutcome = "unknown"
	OutcomeResolved       ReportOutcome = "resolved"
	OutcomeRequired       ReportOutcome = "required"
	OutcomeVerified       ReportOutcome = "verified"
	OutcomeNotChecked     ReportOutcome = "not-checked"
)

// ReportTarget contains fixed IDs, including allocated IDs on failed saves.
// Confirm persisted identities with Saved and Snapshot before offering recovery.
// Empty IDs mean no identity is known; manual guidance never promises a task.
type ReportTarget struct {
	HostID         string
	KeyID          string
	RegistrationID string
	OnboardingID   string
	CreationID     string
	DeletionID     string
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
	NoChanges       bool
	Status          string
	Stage           string
	Operation       string
	ConnectionCheck string
	Path            string
}

func (options Options) report(report Report) {
	if options.Report == nil {
		return
	}
	report.ReportTarget = options.target
	options.Report(report)
}

func pairTarget(document *state.State, hostID, keyID string) ReportTarget {
	target := ReportTarget{HostID: hostID, KeyID: keyID}
	for _, registration := range document.Registrations {
		if registration.HostID == hostID && registration.KeyID == keyID {
			target.RegistrationID = registration.ID
		}
	}
	for _, job := range document.Onboardings {
		if job.HostID == hostID && job.KeyID == keyID {
			target.OnboardingID, target.CreationID = job.ID, job.CreationID
		}
	}
	return target
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

func approvalOutcome(err error) ReportOutcome {
	if err == nil || errors.Is(err, ErrCancelled) {
		return OutcomeDeclined
	}
	return OutcomeUnknown
}
