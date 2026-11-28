package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"ssh-keyflow/internal/app"
)

// defaultMaxAge is --max-age in days (roster-format 3, decision B).
const defaultMaxAge = 30

// maxMaxAge (100 years) keeps --max-age far below time.Duration's limit.
const maxMaxAge = 36500

func fleetCommand(options Options) *cobra.Command {
	fleet := &cobra.Command{
		Use: "fleet", Short: "Create and sign the fleet roster", Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error { return command.Help() },
	}
	fleet.AddCommand(fleetInitCommand(options), fleetSignCommand(options), fleetAddCommand(options), fleetShowCommand(options),
		exportCommand(options, "Write the accepted roster, its signature and the fleet root public key", app.ExportFleet),
		fleetAcceptCommand(options))
	return fleet
}

// maxAgeFlag adds --max-age and returns a check that converts it; a value
// outside 1 to maxMaxAge days is a usage error.
func maxAgeFlag(command *cobra.Command) func() (time.Duration, error) {
	days := defaultMaxAge
	command.Flags().IntVar(&days, "max-age", defaultMaxAge, "Warn when the roster is older than this many days")
	return func() (time.Duration, error) {
		if days < 1 || days > maxMaxAge {
			return 0, fmt.Errorf("%w: --max-age must be 1 to %d days", app.ErrInvalidInput, maxMaxAge)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
}

// checkDirectory rejects a missing or unreadable `<dir>` as a usage error,
// before the terminal check.
func checkDirectory(options Options) func(*cobra.Command, []string) error {
	return func(_ *cobra.Command, arguments []string) error {
		home, err := userHome(options)
		if err != nil {
			return executionError{err}
		}
		return app.CheckInputDirectory(home, arguments[0])
	}
}

// exportCommand is `node export` or `fleet export`: no state change, so no
// confirmation; --out must be absent or empty.
func exportCommand(options Options, short string, export func(app.ExportOptions) error) *cobra.Command {
	out := ""
	command := &cobra.Command{
		Use: "export", Short: short, Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			home, err := userHome(options)
			if err != nil {
				return executionError{err}
			}
			return asExecutionError(export(app.ExportOptions{Home: home, Out: out, Output: options.Output, Error: options.Error}))
		},
	}
	command.Flags().StringVar(&out, "out", "", "Output directory, absent or empty (required)")
	command.MarkFlagRequired("out")
	return command
}

// directoryCommand is `fleet add <dir>` or `fleet accept <dir>`: `<dir>` is
// checked as a usage error before anything else runs.
func directoryCommand(options Options, request *app.FleetOptions, need terminalNeed, use, short string, run func(context.Context, app.FleetOptions) error) *cobra.Command {
	command := &cobra.Command{
		Use: use, Short: short, Args: cobra.ExactArgs(1), PreRunE: checkDirectory(options),
		RunE: fleetChange(options, request, need, func(command *cobra.Command, fleet app.FleetOptions) error {
			fleet.Dir = command.Flags().Arg(0)
			return run(command.Context(), fleet)
		}),
	}
	command.Flags().BoolVar(&request.DryRun, "dry-run", false, "Print the plan without asking or writing")
	return command
}

func fleetAddCommand(options Options) *cobra.Command {
	return directoryCommand(options, &app.FleetOptions{}, signingTerminal, "add <dir>",
		"Add or replace the node in <dir>/node.json and sign the next roster version (admin node)", app.AddNode)
}

func fleetAcceptCommand(options Options) *cobra.Command {
	request := &app.FleetOptions{}
	command := directoryCommand(options, request, confirmationTerminal, "accept <dir>", "Verify and accept the roster exported to <dir>", app.AcceptFleet)
	readMaxAge, checkDir := maxAgeFlag(command), command.PreRunE
	command.PreRunE = func(command *cobra.Command, arguments []string) (err error) {
		if request.MaxAge, err = readMaxAge(); err != nil {
			return err
		}
		return checkDir(command, arguments)
	}
	return command
}

// terminalNeed is when and why a fleet change needs a real terminal.
type terminalNeed int

const (
	// signingTerminal is checked before planning: native ssh-keygen asks for
	// the passphrase there.
	signingTerminal terminalNeed = iota
	// confirmationTerminal is checked only once the change needs a prompt, so
	// accepting the same roster again needs none (roster-format 2 rule 5).
	confirmationTerminal
)

// fleetChange runs a fleet command that changes state; a terminal is
// required unless --dry-run.
func fleetChange(options Options, request *app.FleetOptions, need terminalNeed, run func(*cobra.Command, app.FleetOptions) error) func(*cobra.Command, []string) error {
	return func(command *cobra.Command, _ []string) error {
		if !request.DryRun && need == signingTerminal && !terminal(options.Input) {
			return executionError{fmt.Errorf("this operation requires a real terminal for explicit confirmation and the passphrase; --dry-run needs none")}
		}
		home, err := userHome(options)
		if err != nil {
			return executionError{err}
		}
		fleet := *request
		fleet.Home, fleet.Input, fleet.Output, fleet.Error, fleet.Runner = home, options.Input, options.Output, options.Error, options.Runner
		fleet.Confirm = func(string) (bool, error) { return app.ReadConfirmation(options.Input) }
		if need == confirmationTerminal {
			fleet.RequireTerminal = func() error {
				if !terminal(options.Input) {
					return fmt.Errorf("this operation requires a real terminal for explicit confirmation; --dry-run needs none")
				}
				return nil
			}
		}
		return asExecutionError(run(command, fleet))
	}
}

func fleetInitCommand(options Options) *cobra.Command {
	request := &app.FleetOptions{}
	command := &cobra.Command{
		Use: "init", Short: "Create the fleet root key and sign roster version 1 (admin node)", Args: cobra.NoArgs,
		PreRunE: func(*cobra.Command, []string) error { return app.CheckLoginAccount(request.LoginAccount) },
		RunE: fleetChange(options, request, signingTerminal, func(command *cobra.Command, fleet app.FleetOptions) error {
			return app.InitFleet(command.Context(), fleet)
		}),
	}
	command.Flags().StringVar(&request.LoginAccount, "login-account", "", "The fleet's common login account (required)")
	command.Flags().BoolVar(&request.DryRun, "dry-run", false, "Print the plan without asking or writing")
	command.MarkFlagRequired("login-account")
	return command
}

func fleetSignCommand(options Options) *cobra.Command {
	request := &app.FleetOptions{}
	command := &cobra.Command{
		Use: "sign", Short: "Sign the next roster version with this node's current description (admin node)", Args: cobra.NoArgs,
		RunE: fleetChange(options, request, signingTerminal, func(command *cobra.Command, fleet app.FleetOptions) error {
			return app.SignFleet(command.Context(), fleet)
		}),
	}
	command.Flags().BoolVar(&request.DryRun, "dry-run", false, "Print the plan without asking or writing")
	return command
}

func fleetShowCommand(options Options) *cobra.Command {
	asJSON := false
	command := &cobra.Command{
		Use: "show", Short: "Print the accepted roster and pinned root fingerprints", Args: cobra.NoArgs,
	}
	readMaxAge := maxAgeFlag(command)
	command.RunE = func(*cobra.Command, []string) error {
		maxAge, err := readMaxAge()
		if err != nil {
			return err
		}
		home, err := userHome(options)
		if err != nil {
			return executionError{err}
		}
		now := time.Now()
		view, warnings, err := app.ShowFleet(home, maxAge, now)
		if err != nil {
			return executionError{err}
		}
		for _, warning := range warnings {
			fmt.Fprintln(options.Error, warning)
		}
		if asJSON {
			return asExecutionError(json.NewEncoder(options.Output).Encode(view))
		}
		if view.Fleet == nil {
			_, err := fmt.Fprintln(options.Output, "No accepted roster yet; run fleet init on the admin node.")
			return asExecutionError(err)
		}
		_, err = fmt.Fprint(options.Output, app.DescribeRoster(*view.Fleet, now))
		return asExecutionError(err)
	}
	command.Flags().BoolVar(&asJSON, "json", false, "Write machine-readable JSON; diagnostics go to stderr")
	return command
}
