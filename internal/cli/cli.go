package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"ssh-keyflow/internal/app"
)

type Options struct {
	Home    string
	Version string
	Input   io.Reader
	Output  io.Writer
	Error   io.Writer
	Runner  app.Runner
}

// Run is the CLI entry point. It returns the process exit status.
func Run(ctx context.Context, arguments []string, options Options) int {
	if options.Input == nil {
		options.Input = os.Stdin
	}
	if options.Output == nil {
		options.Output = os.Stdout
	}
	if options.Error == nil {
		options.Error = os.Stderr
	}
	if options.Version == "" {
		options.Version = "dev"
	}
	root := &cobra.Command{
		Use: "ssh-keyflow", Short: "Manage this machine's SSH key pairs",
		Version:       options.Version,
		Args:          cobra.NoArgs,
		SilenceErrors: true, SilenceUsage: true,
		RunE: func(command *cobra.Command, _ []string) error { return command.Help() },
	}
	root.SetVersionTemplate("SSH KeyFlow {{.Version}}\n")
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetArgs(arguments)
	root.SetIn(options.Input)
	root.SetOut(options.Output)
	root.SetErr(options.Error)

	keys := &cobra.Command{
		Use: "keys", Short: "Manage local key pairs", Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error { return command.Help() },
	}
	keys.AddCommand(listCommand(options), publicCommand(options))

	createRequest := &app.Request{Operation: "create"}
	create := operationCommand("create", "Create an Ed25519 key pair", createRequest, options)
	create.Flags().StringVar(&createRequest.Name, "name", "", "Management name (required)")
	create.MarkFlagRequired("name")

	importRequest := &app.Request{Operation: "import"}
	importKey := operationCommand("import", "Adopt an existing local key pair; files stay in place", importRequest, options)
	importKey.Flags().StringVar(&importRequest.Name, "name", "", "Management name (required)")
	importKey.Flags().StringVar(&importRequest.PublicPath, "public", "", "Existing public key path (required)")
	importKey.Flags().StringVar(&importRequest.PrivatePath, "private", "", "Matching private key path (required)")
	for _, flag := range []string{"name", "public", "private"} {
		importKey.MarkFlagRequired(flag)
	}

	removeRequest := &app.Request{Operation: "remove"}
	remove := operationCommand("remove", "Stop managing a key pair; key files are preserved", removeRequest, options)
	remove.Flags().StringVar(&removeRequest.Key, "key", "", "Managed key fixed ID or unambiguous name (required)")
	remove.MarkFlagRequired("key")
	keys.AddCommand(create, importKey, remove)

	resumeRequest := &app.Request{Operation: "resume"}
	resume := operationCommand("resume", "Resume an unfinished local key creation", resumeRequest, options)
	resume.Flags().StringVar(&resumeRequest.Resume, "id", "", "Unfinished local creation fixed ID shown by keys list (required)")
	resume.MarkFlagRequired("id")
	root.AddCommand(keys, resume)

	if command, err := root.ExecuteContextC(ctx); err != nil {
		fmt.Fprintln(options.Error, err)
		var failure executionError
		if errors.As(err, &failure) {
			return 1
		}
		if command == nil {
			command = root
		}
		fmt.Fprint(options.Error, command.UsageString())
		return 2
	}
	return 0
}

type executionError struct{ error }

func asExecutionError(err error) error {
	if err != nil {
		if errors.Is(err, app.ErrInvalidInput) {
			return err
		}
		return executionError{err}
	}
	return nil
}

func userHome(options Options) (string, error) {
	if options.Home != "" {
		return options.Home, nil
	}
	return os.UserHomeDir()
}

func listCommand(options Options) *cobra.Command {
	asJSON := false
	command := &cobra.Command{
		Use: "list", Short: "List managed key pairs and unfinished creations", Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			home, err := userHome(options)
			if err != nil {
				return executionError{err}
			}
			view, err := app.Snapshot(app.Options{Home: home})
			if err != nil {
				return executionError{err}
			}
			if view.Review != "" {
				fmt.Fprintln(options.Error, view.Review)
			}
			if asJSON {
				return asExecutionError(json.NewEncoder(options.Output).Encode(map[string]any{
					"keys": array(view.State.Keys), "creations": array(view.State.Creations),
				}))
			}
			table := tabwriter.NewWriter(options.Output, 0, 4, 2, ' ', 0)
			fmt.Fprintln(table, "ID\tNAME\tFINGERPRINT\tPUBLIC PATH")
			for _, key := range view.State.Keys {
				fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", key.ID, key.Name, key.Fingerprint, key.PublicPath)
			}
			for _, creation := range view.State.Creations {
				fmt.Fprintf(table, "creation %s\t%s\tstage=%s\n", creation.ID, creation.Name, creation.Stage)
			}
			return asExecutionError(table.Flush())
		},
	}
	command.Flags().BoolVar(&asJSON, "json", false, "Write machine-readable JSON; diagnostics go to stderr")
	return command
}

func array[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}

func publicCommand(options Options) *cobra.Command {
	var reference string
	command := &cobra.Command{
		Use: "public", Short: "Print a managed public key for copying", Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			home, err := userHome(options)
			if err != nil {
				return executionError{err}
			}
			text, err := app.PublicKey(home, reference)
			if err != nil {
				return asExecutionError(err)
			}
			_, err = fmt.Fprintln(options.Output, text)
			return asExecutionError(err)
		},
	}
	command.Flags().StringVar(&reference, "key", "", "Managed key fixed ID or unambiguous name (required)")
	command.MarkFlagRequired("key")
	return command
}

func terminal(input io.Reader) bool {
	file, ok := input.(*os.File)
	return ok && term.IsTerminal(file.Fd())
}

func operationCommand(name, description string, request *app.Request, options Options) *cobra.Command {
	return &cobra.Command{
		Use: name, Short: description, Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if err := request.Validate(); err != nil {
				return err
			}
			if !terminal(options.Input) {
				return executionError{fmt.Errorf("this operation requires a real terminal for explicit confirmation")}
			}
			home, err := userHome(options)
			if err != nil {
				return executionError{err}
			}
			return asExecutionError(app.Run(command.Context(), app.Options{
				Home: home, Input: options.Input, Output: options.Output, Runner: options.Runner, Request: request,
				Confirm: func(string) (bool, error) { return app.ReadConfirmation(options.Input) },
			}))
		},
	}
}
