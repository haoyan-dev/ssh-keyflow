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
		Use: "ssh-keyflow", Short: "Manage local SSH keys and copy-paste destination setup",
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
	for _, category := range []string{"hosts", "keys"} {
		parent := &cobra.Command{
			Use: category, Short: "Manage " + category, Args: cobra.NoArgs,
			RunE: func(command *cobra.Command, _ []string) error { return command.Help() },
		}
		asJSON := false
		list := &cobra.Command{
			Use: "list", Short: "List saved " + category, Args: cobra.NoArgs,
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
					fmt.Fprintln(options.Error, view.Review)
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
			publicRequest := &app.Request{Operation: "public"}
			public := readCommand("public", "Print a managed public key for copying", publicRequest, options)
			public.Flags().StringVar(&publicRequest.Key, "key", "", "Managed key fixed ID or unambiguous name (required)")
			public.MarkFlagRequired("key")
			removeRequest := &app.Request{Operation: "remove-key"}
			remove := operationCommand("remove", "Remove a managed key and its local associations; preserve key files", removeRequest, options)
			remove.Flags().StringVar(&removeRequest.Key, "key", "", "Managed key fixed ID or unambiguous name (required)")
			remove.MarkFlagRequired("key")
			parent.AddCommand(create, importKey, public, remove)
		case "hosts":
			request := &app.Request{Operation: "add"}
			add := operationCommand("add", "Save a destination associated with an existing key; no remote changes", request, options)
			add.Flags().StringVar(&request.Destination.Alias, "alias", "", "Destination alias (required)")
			add.Flags().StringVar(&request.Destination.Hostname, "hostname", "", "Hostname, Tailscale name or IP (required)")
			add.Flags().StringVar(&request.Destination.User, "user", "", "Login user (required)")
			add.Flags().IntVar(&request.Destination.Port, "port", 22, "SSH port")
			add.Flags().StringVar(&request.Destination.OS, "os", "", "Destination OS: linux, macos, windows (required)")
			add.Flags().StringVar(&request.Destination.WindowsAccount, "windows-account", "", "Windows account: standard or administrator (required for Windows)")
			add.Flags().StringVar(&request.Key, "key", "", "Existing managed key fixed ID or unambiguous name (required)")
			for _, flag := range []string{"alias", "hostname", "user", "os", "key"} {
				add.MarkFlagRequired(flag)
			}
			associateRequest := &app.Request{Operation: "associate"}
			associate := operationCommand("associate", "Save a key association; remote registration is not verified", associateRequest, options)
			dissociateRequest := &app.Request{Operation: "dissociate"}
			dissociate := operationCommand("dissociate", "Disconnect a local key association; preserve keys, destination and SSH access", dissociateRequest, options)
			removeRequest := &app.Request{Operation: "remove-host"}
			remove := operationCommand("remove", "Remove a destination and its local associations; preserve SSH access", removeRequest, options)
			remove.Flags().StringVar(&removeRequest.Host, "host", "", "Destination fixed ID or alias (required)")
			remove.MarkFlagRequired("host")
			guideRequest := &app.Request{Operation: "guide"}
			guide := readCommand("guide", "Print copy-paste setup and removal instructions without changing files", guideRequest, options)
			guide.Flags().StringVar(&guideRequest.GuideOptions.LocalOS, "local-os", "", "Local terminal OS: linux, macos, windows (default: running app OS)")
			guide.Flags().StringVar(&guideRequest.GuideOptions.RemoteOS, "remote-os", "", "Destination OS: linux, macos, windows (default: saved destination)")
			guide.Flags().StringVar(&guideRequest.GuideOptions.WindowsAccount, "windows-account", "", "Destination Windows account: standard or administrator (default: saved account)")
			guide.Flags().StringVar(&guideRequest.GuideOptions.IdentityFile, "identity-file", "", "Matching private-key path in the local terminal (required when local OS differs)")
			for command, request := range map[*cobra.Command]*app.Request{associate: associateRequest, dissociate: dissociateRequest, guide: guideRequest} {
				command.Flags().StringVar(&request.Host, "host", "", "Destination fixed ID or alias (required)")
				command.Flags().StringVar(&request.Key, "key", "", "Managed key fixed ID or unambiguous name (required)")
				command.MarkFlagRequired("host")
				command.MarkFlagRequired("key")
			}
			parent.AddCommand(add, associate, dissociate, guide, remove)
		}
		root.AddCommand(parent)
	}
	resumeRequest := &app.Request{Operation: "resume"}
	resume := operationCommand("resume", "Resume an unfinished local key creation", resumeRequest, options)
	resume.Flags().StringVar(&resumeRequest.Resume, "id", "", "Unfinished local creation fixed ID shown by keys list (required)")
	resume.MarkFlagRequired("id")
	root.AddCommand(resume, &cobra.Command{
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
	if category == "hosts" {
		return map[string]any{"hosts": array(view.State.Hosts), "associations": array(view.State.Associations)}
	}
	return map[string]any{"keys": array(view.State.Keys), "creations": array(view.State.Creations)}
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
		for _, association := range view.State.Associations {
			fmt.Fprintf(table, "association\thost=%s\tkey=%s\n", association.HostID, association.KeyID)
		}
	case "keys":
		fmt.Fprintln(table, "ID\tNAME\tFINGERPRINT\tPUBLIC PATH")
		for _, key := range view.State.Keys {
			fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", key.ID, key.Name, key.Fingerprint, key.PublicPath)
		}
		for _, creation := range view.State.Creations {
			if creation.Stage != "complete" {
				fmt.Fprintf(table, "creation %s\t%s\tstage=%s\n", creation.ID, creation.Name, creation.Stage)
			}
		}
	}
	return table.Flush()
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

func readCommand(name, description string, request *app.Request, options Options) *cobra.Command {
	return &cobra.Command{
		Use: name, Short: description, Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := request.Validate(); err != nil {
				return err
			}
			home, err := userHome(options)
			if err != nil {
				return executionError{err}
			}
			var text string
			if request.Operation == "public" {
				text, err = app.PublicKey(home, request.Key)
			} else {
				var guide app.SetupGuide
				guide, err = app.PrepareGuideWithOptions(home, request.Host, request.Key, request.GuideOptions)
				text = guide.Text
			}
			if err != nil {
				return asExecutionError(err)
			}
			_, err = fmt.Fprint(options.Output, text)
			return asExecutionError(err)
		},
	}
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
