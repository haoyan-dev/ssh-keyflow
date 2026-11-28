package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"ssh-keyflow/internal/roster"
	"ssh-keyflow/internal/state"
)

// Minimum OpenSSH for fleet signing (roster-format 5).
const minimumMajor, minimumMinor = 8, 4

// rootComment labels the fleet root public key; it never reaches the roster.
const rootComment = "ssh-keyflow-fleet-root"

// signerPrincipal is the principal of the allowed_signers file (roster-format 2).
const signerPrincipal = "fleet-root"

// FleetOptions runs `fleet init` and `fleet sign`. Input is the terminal that
// native ssh-keygen reads the passphrase from.
type FleetOptions struct {
	Home   string
	Input  io.Reader
	Output io.Writer
	// Error receives the migration notice of a dry run (contract: notices of
	// read-only paths go to stderr).
	Error   io.Writer
	Runner  Runner
	Confirm func(string) (bool, error)
	// LoginAccount is the roster's common account (`fleet init` only).
	LoginAccount string
	DryRun       bool
}

func (options *FleetOptions) defaults() error {
	if options.Home == "" {
		return fmt.Errorf("Home is required")
	}
	if options.Input == nil {
		options.Input = os.Stdin
	}
	if options.Output == nil {
		options.Output = os.Stdout
	}
	if options.Error == nil {
		options.Error = os.Stderr
	}
	if options.Runner == nil {
		options.Runner = SystemRunner{}
	}
	return nil
}

// keygen is the one ssh-keygen binary a command uses for every call.
type keygen struct {
	path, version string
	runner        Runner
}

func (k keygen) String() string { return k.path + " (" + k.version + ")" }

func (k keygen) run(ctx context.Context, command Command) error {
	command.Name = k.path
	return k.runner.Run(ctx, command)
}

var opensshVersion = regexp.MustCompile(`OpenSSH(?:_for_Windows)?_(\d+)\.(\d+)[0-9A-Za-z.]*`)

// resolveKeygen finds ssh-keygen on PATH and reads its version from the ssh
// next to it, since ssh-keygen has no version flag. Below 8.4 it stops.
func resolveKeygen(ctx context.Context, runner Runner) (keygen, error) {
	found, err := runner.LookPath("ssh-keygen")
	if err != nil {
		return keygen{}, fmt.Errorf("ssh-keygen not found on PATH; fleet signing needs OpenSSH %d.%d or later: %w", minimumMajor, minimumMinor, err)
	}
	path, err := filepath.Abs(found)
	if err != nil {
		return keygen{}, err
	}
	ssh := filepath.Join(filepath.Dir(path), "ssh"+filepath.Ext(path))
	var output bytes.Buffer
	runErr := runner.Run(ctx, Command{Name: ssh, Args: []string{"-V"}, Stdin: strings.NewReader(""), Stdout: &output, Stderr: &output})
	match := opensshVersion.FindStringSubmatch(output.String())
	if runErr != nil || match == nil {
		return keygen{}, fmt.Errorf("cannot read the OpenSSH version of %s from %s -V; fleet signing needs OpenSSH %d.%d or later", path, ssh, minimumMajor, minimumMinor)
	}
	major, _ := strconv.Atoi(match[1])
	minor, _ := strconv.Atoi(match[2])
	k := keygen{path: path, version: match[0], runner: runner}
	if major < minimumMajor || (major == minimumMajor && minor < minimumMinor) {
		return keygen{}, fmt.Errorf("%s is older than OpenSSH %d.%d, which fleet signing needs; put a newer ssh-keygen first on PATH", k, minimumMajor, minimumMinor)
	}
	return k, nil
}

func fleetDirectory(home string) (string, error) {
	return filepath.Abs(filepath.Join(state.Directory(home), "fleet"))
}

// rootPaths are where `fleet init` keeps the root key pair.
func rootPaths(home string) (private, public string, err error) {
	directory, err := fleetDirectory(home)
	if err != nil {
		return "", "", err
	}
	private = filepath.Join(directory, "root_ed25519")
	return private, private + ".pub", nil
}

