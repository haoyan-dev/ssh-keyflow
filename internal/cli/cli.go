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
	RunTUI  func(context.Context, app.Options) error
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
		Use: "ssh-keyflow", Short: "Manage SSH keys, destinations and registration history",
		Version:       options.Version,
		Args:          cobra.NoArgs,
		SilenceErrors: true, SilenceUsage: true,
		RunE: func(command *cobra.Command, _ []string) error {
			if terminal(options.Input) && terminalWriter(options.Output) {
				return openTUI(command.Context(), options)
			}
			return command.Help()
		},
	}
	root.SetVersionTemplate("SSH KeyFlow {{.Version}}\n")
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetArgs(arguments)
	root.SetIn(options.Input)
	root.SetOut(options.Output)
	root.SetErr(options.Error)
	for _, category := range []string{"hosts", "keys", "registrations"} {
		parent := &cobra.Command{
			Use: category, Short: "Manage " + category, Args: cobra.NoArgs,
			RunE: func(command *cobra.Command, _ []string) error { return command.Help() },
		}
		asJSON := false
		list := &cobra.Command{
			Use: "list", Short: "List " + category + " and saved operation progress", Args: cobra.NoArgs,
			RunE: func(command *cobra.Command, _ []string) error {
				home, err := userHome(options)
				if err != nil {
					return executionError{err}
				}
				view, err := app.Snapshot(app.Options{Home: home})
				if err != nil {
					return executionError{err}
				}
				if view.Review != "" {
					fmt.Fprintln(options.Error, view.Review, "(updates stopped)")
				}
				if asJSON {
					return asExecutionError(json.NewEncoder(options.Output).Encode(records(category, view)))
				}
				return asExecutionError(printRecords(options.Output, category, view))
			},
		}
		list.Flags().BoolVar(&asJSON, "json", false, "Write machine-readable JSON; diagnostics go to stderr")
		parent.AddCommand(list)
		switch category {
		case "keys":
			createRequest := &app.Request{Operation: "create"}
			create := operationCommand("create", "Create an Ed25519 key", createRequest, options)
			create.Flags().StringVar(&createRequest.Name, "name", "", "Management name (required)")
			create.MarkFlagRequired("name")
			importRequest := &app.Request{Operation: "import"}
			importKey := operationCommand("import", "Explicitly import an existing key pair or public key", importRequest, options)
			importKey.Flags().StringVar(&importRequest.Name, "name", "", "Management name (required)")
			importKey.Flags().StringVar(&importRequest.PublicPath, "public", "", "Existing public key path (required)")
			importKey.Flags().StringVar(&importRequest.PrivatePath, "private", "", "Existing private key path; omitted for public-only import")
			importKey.MarkFlagRequired("name")
			importKey.MarkFlagRequired("public")
			deleteRequest := &app.Request{Operation: "delete"}
			deleteKey := operationCommand("delete", "Delete managed local files after recorded registrations are removed", deleteRequest, options)
			deleteKey.Flags().StringVar(&deleteRequest.Key, "key", "", "Managed key fixed ID or unambiguous name (required)")
			deleteKey.MarkFlagRequired("key")
			parent.AddCommand(create, importKey, deleteKey)
		case "hosts":
			request := &app.Request{Operation: "add"}
			add := operationCommand("add", "Add a destination and register a new or managed key", request, options)
			add.Flags().StringVar(&request.Destination.Alias, "alias", "", "Destination alias (required)")
			add.Flags().StringVar(&request.Destination.Hostname, "hostname", "", "Hostname, Tailscale name or IP (required)")
			add.Flags().StringVar(&request.Destination.User, "user", "", "Login user (required)")
			add.Flags().IntVar(&request.Destination.Port, "port", 22, "SSH port")
			add.Flags().StringVar(&request.Destination.OS, "os", "", "Confirmed remote OS: linux, macos, windows (required)")
			add.Flags().StringVar(&request.Destination.WindowsAccount, "windows-account", "", "Windows account: standard or administrator (required for Windows)")
			add.Flags().StringVar(&request.Key, "key", "", "Existing managed key fixed ID or unambiguous name")
			add.Flags().StringVar(&request.Name, "new-key", "", "Management name for a new Ed25519 key")
			for _, flag := range []string{"alias", "hostname", "user", "os"} {
				add.MarkFlagRequired(flag)
			}
			add.MarkFlagsOneRequired("key", "new-key")
			add.MarkFlagsMutuallyExclusive("key", "new-key")
			authenticationFlags(add, request)
			parent.AddCommand(add)
		}
		root.AddCommand(parent)
	}
	registerRequest := &app.Request{Operation: "register"}
	register := operationCommand("register", "Register a managed public key at a saved destination", registerRequest, options)
	register.Flags().StringVar(&registerRequest.Host, "host", "", "Destination fixed ID or alias (required)")
	register.Flags().StringVar(&registerRequest.Key, "key", "", "Managed key fixed ID or unambiguous name (required)")
	register.MarkFlagRequired("host")
	register.MarkFlagRequired("key")
	authenticationFlags(register, registerRequest)
	unregisterRequest := &app.Request{Operation: "unregister"}
	unregister := operationCommand("unregister", "Remove one public-key registration using fresh alternate authentication", unregisterRequest, options)
	unregister.Flags().StringVar(&unregisterRequest.Registration, "registration", "", "Registration fixed ID (required)")
	unregister.MarkFlagRequired("registration")
	authenticationFlags(unregister, unregisterRequest)
	resumeRequest := &app.Request{Operation: "resume"}
	resume := operationCommand("resume", "Resume a saved creation, onboarding, registration or local deletion", resumeRequest, options)
	resume.Flags().StringVar(&resumeRequest.Resume, "id", "", "Saved operation fixed ID from a list (required)")
	resume.MarkFlagRequired("id")
	authenticationFlags(resume, resumeRequest)
	root.AddCommand(register, unregister, resume, &cobra.Command{
		Use: "tui", Short: "Open the terminal user interface", Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return openTUI(command.Context(), options)
		},
	})
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

