package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"ssh-keyflow/internal/roster"
	"ssh-keyflow/internal/sshd"
	"ssh-keyflow/internal/state"
)

// NodeRequest describes this machine as a node. Each `node set` gives the
// whole description; the node id is kept across runs.
type NodeRequest struct {
	Name         string
	Roles        []string
	Key          string
	Addresses    []string
	Port         int
	LoginAccount string
	SSHDConfig   string
	DryRun       bool
	// SSHDir overrides sshd's directory for Include and default host keys (tests).
	SSHDir string
}

// Validate checks the flags alone, before any file is read. Failures are usage errors.
func (request *NodeRequest) Validate() error {
	invalid := func(format string, arguments ...any) error {
		return fmt.Errorf("%w: %s", ErrInvalidInput, fmt.Sprintf(format, arguments...))
	}
	if !roster.ValidName(request.Name) {
		return invalid("--name must be one lowercase DNS label (a-z, 0-9, -; at most 63 characters)")
	}
	roles, err := roster.Roles(request.Roles)
	if err != nil {
		return invalid("--roles: %v", err)
	}
	request.Roles = roles
	client, remote := slices.Contains(roles, roster.RoleClient), slices.Contains(roles, roster.RoleRemote)
	if client != (request.Key != "") {
		return invalid("--key is required for, and only allowed with, the client role")
	}
	if !remote && (len(request.Addresses) > 0 || request.Port != 0 || request.LoginAccount != "" || request.SSHDConfig != "") {
		return invalid("--addresses, --port, --login-account and --sshd-config need the remote role")
	}
	if !remote {
		return nil
	}
	if len(request.Addresses) == 0 || len(request.Addresses) > roster.MaxAddresses {
		return invalid("the remote role needs 1 to %d --addresses", roster.MaxAddresses)
	}
	for _, address := range request.Addresses {
		if !roster.ValidAddress(address) {
			return invalid("--addresses: %q is not a lowercase DNS name or canonical IP address (no port, no brackets)", address)
		}
	}
	addresses := slices.Clone(request.Addresses)
	slices.Sort(addresses)
	if len(slices.Compact(addresses)) != len(request.Addresses) {
		return invalid("--addresses lists an address twice")
	}
	request.Addresses = addresses
	if request.Port < 0 || request.Port > 65535 {
		return invalid("--port must be 1-65535")
	}
	if request.Port == roster.DefaultPort {
		request.Port = 0
	}
	if request.LoginAccount != "" && !roster.ValidAccount(request.LoginAccount) {
		return invalid("--login-account: letters, digits, '.', '_' and '-', at most 64 characters")
	}
	return nil
}

// NodeOptions runs `node set`.
type NodeOptions struct {
	Home   string
	Output io.Writer
	// Error receives the migration notice of a dry run, which writes nothing
	// and so is not confirmed (contract: notices of read-only paths go to stderr).
	Error   io.Writer
	Request NodeRequest
	Confirm func(string) (bool, error)
}

type nodePlan struct {
	node     state.Node
	key      *state.Key
	hostKeys []sshd.HostKey
	skipped  []string
}

func (request NodeRequest) plan(home string, document state.State) (nodePlan, error) {
	node := state.Node{Name: request.Name, Roles: request.Roles, LoginAccount: request.LoginAccount, Addresses: request.Addresses, Port: request.Port}
	plan := nodePlan{}
	if request.Key != "" {
		key, err := usableKey(document, request.Key)
		if err != nil {
			return plan, err
		}
		data, err := os.ReadFile(key.PublicPath)
		if err != nil {
			return plan, err
		}
		if _, err := roster.ParsePublicKey(string(data), false); err != nil {
			return plan, fmt.Errorf("managed key %q cannot be a client key: %w", key.Name, err)
		}
		plan.key = &key
		node.ClientKeyIDs = []string{key.ID}
	}
	if slices.Contains(request.Roles, roster.RoleRemote) {
		dir := request.SSHDir
		if dir == "" {
			dir = sshd.Directory()
		}
		config, required := request.SSHDConfig, true
		if config == "" {
			config, required = filepath.Join(dir, "sshd_config"), false
		}
		config, err := filepath.Abs(expandPath(home, config))
		if err != nil {
			return plan, err
		}
		keys, skipped, err := sshd.ReadHostKeys(config, dir, required)
		if err != nil {
			return plan, err
		}
		plan.hostKeys, plan.skipped = keys, skipped
		for _, key := range keys {
			node.HostKeys = append(node.HostKeys, key.Key.Text)
		}
		slices.Sort(node.HostKeys)
		node.SSHDConfig = config
	}
	if document.Node != nil {
		node.ID = document.Node.ID
	}
	plan.node = node
	return plan, nil
}