// fleetWorkspace creates a new temporary directory under fleet/ and refuses a
// linked fleet directory. The caller removes it.
func fleetWorkspace(home, pattern string) (string, error) {
	directory, err := fleetDirectory(home)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return "", err
	}
	if err := plainDirectory(directory); err != nil {
		return "", err
	}
	return os.MkdirTemp(directory, pattern)
}

func createFile(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	return errors.Join(writeErr, closeErr)
}

// rosterNode is this machine's node in roster form; a login account equal to
// the common account is omitted (roster-format 1.2).
func rosterNode(document state.State, common string) (roster.Node, error) {
	node := document.Node
	if node == nil {
		return roster.Node{}, fmt.Errorf("no node description; run node set first")
	}
	entry := roster.Node{ID: node.ID, Name: node.Name, Roles: node.Roles, LoginAccount: node.LoginAccount,
		Addresses: node.Addresses, Port: node.Port, HostKeys: node.HostKeys}
	if entry.LoginAccount == common {
		entry.LoginAccount = ""
	}
	for _, id := range node.ClientKeyIDs {
		text, err := publicKeyText(document, id)
		if err != nil {
			return roster.Node{}, fmt.Errorf("client key %s: %w", id, err)
		}
		entry.ClientKeys = append(entry.ClientKeys, text)
	}
	slices.Sort(entry.ClientKeys)
	return entry, nil
}

func describeRosterNode(text *strings.Builder, prefix string, node roster.Node) {
	fmt.Fprintf(text, "%s%s %s, roles %s\n", prefix, node.Name, node.ID, strings.Join(node.Roles, ","))
	if node.LoginAccount != "" {
		fmt.Fprintf(text, "%s  Login account: %s\n", prefix, node.LoginAccount)
	}
	if len(node.Addresses) > 0 {
		port := node.Port
		if port == 0 {
			port = roster.DefaultPort
		}
		fmt.Fprintf(text, "%s  Addresses: %s, port %d\n", prefix, strings.Join(node.Addresses, ", "), port)
	}
	for _, list := range []struct {
		label string
		keys  []string
		host  bool
	}{{"Client key", node.ClientKeys, false}, {"Host key", node.HostKeys, true}} {
		for _, line := range list.keys {
			if key, err := roster.ParsePublicKey(line, list.host); err == nil {
				fmt.Fprintf(text, "%s  %s: %s %s\n", prefix, list.label, key.Type, key.Fingerprint)
			} else {
				fmt.Fprintf(text, "%s  %s: unreadable (%v); compare with --json\n", prefix, list.label, err)
			}
		}
	}
}

// pinnedRoot returns the pinned root the admin's root key belongs to.
func pinnedRoot(fleet state.Fleet) (state.Root, error) {
	for _, root := range fleet.Roots {
		if fleet.RootKey != nil && root.Fingerprint == fleet.RootKey.Fingerprint {
			return root, nil
		}
	}
	return state.Root{}, fmt.Errorf("requires review: the fleet root key is not pinned")
}

// checkRootKey confirms the root key files still match the pinned key. The
// private key is only checked to be a regular file, never read.
func checkRootKey(fleet state.Fleet) error {
	key := fleet.RootKey
	data, err := os.ReadFile(key.PublicPath)
	if err != nil {
		return fmt.Errorf("requires review: fleet root public key unreadable: %w", err)
	}
	if identity, err := fingerprint(data); err != nil || identity != key.Fingerprint {
		return fmt.Errorf("requires review: %s does not match the pinned root %s", key.PublicPath, key.Fingerprint)
	}
	info, err := os.Lstat(key.PrivatePath)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("requires review: fleet root private key missing or not a regular file at %s", key.PrivatePath)
	}
	return nil
}

