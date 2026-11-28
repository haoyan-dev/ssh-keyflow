package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/roster"
)

func nodeCommand(options Options) *cobra.Command {
	node := &cobra.Command{
		Use: "node", Short: "Describe this machine as a fleet node", Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error { return command.Help() },
	}
	node.AddCommand(nodeSetCommand(options), nodeShowCommand(options))
	return node
}

func nodeSetCommand(options Options) *cobra.Command {
	request := app.NodeRequest{}
	command := &cobra.Command{
		Use: "set", Short: "Create or replace this machine's node description", Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if command.Flags().Changed("port") && request.Port < 1 {
				return fmt.Errorf("%w: --port must be 1-65535", app.ErrInvalidInput)
			}
			if err := request.Validate(); err != nil {
				return err
			}
			if !request.DryRun && !terminal(options.Input) {
				return executionError{fmt.Errorf("this operation requires a real terminal for explicit confirmation; --dry-run needs none")}
			}
			home, err := userHome(options)
			if err != nil {
				return executionError{err}
			}
			return asExecutionError(app.SetNode(app.NodeOptions{
				Home: home, Output: options.Output, Error: options.Error, Request: request,
				Confirm: func(string) (bool, error) { return app.ReadConfirmation(options.Input) },
			}))
		},
	}
	flags := command.Flags()
	flags.StringVar(&request.Name, "name", "", "Node name: one lowercase DNS label (required)")
	flags.Var(&listFlag{target: &request.Roles}, "roles", "Comma-separated roles: client, remote (required)")
	flags.StringVar(&request.Key, "key", "", "Client role: managed key fixed ID or unambiguous name")
	flags.Var(&listFlag{target: &request.Addresses}, "addresses", "Remote role: comma-separated DNS names or IP addresses")
	flags.IntVar(&request.Port, "port", 0, "Remote role: SSH port (default 22)")
	flags.StringVar(&request.LoginAccount, "login-account", "", "Remote role: login account if not the fleet's common account")
	flags.StringVar(&request.SSHDConfig, "sshd-config", "", "Remote role: sshd_config to read HostKey entries from (default: the platform's)")
	flags.BoolVar(&request.DryRun, "dry-run", false, "Print the plan without asking or writing")
	command.MarkFlagRequired("name")
	command.MarkFlagRequired("roles")
	return command
}

// listFlag takes exactly one comma-separated value (roster-format 3); a
// repeated flag is a usage error instead of being merged or overridden.
type listFlag struct {
	target *[]string
	set    bool
}

func (flag *listFlag) String() string {
	if flag.target == nil {
		return ""
	}
	return strings.Join(*flag.target, ",")
}

func (flag *listFlag) Set(value string) error {
	if flag.set {
		return fmt.Errorf("give it once, as one comma-separated value")
	}
	flag.set = true
	*flag.target = strings.Split(value, ",")
	return nil
}

func (*listFlag) Type() string { return "list" }

func nodeShowCommand(options Options) *cobra.Command {
	asJSON := false
	command := &cobra.Command{
		Use: "show", Short: "Print this machine's node description", Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			home, err := userHome(options)
			if err != nil {
				return executionError{err}
			}
			view, review, err := app.ShowNode(home)
			if err != nil {
				return executionError{err}
			}
			if review != "" {
				fmt.Fprintln(options.Error, review)
			}
			if asJSON {
				return asExecutionError(json.NewEncoder(options.Output).Encode(view))
			}
			if view.Node == nil {
				_, err := fmt.Fprintln(options.Output, "No node description yet; run node set.")
				return asExecutionError(err)
			}
			node := view.Node
			var text strings.Builder
			fmt.Fprintf(&text, "ID: %s\nName: %s\nRoles: %s\nAdmin node: %t\n", node.ID, node.Name, strings.Join(node.Roles, ","), view.IsAdmin)
			describe := func(label, line string, host bool) {
				if key, err := roster.ParsePublicKey(line, host); err == nil {
					fmt.Fprintf(&text, "%s: %s %s\n", label, key.Type, key.Fingerprint)
				} else {
					fmt.Fprintf(&text, "%s: unreadable (%v); compare with --json\n", label, err)
				}
			}
			for _, key := range node.ClientKeys {
				describe("Client key", key, false)
			}
			if len(node.Addresses) > 0 {
				port := node.Port
				if port == 0 {
					port = roster.DefaultPort
				}
				fmt.Fprintf(&text, "Addresses: %s\nPort: %d\n", strings.Join(node.Addresses, ", "), port)
				if node.LoginAccount != "" {
					fmt.Fprintf(&text, "Login account: %s\n", node.LoginAccount)
				}
			}
			for _, key := range node.HostKeys {
				describe("Host key", key, true)
			}
			_, err = fmt.Fprint(options.Output, text.String())
			return asExecutionError(err)
		},
	}
	command.Flags().BoolVar(&asJSON, "json", false, "Write machine-readable JSON; diagnostics go to stderr")
	return command
}