func (plan nodePlan) describe(output io.Writer, previous *state.Node) error {
	node := plan.node
	var text strings.Builder
	if previous == nil {
		fmt.Fprintf(&text, "Set this machine's node description (new node id)\n")
	} else {
		fmt.Fprintf(&text, "Replace this machine's node description %s (was %q, roles %s)\n", previous.ID, previous.Name, strings.Join(previous.Roles, ","))
	}
	fmt.Fprintf(&text, "Name: %s\nRoles: %s\n", node.Name, strings.Join(node.Roles, ","))
	if plan.key != nil {
		fmt.Fprintf(&text, "Client key: %q [%s] %s\n", plan.key.Name, plan.key.ID, plan.key.Fingerprint)
	}
	if slices.Contains(node.Roles, roster.RoleRemote) {
		port := node.Port
		if port == 0 {
			port = roster.DefaultPort
		}
		account := node.LoginAccount
		if account == "" {
			account = "(the fleet's common account)"
		}
		fmt.Fprintf(&text, "Addresses: %s\nPort: %d\nLogin account: %s\nsshd config: %s\n", strings.Join(node.Addresses, ", "), port, account, node.SSHDConfig)
		for _, key := range plan.hostKeys {
			fmt.Fprintf(&text, "Host key: %s %s (%s)\n", key.Key.Type, key.Key.Fingerprint, key.Path)
		}
		for _, path := range plan.skipped {
			fmt.Fprintf(&text, "Not found, skipped: %s\n", path)
		}
	}
	text.WriteString("Only state.json changes; no SSH file is written.\n")
	_, err := io.WriteString(output, text.String())
	return err
}

// SetNode creates or replaces this machine's node description after a
// terminal yes. With DryRun it prints the plan and never initializes storage.
func SetNode(options NodeOptions) error {
	if options.Home == "" {
		return fmt.Errorf("Home is required")
	}
	if options.Output == nil {
		options.Output = os.Stdout
	}
	if options.Error == nil {
		options.Error = os.Stderr
	}
	request := options.Request
	if err := request.Validate(); err != nil {
		return err
	}
	document, err := state.Load(options.Home)
	if err != nil {
		return err
	}
	if options.Request.DryRun {
		if err := migrationNotice(options.Error, document); err != nil {
			return err
		}
		plan, err := request.plan(options.Home, document)
		if err != nil {
			return err
		}
		if err := plan.describe(options.Output, document.Node); err != nil {
			return err
		}
		_, err = fmt.Fprintln(options.Output, "Dry run: no changes made.")
		return err
	}
	if _, err := request.plan(options.Home, document); err != nil {
		return err
	}
	if err := state.CheckWritable(options.Home); err != nil {
		return err
	}
	return state.WithLock(options.Home, func(store *state.LockedStore) error {
		document, err := store.Load()
		if err != nil {
			return err
		}
		if err := migrationNotice(options.Output, document); err != nil {
			return err
		}
		plan, err := request.plan(options.Home, document)
		if err != nil {
			return err
		}
		if err := plan.describe(options.Output, document.Node); err != nil {
			return err
		}
		if _, err := prompt(Options{Output: options.Output, Confirm: options.Confirm}, "Type yes to save this node description:"); err != nil {
			return err
		}
		node := plan.node
		if node.ID == "" {
			if node.ID, err = newID("node"); err != nil {
				return err
			}
		}
		node.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		document.Node = &node
		if err := store.Save(document); err != nil {
			return err
		}
		_, err = fmt.Fprintf(options.Output, "Node description saved as %s.\n", node.ID)
		return err
	})
}

// NodeView is the automation-facing `node show --json` document.
type NodeView struct {
	Node    *RosterNode `json:"node"`
	IsAdmin bool        `json:"is_admin"`
}

// RosterNode is a node in roster form (roster-format.md 1.2): client keys
// as public key text instead of managed key ids.
type RosterNode struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Roles        []string `json:"roles"`
	LoginAccount string   `json:"login_account,omitempty"`
	Addresses    []string `json:"addresses,omitempty"`
	Port         int      `json:"port,omitempty"`
	ClientKeys   []string `json:"client_keys,omitempty"`
	HostKeys     []string `json:"host_keys,omitempty"`
}

// ShowNode reads this machine's node description without initializing
// storage or migrating state. Review holds notes for stderr.
func ShowNode(home string) (NodeView, string, error) {
	if home == "" {
		return NodeView{}, "", fmt.Errorf("Home is required")
	}
	document, err := state.Load(home)
	if err != nil {
		return NodeView{}, "", err
	}
	review := ""
	if document.Migration != nil {
		review = document.Migration.Notice()
	}
	view := NodeView{IsAdmin: document.Fleet != nil && document.Fleet.RootKey != nil}
	if document.Node == nil {
		return view, review, nil
	}
	node := document.Node
	shown := RosterNode{ID: node.ID, Name: node.Name, Roles: node.Roles, LoginAccount: node.LoginAccount,
		Addresses: node.Addresses, Port: node.Port, HostKeys: node.HostKeys}
	for _, id := range node.ClientKeyIDs {
		text, err := PublicKey(home, id)
		if err != nil {
			return NodeView{}, "", fmt.Errorf("client key %s: %w", id, err)
		}
		shown.ClientKeys = append(shown.ClientKeys, text)
	}
	slices.Sort(shown.ClientKeys)
	view.Node = &shown
	return view, review, nil
}