// sign signs the roster with the root key, verifies the signature against the
// pinned root, and saves it as the accepted roster. Native ssh-keygen asks for
// the passphrase on the inherited terminal.
func sign(ctx context.Context, options FleetOptions, k keygen, store *state.LockedStore, document *state.State, next roster.Roster) error {
	fleet := document.Fleet
	root, err := pinnedRoot(*fleet)
	if err != nil {
		return err
	}
	if err := checkRootKey(*fleet); err != nil {
		return err
	}
	next.CreatedAt = time.Now().UTC().Format(roster.TimeLayout)
	data, err := next.Marshal()
	if err != nil {
		return err
	}
	workspace, err := fleetWorkspace(options.Home, ".sign-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workspace)
	file := filepath.Join(workspace, "roster.json")
	if err := createFile(file, data); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(options.Output, "Enter the fleet root key passphrase in native ssh-keygen to sign roster version %d.\n", next.Version); err != nil {
		return err
	}
	if err := k.run(ctx, Command{Args: []string{"-Y", "sign", "-f", fleet.RootKey.PrivatePath, "-n", roster.Namespace, file},
		Stdin: options.Input, Stdout: options.Output, Stderr: options.Output, Interactive: true}); err != nil {
		return fmt.Errorf("%s could not sign roster version %d: %w; nothing was saved", k, next.Version, err)
	}
	written, err := os.ReadFile(file + ".sig")
	if err != nil {
		return fmt.Errorf("%s wrote no signature: %w; nothing was saved", k, err)
	}
	signature := strings.ReplaceAll(string(written), "\r\n", "\n")
	if !roster.ValidSignature(signature) {
		return fmt.Errorf("%s wrote an unreadable signature; nothing was saved", k)
	}
	allowed, checked := filepath.Join(workspace, "allowed_signers"), filepath.Join(workspace, "checked.sig")
	if err := createFile(allowed, []byte(signerPrincipal+` namespaces="`+roster.Namespace+`" `+root.PublicKey+"\n")); err != nil {
		return err
	}
	if err := createFile(checked, []byte(signature)); err != nil {
		return err
	}
	var verifyOutput bytes.Buffer
	if err := k.run(ctx, Command{Args: []string{"-Y", "verify", "-f", allowed, "-I", signerPrincipal, "-n", roster.Namespace, "-s", checked},
		Stdin: bytes.NewReader(data), Stdout: &verifyOutput, Stderr: &verifyOutput}); err != nil {
		return fmt.Errorf("%s did not verify the new signature against the pinned root %s: %w (%s); nothing was saved", k, root.Fingerprint, err, strings.TrimSpace(verifyOutput.String()))
	}
	digest := sha256.Sum256(data)
	fleet.Accepted = &state.Accepted{Version: next.Version, SHA256: hex.EncodeToString(digest[:]), Roster: string(data), Signature: signature,
		AcceptedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	if err := store.Save(*document); err != nil {
		return err
	}
	_, err = fmt.Fprintf(options.Output, "Signed and accepted roster version %d (%d nodes, SHA-256 %s).\n", next.Version, len(next.Nodes), fleet.Accepted.SHA256)
	return err
}

// migrationNotice prints the notice of an older state file, if any.
func migrationNotice(output io.Writer, document state.State) error {
	if document.Migration == nil {
		return nil
	}
	_, err := fmt.Fprintln(output, document.Migration.Notice())
	return err
}

// existingPath returns the first of paths that exists, or "".
func existingPath(paths ...string) (string, error) {
	for _, path := range paths {
		if _, err := os.Lstat(path); err == nil {
			return path, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}
	return "", nil
}

// fleetChange is a planned `fleet init` or `fleet sign`.
type fleetChange interface {
	describe(io.Writer) error
	label() string
	apply(context.Context, FleetOptions, *state.LockedStore, *state.State) error
}

// runFleetChange plans, describes and, after a terminal yes, applies a fleet
// change under the state lock. A dry run only plans and describes, and never
// initializes storage.
func runFleetChange(options FleetOptions, plan func(state.State) (fleetChange, error)) func(context.Context) error {
	return func(ctx context.Context) error {
		document, err := state.Load(options.Home)
		if err != nil {
			return err
		}
		if options.DryRun {
			if err := migrationNotice(options.Error, document); err != nil {
				return err
			}
			change, err := plan(document)
			if err != nil {
				return err
			}
			if err := change.describe(options.Output); err != nil {
				return err
			}
			_, err = fmt.Fprintln(options.Output, "Dry run: no changes made.")
			return err
		}
		if _, err := plan(document); err != nil {
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
			change, err := plan(document)
			if err != nil {
				return err
			}
			if err := change.describe(options.Output); err != nil {
				return err
			}
			if _, err := prompt(Options{Output: options.Output, Confirm: options.Confirm}, change.label()); err != nil {
				return err
			}
			return change.apply(ctx, options, store, &document)
		})
	}
}

// initPlan is what `fleet init` will do: create the root key, or only sign
// version 1 when an earlier init saved the key but did not sign.
type initPlan struct {
	resume          bool
	private, public string
	// rootFingerprint is the saved root key when resuming.
	rootFingerprint string
	version1        roster.Roster
	keygen          keygen
}

func (options FleetOptions) planInit(ctx context.Context, document state.State) (initPlan, error) {
	if document.Node == nil {
		return initPlan{}, fmt.Errorf("no node description; run node set first")
	}
	plan := initPlan{}
	switch fleet := document.Fleet; {
	case fleet == nil:
		private, public, err := rootPaths(options.Home)
		if err != nil {
			return initPlan{}, err
		}
		if path, err := existingPath(private, public); err != nil {
			return initPlan{}, err
		} else if path != "" {
			return initPlan{}, fmt.Errorf("requires review: %s exists but no fleet is in state; it is never reused or replaced", path)
		}
		plan.private, plan.public = private, public
	case fleet.Accepted != nil:
		return initPlan{}, fmt.Errorf("this node already accepted fleet %s; fleet init runs once", fleet.ID)
	case fleet.RootKey == nil:
		return initPlan{}, fmt.Errorf("this node already pins fleet %s without its root key; fleet init runs on the admin node only", fleet.ID)
	default:
		if err := checkRootKey(*fleet); err != nil {
			return initPlan{}, err
		}
		plan.resume, plan.private, plan.public, plan.rootFingerprint = true, fleet.RootKey.PrivatePath, fleet.RootKey.PublicPath, fleet.RootKey.Fingerprint
	}
	node, err := rosterNode(document, options.LoginAccount)
	if err != nil {
		return initPlan{}, err
	}
	fleetID := ""
	if document.Fleet != nil {
		fleetID = document.Fleet.ID
	} else if fleetID, err = newID("fleet"); err != nil {
		return initPlan{}, err
	}
	plan.version1 = roster.Roster{Format: roster.Format, Schema: roster.Schema, FleetID: fleetID, Version: 1,
		CreatedAt: time.Now().UTC().Format(roster.TimeLayout), LoginAccount: options.LoginAccount, Nodes: []roster.Node{node}}
	if err := plan.version1.Validate(); err != nil {
		return initPlan{}, fmt.Errorf("roster version 1: %w", err)
	}
	if plan.keygen, err = resolveKeygen(ctx, options.Runner); err != nil {
		return initPlan{}, err
	}
	return plan, nil
}

func (plan initPlan) label() string {
	if plan.resume {
		return "Type yes to sign roster version 1:"
	}
	return "Type yes to create the fleet root key:"
}

func (plan initPlan) apply(ctx context.Context, options FleetOptions, store *state.LockedStore, document *state.State) error {
	if !plan.resume {
		if err := createRoot(ctx, options, plan, store, document); err != nil {
			return err
		}
	}
	return sign(ctx, options, plan.keygen, store, document, plan.version1)
}

func (plan initPlan) describe(output io.Writer) error {
	var text strings.Builder
	if plan.resume {
		fmt.Fprintf(&text, "Finish fleet init: sign roster version 1 of fleet %s with the existing root key %s\n", plan.version1.FleetID, plan.rootFingerprint)
	} else {
		fmt.Fprintf(&text, "Create a fleet root key and sign roster version 1 (new fleet id)\n")
		fmt.Fprintf(&text, "Root key: %s (+ .pub); state keeps only its path and fingerprint\n", plan.private)
	}
	fmt.Fprintf(&text, "ssh-keygen: %s\nCommon login account: %s\nRoster version 1 holds this node:\n", plan.keygen, plan.version1.LoginAccount)
	describeRosterNode(&text, "  ", plan.version1.Nodes[0])
	if !plan.resume {
		text.WriteString("Enter a passphrase in native ssh-keygen; an empty passphrase is refused. ssh-keygen then asks for it again to sign.\n")
	}
	text.WriteString("No SSH file is written.\n")
	_, err := io.WriteString(output, text.String())
	return err
}

// CheckLoginAccount checks `fleet init --login-account`; failure is a usage error.
func CheckLoginAccount(account string) error {
	if !roster.ValidAccount(account) {
		return fmt.Errorf("%w: --login-account is required: letters, digits, '.', '_' and '-', at most 64 characters", ErrInvalidInput)
	}
	return nil
}

// InitFleet creates the passphrase-protected fleet root key with native
// ssh-keygen, pins it, and signs roster version 1 holding this node. The
// private key never enters state or output. Rerunning it after a failed
// signature signs version 1 with the saved key.
func InitFleet(ctx context.Context, options FleetOptions) error {
	if err := options.defaults(); err != nil {
		return err
	}
	if err := CheckLoginAccount(options.LoginAccount); err != nil {
		return err
	}
	return runFleetChange(options, func(document state.State) (fleetChange, error) {
		plan, err := options.planInit(ctx, document)
		return plan, err
	})(ctx)
}

// createRoot generates the root key in a new temporary directory, refuses an
// unprotected key, moves the pair into place and pins it in one save.
func createRoot(ctx context.Context, options FleetOptions, plan initPlan, store *state.LockedStore, document *state.State) error {
	k := plan.keygen
	workspace, err := fleetWorkspace(options.Home, ".root-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workspace)
	generated := filepath.Join(workspace, "root_ed25519")
	if err := k.run(ctx, Command{Args: []string{"-t", "ed25519", "-C", rootComment, "-f", generated},
		Stdin: options.Input, Stdout: options.Output, Stderr: options.Output, Interactive: true}); err != nil {
		return fmt.Errorf("%s did not create the fleet root key: %w; nothing was saved", k, err)
	}
	// With an empty passphrase this succeeds: the key is unprotected. Any other
	// failure than the wrong-passphrase error leaves protection unconfirmed.
	var probe bytes.Buffer
	if err := k.run(ctx, Command{Args: []string{"-y", "-P", "", "-f", generated}, Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: &probe}); err == nil {
		return fmt.Errorf("%s: an empty passphrase is refused: the fleet root key must be passphrase-protected; the key was discarded and nothing was saved", k)
	} else if !strings.Contains(probe.String(), "incorrect passphrase") {
		return fmt.Errorf("%s could not confirm the new fleet root key is passphrase-protected: %w (%s); the key was discarded and nothing was saved", k, err, strings.TrimSpace(probe.String()))
	}
	data, err := os.ReadFile(generated + ".pub")
	if err != nil {
		return fmt.Errorf("%s wrote no public key: %w; nothing was saved", k, err)
	}
	key, err := roster.ParsePublicKey(strings.TrimSpace(string(data)), false)
	if err != nil || key.Type != "ssh-ed25519" {
		return fmt.Errorf("requires review: %s did not write an Ed25519 public key; nothing was saved", k)
	}
	if info, err := os.Lstat(generated); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("%s wrote no private key; nothing was saved", k)
	}
	if path, err := existingPath(plan.private, plan.public); err != nil {
		return err
	} else if path != "" {
		return fmt.Errorf("requires review: %s appeared during confirmation; preserved it and saved nothing", path)
	}
	// A hard link never replaces an existing file, unlike a rename on Unix;
	// the temporary names go with the workspace.
	if err := os.Link(generated+".pub", plan.public); err != nil {
		return fmt.Errorf("requires review: could not place %s without replacing a file: %w; nothing was saved", plan.public, err)
	}
	if err := os.Link(generated, plan.private); err != nil {
		return fmt.Errorf("requires review: placed %s but not the private key: %w; nothing was saved", plan.public, err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	document.Fleet = &state.Fleet{ID: plan.version1.FleetID, Roots: []state.Root{{PublicKey: key.Text, Fingerprint: key.Fingerprint, PinnedAt: now}},
		RootKey: &state.RootKey{Fingerprint: key.Fingerprint, PublicPath: plan.public, PrivatePath: plan.private}}
	if err := store.Save(*document); err != nil {
		return fmt.Errorf("requires review: the root key is at %s but state was not saved: %w", plan.private, err)
	}
	_, err = fmt.Fprintf(options.Output, "Fleet %s: root key %s pinned. If signing fails, rerun fleet init to sign version 1.\n", plan.version1.FleetID, key.Fingerprint)
	return err
}

// signPlan is the next roster `fleet sign` signs: the accepted roster with
// this node's current description and version + 1.
type signPlan struct {
	next     roster.Roster
	node     roster.Node
	previous *roster.Node
	keygen   keygen
}

func (options FleetOptions) planSign(ctx context.Context, document state.State) (signPlan, error) {
	fleet := document.Fleet
	switch {
	case fleet == nil:
		return signPlan{}, fmt.Errorf("no fleet; run fleet init on the admin node")
	case fleet.RootKey == nil:
		return signPlan{}, fmt.Errorf("this node has no fleet root key; only the admin node signs")
	case fleet.Accepted == nil:
		return signPlan{}, fmt.Errorf("fleet init is unfinished; rerun fleet init --login-account <name> to sign version 1")
	}
	if err := checkRootKey(*fleet); err != nil {
		return signPlan{}, err
	}
	current, err := roster.Parse([]byte(fleet.Accepted.Roster))
	if err != nil {
		return signPlan{}, fmt.Errorf("accepted roster: %w", err)
	}
	node, err := rosterNode(document, current.LoginAccount)
	if err != nil {
		return signPlan{}, err
	}
	plan := signPlan{next: current, node: node}
	plan.next.Nodes = slices.Clone(current.Nodes)
	index, found := slices.BinarySearchFunc(plan.next.Nodes, node.ID, func(entry roster.Node, id string) int { return strings.Compare(entry.ID, id) })
	if found {
		previous := plan.next.Nodes[index]
		plan.previous = &previous
		plan.next.Nodes[index] = node
	} else {
		plan.next.Nodes = slices.Insert(plan.next.Nodes, index, node)
	}
	plan.next.Version++
	plan.next.CreatedAt = time.Now().UTC().Format(roster.TimeLayout)
	if err := plan.next.Validate(); err != nil {
		return signPlan{}, fmt.Errorf("roster version %d: %w", plan.next.Version, err)
	}
	if plan.keygen, err = resolveKeygen(ctx, options.Runner); err != nil {
		return signPlan{}, err
	}
	return plan, nil
}

func (plan signPlan) describe(output io.Writer) error {
	var text strings.Builder
	fmt.Fprintf(&text, "Sign roster version %d of fleet %s (%d nodes)\nssh-keygen: %s\n", plan.next.Version, plan.next.FleetID, len(plan.next.Nodes), plan.keygen)
	switch {
	case plan.previous == nil:
		text.WriteString("Adds this node:\n")
		describeRosterNode(&text, "  ", plan.node)
	case reflect.DeepEqual(*plan.previous, plan.node):
		text.WriteString("This node's entry is unchanged; signing refreshes created_at.\n")
	default:
		text.WriteString("Replaces this node's entry:\n")
		describeRosterNode(&text, "  was: ", *plan.previous)
		describeRosterNode(&text, "  now: ", plan.node)
	}
	text.WriteString("No SSH file is written.\n")
	_, err := io.WriteString(output, text.String())
	return err
}

func (plan signPlan) label() string {
	return fmt.Sprintf("Type yes to sign roster version %d:", plan.next.Version)
}

func (plan signPlan) apply(ctx context.Context, options FleetOptions, store *state.LockedStore, document *state.State) error {
	return sign(ctx, options, plan.keygen, store, document, plan.next)
}

// SignFleet signs the next roster version on the admin node with this node's
// current description, after a terminal yes. It refreshes created_at even
// when nothing changed, so the admin can renew a roster before --max-age.
func SignFleet(ctx context.Context, options FleetOptions) error {
	if err := options.defaults(); err != nil {
		return err
	}
	return runFleetChange(options, func(document state.State) (fleetChange, error) {
		plan, err := options.planSign(ctx, document)
		return plan, err
	})(ctx)
}

// FleetView is the automation-facing `fleet show --json` document
// (roster-format 4); Fleet is nil before any roster is accepted.
type FleetView struct {
	Fleet *FleetRecord `json:"fleet"`
}

type FleetRecord struct {
	Roots  []FleetRoot     `json:"roots"`
	SHA256 string          `json:"sha256"`
	Roster json.RawMessage `json:"roster"`
	// Parsed is Roster decoded, for the text view.
	Parsed roster.Roster `json:"-"`
	// IsAdmin reports whether this node holds the fleet root key.
	IsAdmin bool `json:"-"`
}

type FleetRoot struct {
	Fingerprint string `json:"fingerprint"`
}

// ShowFleet reads the accepted roster without initializing storage, probing
// commands or migrating state. Warnings are for stderr only.
func ShowFleet(home string, maxAge time.Duration, now time.Time) (FleetView, []string, error) {
	if home == "" {
		return FleetView{}, nil, fmt.Errorf("Home is required")
	}
	document, err := state.Load(home)
	if err != nil {
		return FleetView{}, nil, err
	}
	var warnings []string
	if document.Migration != nil {
		warnings = append(warnings, document.Migration.Notice())
	}
	fleet := document.Fleet
	if fleet == nil {
		return FleetView{}, warnings, nil
	}
	if fleet.Accepted == nil {
		if fleet.RootKey != nil {
			warnings = append(warnings, fmt.Sprintf("fleet init is unfinished: root key %s is pinned but no roster is signed; rerun fleet init --login-account <name>", fleet.RootKey.Fingerprint))
		}
		return FleetView{}, warnings, nil
	}
	parsed, err := roster.Parse([]byte(fleet.Accepted.Roster))
	if err != nil {
		return FleetView{}, nil, fmt.Errorf("accepted roster: %w", err)
	}
	record := &FleetRecord{Roots: []FleetRoot{}, SHA256: fleet.Accepted.SHA256, Roster: json.RawMessage(fleet.Accepted.Roster), Parsed: parsed, IsAdmin: fleet.RootKey != nil}
	for _, root := range fleet.Roots {
		record.Roots = append(record.Roots, FleetRoot{Fingerprint: root.Fingerprint})
	}
	if warning := roster.AgeWarning(parsed, maxAge, now); warning != "" {
		warnings = append(warnings, warning)
	}
	return FleetView{Fleet: record}, warnings, nil
}

// DescribeRoster is the text view of `fleet show`, with the roster's age at now.
func DescribeRoster(record FleetRecord, now time.Time) string {
	var text strings.Builder
	parsed := record.Parsed
	fmt.Fprintf(&text, "Fleet: %s\n", parsed.FleetID)
	for _, root := range record.Roots {
		fmt.Fprintf(&text, "Root key: %s\n", root.Fingerprint)
	}
	age := "age unknown"
	if created, err := time.Parse(roster.TimeLayout, parsed.CreatedAt); err == nil {
		age = fmt.Sprintf("%d days old", int(now.Sub(created).Hours()/24))
	}
	fmt.Fprintf(&text, "Admin node: %t\nRoster version: %d, created %s (%s)\nSHA-256: %s\nCommon login account: %s\nNodes: %d\n",
		record.IsAdmin, parsed.Version, parsed.CreatedAt, age, record.SHA256, parsed.LoginAccount, len(parsed.Nodes))
	for _, node := range parsed.Nodes {
		describeRosterNode(&text, "  ", node)
	}
	return text.String()
}
