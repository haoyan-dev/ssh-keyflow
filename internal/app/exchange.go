package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"

	"ssh-keyflow/internal/roster"
	"ssh-keyflow/internal/state"
)

// Files of an exported roster (roster-format 1.1) and of a node description.
const (
	rosterFile    = "roster.json"
	signatureFile = "roster.json.sig"
	rootFile      = "fleet-root.pub"
	nodeFile      = "node.json"
	// maxRootFileBytes bounds fleet-root.pub, one public key line.
	maxRootFileBytes = 16 << 10
)

// ExportOptions runs `node export` and `fleet export`, which change no state.
type ExportOptions struct {
	Home string
	// Out is the output directory; it must be absent or empty.
	Out    string
	Output io.Writer
	// Error receives the migration notice (contract: notices of read-only
	// paths go to stderr).
	Error io.Writer
}

// load fills defaults and reads state without initializing storage, printing
// any migration notice to Error.
func (options *ExportOptions) load() (state.State, error) {
	if options.Home == "" {
		return state.State{}, fmt.Errorf("Home is required")
	}
	if options.Out == "" {
		return state.State{}, fmt.Errorf("%w: --out is required", ErrInvalidInput)
	}
	if options.Output == nil {
		options.Output = os.Stdout
	}
	if options.Error == nil {
		options.Error = os.Stderr
	}
	document, err := state.Load(options.Home)
	if err != nil {
		return state.State{}, err
	}
	return document, migrationNotice(options.Error, document)
}

type exportFile struct {
	name string
	data []byte
}

// writeExport creates out (absent or empty) and each file exclusively. On a
// failure it removes the files it wrote.
func writeExport(home, out string, files []exportFile) (string, error) {
	directory, err := filepath.Abs(expandPath(home, out))
	if err != nil {
		return "", err
	}
	switch info, err := os.Lstat(directory); {
	case os.IsNotExist(err):
		if err := os.MkdirAll(directory, 0700); err != nil {
			return "", err
		}
	case err != nil:
		return "", err
	case !info.IsDir():
		return "", fmt.Errorf("--out %s is a link or not a directory; nothing was written", directory)
	default:
		entries, err := os.ReadDir(directory)
		if err != nil {
			return "", err
		}
		if len(entries) > 0 {
			return "", fmt.Errorf("--out %s must be absent or empty; nothing was written", directory)
		}
	}
	var written []string
	for _, file := range files {
		path := filepath.Join(directory, file.name)
		if err := createFile(path, file.data, 0644); err != nil {
			if !errors.Is(err, fs.ErrExist) {
				written = append(written, path)
			}
			var cleanup []error
			for _, done := range written {
				cleanup = append(cleanup, os.Remove(done))
			}
			if removeErr := errors.Join(cleanup...); removeErr != nil {
				return "", fmt.Errorf("could not write %s: %w; could not remove every file written so far: %w", path, err, removeErr)
			}
			return "", fmt.Errorf("could not write %s: %w; removed the files written so far", path, err)
		}
		written = append(written, path)
	}
	return directory, nil
}

// ExportNode writes this machine's node description file, node.json, for the
// admin node to add with `fleet add`.
func ExportNode(options ExportOptions) error {
	document, err := options.load()
	if err != nil {
		return err
	}
	// The common account is the admin node's; `fleet add` drops an equal override.
	node, err := rosterNode(document, "")
	if err != nil {
		return err
	}
	data, err := roster.NodeFile{Format: roster.NodeFormat, Schema: roster.NodeSchema, Node: node}.Marshal()
	if err != nil {
		return fmt.Errorf("node description: %w", err)
	}
	directory, err := writeExport(options.Home, options.Out, []exportFile{{nodeFile, data}})
	if err != nil {
		return err
	}
	var text strings.Builder
	fmt.Fprintf(&text, "Wrote %s for this node:\n", filepath.Join(directory, nodeFile))
	describeRosterNode(&text, "  ", node)
	fmt.Fprintf(&text, "On the admin node, run fleet add <dir> with a copy of this directory and compare these fingerprints.\n")
	_, err = io.WriteString(options.Output, text.String())
	return err
}

// ExportFleet writes the accepted roster, its signature and the fleet root
// public key (roster-format 1.1). It needs no passphrase.
func ExportFleet(options ExportOptions) error {
	document, err := options.load()
	if err != nil {
		return err
	}
	fleet := document.Fleet
	if fleet == nil || fleet.Accepted == nil {
		return fmt.Errorf("no accepted roster to export; run fleet init on the admin node or fleet accept")
	}
	// Phase 1 pins exactly one root, which signed every accepted roster.
	root := fleet.Roots[0]
	directory, err := writeExport(options.Home, options.Out, []exportFile{
		{rosterFile, []byte(fleet.Accepted.Roster)},
		{signatureFile, []byte(fleet.Accepted.Signature)},
		{rootFile, []byte(root.PublicKey + "\n")},
	})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(options.Output, "Wrote roster version %d of fleet %s (SHA-256 %s, root %s) to %s.\nOn another node, run fleet accept <dir> with a copy of this directory.\n",
		fleet.Accepted.Version, fleet.ID, fleet.Accepted.SHA256, root.Fingerprint, directory)
	return err
}

