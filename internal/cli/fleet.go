package cli

import (
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
	fleet.AddCommand(fleetInitCommand(options), fleetSignCommand(options), fleetShowCommand(options))
	return fleet
}

// fleetChange runs a fleet command that signs: a terminal is required unless
// --dry-run, since native ssh-keygen asks for the passphrase there.
func fleetChange(options Options, request *app.FleetOptions, run func(*cobra.Command, app.FleetOptions) error) func(*cobra.Command, []string) error {
	return func(command *cobra.Command, _ []string) error {
		if !request.DryRun && !terminal(options.Input) {
			return executionError{fmt.Errorf("this operation requires a real terminal for explicit confirmation and the passphrase; --dry-run needs none")}
		}
		home, err := userHome(options)
		if err != nil {
			return executionError{err}
		}
		fleet := *request
		fleet.Home, fleet.Input, fleet.Output, fleet.Error, fleet.Runner = home, options.Input, options.Output, options.Error, options.Runner
		fleet.Confirm = func(string) (bool, error) { return app.ReadConfirmation(options.Input) }
		return asExecutionError(run(command, fleet))
	}
}

func fleetInitCommand(options Options) *cobra.Command {
	request := &app.FleetOptions{}
	command := &cobra.Command{
		Use: "init", Short: "Create the fleet root key and sign roster version 1 (admin node)", Args: cobra.NoArgs,
		PreRunE: func(*cobra.Command, []string) error { return app.CheckLoginAccount(request.LoginAccount) },
		RunE: fleetChange(options, request, func(command *cobra.Command, fleet app.FleetOptions) error {
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
		RunE: fleetChange(options, request, func(command *cobra.Command, fleet app.FleetOptions) error {
			return app.SignFleet(command.Context(), fleet)
		}),
	}
	command.Flags().BoolVar(&request.DryRun, "dry-run", false, "Print the plan without asking or writing")
	return command
}

func fleetShowCommand(options Options) *cobra.Command {
	asJSON, maxAge := false, defaultMaxAge
	command := &cobra.Command{
		Use: "show", Short: "Print the accepted roster and pinned root fingerprints", Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			if maxAge < 1 || maxAge > maxMaxAge {
				return fmt.Errorf("%w: --max-age must be 1 to %d days", app.ErrInvalidInput, maxMaxAge)
			}
			home, err := userHome(options)
			if err != nil {
				return executionError{err}
			}
			now := time.Now()
			view, warnings, err := app.ShowFleet(home, time.Duration(maxAge)*24*time.Hour, now)
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
		},
	}
	command.Flags().BoolVar(&asJSON, "json", false, "Write machine-readable JSON; diagnostics go to stderr")
	command.Flags().IntVar(&maxAge, "max-age", defaultMaxAge, "Warn when the roster is older than this many days")
	return command
}