func records(category string, view app.View) map[string]any {
	result := map[string]any{}
	switch category {
	case "hosts":
		result["hosts"], result["onboardings"] = view.State.Hosts, array(view.State.Onboardings)
	case "keys":
		result["keys"], result["creations"] = view.State.Keys, array(view.State.Creations)
		result["deleted_keys"], result["deletions"] = array(view.State.DeletedKeys), array(view.State.Deletions)
	case "registrations":
		result["registrations"], result["history"] = view.State.Registrations, array(view.State.History)
	}
	if category == "hosts" || category == "keys" {
		result["discovery"] = view.Discovery
	}
	return result
}

func array[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}

func printRecords(output io.Writer, category string, view app.View) error {
	table := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	switch category {
	case "hosts":
		fmt.Fprintln(table, "ID\tALIAS\tDESTINATION\tOS")
		for _, host := range view.State.Hosts {
			fmt.Fprintf(table, "%s\t%s\t%s@%s:%d\t%s\n", host.ID, host.Alias, host.User, host.Hostname, host.Port, host.OS)
		}
		for _, job := range view.State.Onboardings {
			fmt.Fprintf(table, "onboarding %s\thost=%s\tkey=%s\tstage=%s\n", job.ID, job.HostID, job.KeyID, job.Stage)
		}
	case "keys":
		fmt.Fprintln(table, "ID\tNAME\tFINGERPRINT\tCONNECTION\tPUBLIC PATH")
		for _, key := range view.State.Keys {
			fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n", key.ID, key.Name, key.Fingerprint, key.ConnectionCheck, key.PublicPath)
		}
		for _, creation := range view.State.Creations {
			fmt.Fprintf(table, "creation %s\t%s\tstage=%s\n", creation.ID, creation.Name, creation.Stage)
		}
		for _, deletion := range view.State.Deletions {
			fmt.Fprintf(table, "deletion %s\tkey=%s\tstage=%s\n", deletion.ID, deletion.KeyID, deletion.Stage)
		}
		for _, deleted := range view.State.DeletedKeys {
			fmt.Fprintf(table, "deleted %s\t%s\t%s\t%s\n", deleted.Key.ID, deleted.Key.Name, deleted.Key.Fingerprint, deleted.DeletedAt)
		}
	case "registrations":
		fmt.Fprintln(table, "ID\tHOST\tKEY\tSTATUS\tCONNECTION\tOPERATION\tSTAGE")
		for _, registration := range view.State.Registrations {
			fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", registration.ID, registration.HostID, registration.KeyID, registration.Status, registration.ConnectionCheck, registration.Operation, registration.Stage)
		}
		for _, event := range view.State.History {
			fmt.Fprintf(table, "history %s\t%s\t%s\t%s\t%s\t%s\t%s\n", event.RegistrationID, event.At, "", event.Status, event.ConnectionCheck, event.Operation, event.Stage)
		}
	}
	if err := table.Flush(); err != nil {
		return err
	}
	if category == "hosts" || category == "keys" {
		_, err := fmt.Fprint(output, "\nDiscovery (read-only; unmanaged files are not imported)\n", view.Discovery)
		return err
	}
	return nil
}

func terminal(input io.Reader) bool {
	file, ok := input.(*os.File)
	return ok && term.IsTerminal(file.Fd())
}

func openTUI(ctx context.Context, options Options) error {
	if !terminal(options.Input) || !terminalWriter(options.Output) {
		return fmt.Errorf("tui requires terminal input and output")
	}
	if options.RunTUI == nil {
		return executionError{fmt.Errorf("TUI entry point is unavailable")}
	}
	home, err := userHome(options)
	if err != nil {
		return executionError{err}
	}
	return asExecutionError(options.RunTUI(ctx, app.Options{Home: home, Input: options.Input, Output: options.Output, Runner: options.Runner}))
}

func terminalWriter(output io.Writer) bool {
	file, ok := output.(*os.File)
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
				return executionError{fmt.Errorf("this operation requires a real terminal for explicit confirmation and OpenSSH authentication")}
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

func authenticationFlags(command *cobra.Command, request *app.Request) {
	command.Flags().StringVar(&request.Bootstrap, "auth", "manual", "Bootstrap/alternate authentication: manual, password, key")
	command.Flags().StringVar(&request.Identity, "identity", "", "Existing private key for key authentication; secrets stay in OpenSSH")
	if request.Operation != "unregister" {
		command.Flags().BoolVar(&request.Configure, "configure", false, "Offer managed SSH configuration and explicit Include consent after registration")
	}
}