// CheckInputDirectory checks a `<dir>` argument before anything else runs; a
// missing or unreadable directory is a usage error (roster-format 2 rule 1).
func CheckInputDirectory(home, dir string) error {
	_, err := inputDirectory(home, dir)
	return err
}

func inputDirectory(home, dir string) (string, error) {
	path, err := filepath.Abs(expandPath(home, dir))
	if err == nil {
		var info os.FileInfo
		if info, err = os.Stat(path); err == nil && !info.IsDir() {
			err = fmt.Errorf("not a directory")
		}
	}
	if err == nil {
		_, err = os.ReadDir(path)
	}
	if err != nil {
		return "", fmt.Errorf("%w: %s is not a readable directory: %w", ErrInvalidInput, dir, err)
	}
	return path, nil
}

// readRegular reads a regular file of at most limit bytes.
func readRegular(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", path)
	}
	if info.Size() > limit {
		return nil, fmt.Errorf("%s exceeds %d bytes", path, limit)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err == nil && int64(len(data)) > limit {
		err = fmt.Errorf("%s exceeds %d bytes", path, limit)
	}
	return data, err
}

// AddNode adds or replaces the node in `<dir>/node.json` on the admin node
// and signs the next roster version, after a terminal yes.
func AddNode(ctx context.Context, options FleetOptions) error {
	return runDirectoryChange(ctx, options, func(document state.State, directory string) (fleetChange, error) {
		plan, err := options.planAdd(ctx, document, directory)
		return plan, err
	})
}

// runDirectoryChange runs a fleet change that reads `<dir>`, after checking it.
func runDirectoryChange(ctx context.Context, options FleetOptions, plan func(state.State, string) (fleetChange, error)) error {
	if err := options.defaults(); err != nil {
		return err
	}
	directory, err := inputDirectory(options.Home, options.Dir)
	if err != nil {
		return err
	}
	return runFleetChange(options, func(document state.State) (fleetChange, error) {
		return plan(document, directory)
	})(ctx)
}

func (options FleetOptions) planAdd(ctx context.Context, document state.State, directory string) (signPlan, error) {
	current, err := adminRoster(document)
	if err != nil {
		return signPlan{}, err
	}
	data, err := readRegular(filepath.Join(directory, nodeFile), roster.MaxBytes)
	if err != nil {
		return signPlan{}, fmt.Errorf("%s: %w", nodeFile, err)
	}
	file, err := roster.ParseNodeFile(data)
	if err != nil {
		return signPlan{}, fmt.Errorf("%s: %w", nodeFile, err)
	}
	node := file.Node
	if node.LoginAccount == current.LoginAccount {
		node.LoginAccount = ""
	}
	if document.Node != nil && node.ID == document.Node.ID {
		return signPlan{}, fmt.Errorf("%s describes this admin node; sign its own changes with fleet sign", nodeFile)
	}
	for _, entry := range current.Nodes {
		if entry.Name == node.Name && entry.ID != node.ID {
			return signPlan{}, fmt.Errorf("node name %q is already used by %s; rename one node with node set", node.Name, entry.ID)
		}
	}
	return options.planNext(ctx, current, node, true)
}

// rosterFiles are the three files of roster-format 1.1.
type rosterFiles struct {
	roster, signature, root []byte
}

// readRosterFiles requires exactly the three files, each regular.
func readRosterFiles(directory string) (rosterFiles, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return rosterFiles{}, err
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	want := []string{rootFile, rosterFile, signatureFile}
	if !slices.Equal(names, want) {
		return rosterFiles{}, fmt.Errorf("%s must hold exactly %s; found %s", directory, strings.Join(want, ", "), strings.Join(names, ", "))
	}
	var files rosterFiles
	for _, file := range []struct {
		name   string
		limit  int64
		target *[]byte
	}{{rosterFile, roster.MaxBytes, &files.roster}, {signatureFile, roster.MaxSignatureBytes, &files.signature}, {rootFile, maxRootFileBytes, &files.root}} {
		if *file.target, err = readRegular(filepath.Join(directory, file.name), file.limit); err != nil {
			return rosterFiles{}, err
		}
	}
	return files, nil
}

// parseRootFile reads fleet-root.pub: one line `ssh-ed25519 <base64>`, no comment.
func parseRootFile(data []byte) (roster.PublicKey, error) {
	line := strings.TrimSuffix(strings.TrimSuffix(string(data), "\n"), "\r")
	key, err := roster.ParsePublicKey(line, false)
	if err != nil || key.Text != line || key.Type != "ssh-ed25519" {
		return roster.PublicKey{}, fmt.Errorf("%s must be one line `ssh-ed25519 <base64>` without a comment", rootFile)
	}
	return key, nil
}

// acceptPlan is a verified roster that `fleet accept` will save.
type acceptPlan struct {
	data      []byte
	signature string
	received  roster.Roster
	root      roster.PublicKey
	// firstPin is set when this node pins no fleet yet.
	firstPin bool
	// previous is the accepted roster, if any.
	previous *roster.Roster
	// same is set when the roster is the accepted one: nothing to do.
	same    bool
	warning string
	// stderr receives the age warning (roster-format 2 rule 6).
	stderr io.Writer
	keygen keygen
}

func (options FleetOptions) planAccept(ctx context.Context, document state.State, directory string) (acceptPlan, error) {
	files, err := readRosterFiles(directory)
	if err != nil {
		return acceptPlan{}, err
	}
	root, err := parseRootFile(files.root)
	if err != nil {
		return acceptPlan{}, err
	}
	fleet := document.Fleet
	plan := acceptPlan{data: files.roster, root: root, firstPin: fleet == nil, stderr: options.Error}
	if fleet != nil && fleet.RootKey != nil && fleet.Accepted == nil {
		return acceptPlan{}, fmt.Errorf("fleet init is unfinished on this admin node; rerun fleet init --login-account <name> to sign version 1 instead of accepting a roster")
	}
	if fleet != nil && !slices.ContainsFunc(fleet.Roots, func(pin state.Root) bool { return pin.PublicKey == root.Text }) {
		return acceptPlan{}, fmt.Errorf("the roster is signed for root %s, which is not a pinned root of fleet %s; rejected", root.Fingerprint, fleet.ID)
	}
	plan.signature = strings.ReplaceAll(string(files.signature), "\r\n", "\n")
	if !roster.ValidSignature(plan.signature) {
		return acceptPlan{}, fmt.Errorf("%s is not an SSH signature; rejected", signatureFile)
	}
	if plan.keygen, err = resolveKeygen(ctx, options.Runner); err != nil {
		return acceptPlan{}, err
	}
	// Only public data goes here, so the system temporary directory serves
	// and a dry run initializes no storage.
	workspace, err := os.MkdirTemp("", "ssh-keyflow-accept-")
	if err != nil {
		return acceptPlan{}, err
	}
	defer os.RemoveAll(workspace)
	if output, err := plan.keygen.verify(ctx, workspace, root.Text, plan.data, plan.signature); err != nil {
		if plan.keygen.signedByKey(ctx, workspace, root.Text, plan.signature) {
			return acceptPlan{}, fmt.Errorf("%s: %s does not match its signature by root %s (modified or damaged): %w (%s); rejected", plan.keygen, rosterFile, root.Fingerprint, err, output)
		}
		return acceptPlan{}, fmt.Errorf("%s: %s is not a valid signature by root %s: %w (%s); rejected", plan.keygen, signatureFile, root.Fingerprint, err, output)
	}
	if plan.received, err = roster.Parse(plan.data); err != nil {
		return acceptPlan{}, fmt.Errorf("%s: %w; rejected", rosterFile, err)
	}
	if fleet != nil && plan.received.FleetID != fleet.ID {
		return acceptPlan{}, fmt.Errorf("the roster is for fleet %s, but this node pins fleet %s; rejected", plan.received.FleetID, fleet.ID)
	}
	if fleet != nil && fleet.Accepted != nil {
		accepted := fleet.Accepted
		switch {
		case plan.received.Version < accepted.Version:
			return acceptPlan{}, fmt.Errorf("roster version %d is older than the accepted version %d (rollback); rejected", plan.received.Version, accepted.Version)
		case plan.received.Version == accepted.Version && rosterDigest(plan.data) != accepted.SHA256:
			return acceptPlan{}, fmt.Errorf("roster version %d differs from the accepted roster of the same version (SHA-256 %s); rejected", accepted.Version, accepted.SHA256)
		}
		plan.same = plan.received.Version == accepted.Version
		previous, err := roster.Parse([]byte(accepted.Roster))
		if err != nil {
			return acceptPlan{}, fmt.Errorf("accepted roster: %w", err)
		}
		plan.previous = &previous
	}
	plan.warning = roster.AgeWarning(plan.received, options.MaxAge, time.Now())
	return plan, nil
}

func (plan acceptPlan) settled() bool { return plan.same }

func (plan acceptPlan) describe(output io.Writer) error {
	if plan.warning != "" {
		if _, err := fmt.Fprintln(plan.stderr, plan.warning); err != nil {
			return err
		}
	}
	received := plan.received
	var text strings.Builder
	if plan.same {
		fmt.Fprintf(&text, "Roster version %d of fleet %s is already accepted; nothing to do.\n", received.Version, received.FleetID)
		_, err := io.WriteString(output, text.String())
		return err
	}
	fmt.Fprintf(&text, "Accept roster version %d of fleet %s, created %s (%d nodes)\nssh-keygen: %s\n", received.Version, received.FleetID, received.CreatedAt, len(received.Nodes), plan.keygen)
	if plan.firstPin {
		fmt.Fprintf(&text, "First acceptance: this pins fleet root key %s for fleet %s.\n", plan.root.Fingerprint, received.FleetID)
		text.WriteString("Compare the root fingerprint and fleet id with fleet show on the admin node before typing yes.\n")
	} else {
		fmt.Fprintf(&text, "Signature verified against pinned root %s.\n", plan.root.Fingerprint)
	}
	if plan.previous == nil {
		fmt.Fprintf(&text, "Common login account: %s\n", received.LoginAccount)
		for _, node := range received.Nodes {
			describeRosterNode(&text, "  ", node)
		}
	} else {
		describeChanges(&text, *plan.previous, received)
	}
	text.WriteString("Only state.json changes; no SSH file is written.\n")
	_, err := io.WriteString(output, text.String())
	return err
}

// describeChanges lists how next differs from the accepted roster.
func describeChanges(text *strings.Builder, previous, next roster.Roster) {
	fmt.Fprintf(text, "Changes since accepted version %d:\n", previous.Version)
	before := map[string]roster.Node{}
	for _, node := range previous.Nodes {
		before[node.ID] = node
	}
	changed := false
	if previous.LoginAccount != next.LoginAccount {
		fmt.Fprintf(text, "  Common login account: %s -> %s\n", previous.LoginAccount, next.LoginAccount)
		changed = true
	}
	for _, node := range next.Nodes {
		old, found := before[node.ID]
		delete(before, node.ID)
		switch {
		case !found:
			describeRosterNode(text, "  added: ", node)
		case !reflect.DeepEqual(old, node):
			describeRosterNode(text, "  was: ", old)
			describeRosterNode(text, "  now: ", node)
		default:
			continue
		}
		changed = true
	}
	for _, node := range previous.Nodes {
		if _, removed := before[node.ID]; removed {
			fmt.Fprintf(text, "  removed: %s %s\n", node.Name, node.ID)
			changed = true
		}
	}
	if !changed {
		text.WriteString("  none; the roster was re-signed\n")
	}
}

func (plan acceptPlan) label() string {
	if plan.firstPin {
		return fmt.Sprintf("Type yes to pin root %s and accept roster version %d:", plan.root.Fingerprint, plan.received.Version)
	}
	return fmt.Sprintf("Type yes to accept roster version %d:", plan.received.Version)
}

func (plan acceptPlan) apply(_ context.Context, options FleetOptions, store *state.LockedStore, document *state.State) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if plan.firstPin {
		document.Fleet = &state.Fleet{ID: plan.received.FleetID, Roots: []state.Root{{PublicKey: plan.root.Text, Fingerprint: plan.root.Fingerprint, PinnedAt: now}}}
	}
	document.Fleet.Accepted = &state.Accepted{Version: plan.received.Version, SHA256: rosterDigest(plan.data), Roster: string(plan.data),
		Signature: plan.signature, AcceptedAt: now}
	if err := store.Save(*document); err != nil {
		return err
	}
	pinned := ""
	if plan.firstPin {
		pinned = fmt.Sprintf(" Pinned fleet root %s.", plan.root.Fingerprint)
	}
	_, err := fmt.Fprintf(options.Output, "Accepted roster version %d (%d nodes, SHA-256 %s).%s\n", plan.received.Version, len(plan.received.Nodes), document.Fleet.Accepted.SHA256, pinned)
	return err
}

// AcceptFleet verifies the roster in `<dir>` against the pinned fleet root
// keys (or, on first acceptance, the root the user confirms) and saves it
// after a terminal yes. Any failure changes nothing.
func AcceptFleet(ctx context.Context, options FleetOptions) error {
	if options.MaxAge <= 0 {
		return fmt.Errorf("MaxAge is required")
	}
	return runDirectoryChange(ctx, options, func(document state.State, directory string) (fleetChange, error) {
		plan, err := options.planAccept(ctx, document, directory)
		return plan, err
	})
}
