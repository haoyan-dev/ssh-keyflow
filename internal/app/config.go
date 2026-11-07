package app

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"ssh-keyflow/internal/state"
)

func configDigest(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func plainConfig(path string) ([]byte, error) {
	for current := path; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("requires review: symlink/reparse configuration path %s", current)
		}
		if filepath.Dir(current) == current {
			break
		}
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("requires review: configuration is not a regular file")
	}
	return os.ReadFile(path)
}

func ownedConfig(options Options, document *state.State) ([]byte, int, error) {
	path := filepath.Join(state.Directory(options.Home), "config")
	data, err := plainConfig(path)
	if err != nil {
		return nil, -1, err
	}
	for index, record := range document.Configs {
		if record.Path != path {
			continue
		}
		digest := configDigest(data)
		if data != nil && (digest == record.SHA256 || digest == record.PendingSHA256) {
			return data, index, nil
		}
		if data == nil && record.SHA256 == "" {
			return nil, index, nil
		}
		return nil, -1, fmt.Errorf("requires review: managed configuration digest mismatch or missing file; preserved")
	}
	if data != nil {
		return nil, -1, fmt.Errorf("requires review: unowned dedicated config exists; preserved")
	}
	return nil, -1, nil
}

func configureRegistration(options Options, store *state.LockedStore, document *state.State, host state.Host, key state.Key) (err error) {
	options.target = pairTarget(document, host.ID, key.ID)
	path := filepath.Join(state.Directory(options.Home), "config")
	fact := Report{Kind: ReportAlias, Outcome: OutcomeRequiresReview, Path: path}
	aliasDone := false
	defer func() {
		if err != nil && !aliasDone {
			options.report(fact)
			options.report(Report{Kind: ReportInclude, Outcome: OutcomeSkipped, Path: filepath.Join(options.Home, ".ssh", "config")})
		}
	}()
	skipInclude := func() {
		options.report(Report{Kind: ReportInclude, Outcome: OutcomeSkipped, Path: filepath.Join(options.Home, ".ssh", "config")})
	}
	if options.Request != nil && !options.Request.Configure {
		fact.Outcome = OutcomeSkipped
		options.report(fact)
		skipInclude()
		return nil
	}
	answer, err := prompt(options, "Generate SSH alias configuration? Type yes to approve:")
	if err != nil || answer != "yes" {
		fact.Outcome = approvalOutcome(err)
		if err == nil {
			options.report(fact)
			skipInclude()
		}
		return err
	}
	previous, configIndex, err := ownedConfig(options, document)
	if err != nil {
		return err
	}
	if key.PrivatePath == "" {
		fact.Outcome = OutcomeSkipped
		options.report(fact)
		skipInclude()
		fmt.Fprintln(options.Output, "Public-only key: no usable IdentityFile; alias remains manual until a private key is explicitly imported.")
		return nil
	}
	ownedPath := ""
	if configIndex >= 0 {
		ownedPath = path
	}
	for _, candidate := range document.Hosts {
		if candidate.ID != host.ID && candidate.ConfigKeyID == "" {
			continue
		}
		if err := checkAliasExcept(options, candidate.Alias, ownedPath); err != nil {
			return err
		}
	}
	if err := integrity(*document); err != nil {
		return err
	}
	for index := range document.Hosts {
		if document.Hosts[index].ID == host.ID {
			document.Hosts[index].ConfigKeyID = key.ID
		}
	}
	data, err := managedConfiguration(*document)
	if err != nil {
		return err
	}
	fmt.Fprintf(options.Output, "Managed configuration preview %s:\n%s", path, data)
	if configIndex < 0 {
		document.Configs = append(document.Configs, state.Config{Path: path})
		configIndex = len(document.Configs) - 1
	}
	document.Configs[configIndex].PendingSHA256 = configDigest(data)
	fact.Outcome = OutcomeUnknown
	if err := options.saveReport(store, *document, Report{Kind: ReportAlias, Outcome: OutcomePending, Path: path}); err != nil {
		return err
	}
	if err := replaceConfig(path, previous, data); err != nil {
		return err
	}
	document.Configs[configIndex].SHA256 = configDigest(data)
	document.Configs[configIndex].PendingSHA256 = ""
	if err := options.saveReport(store, *document, Report{Kind: ReportAlias, Outcome: OutcomeApplied, Path: path}); err != nil {
		return err
	}
	aliasDone = true
	return includeConfig(options, store, document)
}

func managedConfiguration(document state.State) ([]byte, error) {
	var generated strings.Builder
	for _, destination := range document.Hosts {
		if destination.ConfigKeyID == "" {
			continue
		}
		var identity state.Key
		for _, candidate := range document.Keys {
			if candidate.ID == destination.ConfigKeyID {
				identity = candidate
			}
		}
		if identity.PrivatePath == "" {
			return nil, fmt.Errorf("requires review: configured private key unavailable")
		}
		for _, field := range []string{destination.Alias, destination.Hostname, destination.User, identity.PrivatePath} {
			if strings.ContainsAny(field, "\r\n\x00\"%$") {
				return nil, fmt.Errorf("requires review: unsupported SSH config expansion/quoting")
			}
		}
		fmt.Fprintf(&generated, "Host %s\n  HostName %s\n  User %s\n  Port %d\n  IdentityFile %s\n  IdentitiesOnly yes\n  IdentityAgent none\n  CertificateFile none\n\n", destination.Alias, destination.Hostname, strconv.Quote(destination.User), destination.Port, strconv.Quote(filepath.ToSlash(identity.PrivatePath)))
	}
	generated.WriteString("Host *\n")
	return []byte(generated.String()), nil
}

func removeKeyConfiguration(options Options, store *state.LockedStore, document *state.State, keyID, hostID string) (err error) {
	fact := Report{Kind: ReportAlias, Outcome: OutcomeRequiresReview, Operation: "remove", Path: filepath.Join(state.Directory(options.Home), "config")}
	defer func() {
		if err != nil {
			options.report(fact)
		}
	}()
	previous, configIndex, err := ownedConfig(options, document)
	if err != nil {
		return err
	}
	if configIndex < 0 {
		for _, host := range document.Hosts {
			if host.ConfigKeyID != "" {
				return fmt.Errorf("requires review: missing configuration ownership record")
			}
		}
		if len(document.Configs) != 0 {
			return fmt.Errorf("requires review: configuration ownership path mismatch")
		}
		options.report(Report{Kind: ReportAlias, Outcome: OutcomeSkipped, Operation: "remove", Path: fact.Path})
		return nil
	}
	for index := range document.Hosts {
		if document.Hosts[index].ConfigKeyID == keyID && (hostID == "" || document.Hosts[index].ID == hostID) {
			document.Hosts[index].ConfigKeyID = ""
		}
	}
	data, err := managedConfiguration(*document)
	if err != nil {
		return err
	}
	document.Configs[configIndex].PendingSHA256 = configDigest(data)
	fact.Outcome = OutcomeUnknown
	if err := options.saveReport(store, *document, Report{Kind: ReportAlias, Outcome: OutcomePending, Operation: "remove", Path: fact.Path}); err != nil {
		return err
	}
	if err := replaceConfig(document.Configs[configIndex].Path, previous, data); err != nil {
		return err
	}
	document.Configs[configIndex].SHA256, document.Configs[configIndex].PendingSHA256 = configDigest(data), ""
	return options.saveReport(store, *document, Report{Kind: ReportAlias, Outcome: OutcomeApplied, Operation: "remove", Path: fact.Path})
}

func replaceConfig(path string, before, after []byte) error {
	current, err := plainConfig(path)
	if err != nil {
		return err
	}
	if (current == nil) != (before == nil) || string(current) != string(before) {
		return fmt.Errorf("requires review: configuration changed since preview; preserved")
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".keyflow-config-*")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	if err := temporary.Chmod(0600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(after); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	current, err = plainConfig(path)
	if err != nil || (current == nil) != (before == nil) || string(current) != string(before) {
		return fmt.Errorf("requires review: configuration changed before replacement")
	}
	return os.Rename(temporary.Name(), path)
}

func includeConfig(options Options, store *state.LockedStore, document *state.State) (err error) {
	path := filepath.Join(options.Home, ".ssh", "config")
	fact := Report{Kind: ReportInclude, Outcome: OutcomeRequiresReview, Path: path}
	defer func() {
		if err != nil {
			options.report(fact)
		}
	}()
	original, err := plainConfig(path)
	if err != nil {
		return err
	}
	newline := "\n"
	if strings.Contains(string(original), "\r\n") {
		newline = "\r\n"
	}
	directive := "Include \"ssh-keyflow/config\"" + newline
	pendingIndex := -1
	for index, edit := range document.ConfigEdits {
		if edit.Path != path || edit.Complete {
			continue
		}
		if pendingIndex >= 0 {
			return fmt.Errorf("requires review: multiple unfinished Include edits")
		}
		var backup []byte
		var readErr error
		if edit.BackupPath != "" {
			backup, readErr = plainConfig(edit.BackupPath)
		}
		if readErr != nil || (backup != nil && configDigest(backup) != edit.BeforeSHA256) {
			return fmt.Errorf("requires review: Include backup changed or unreadable")
		}
		if original != nil && configDigest(original) == edit.AfterSHA256 {
			if edit.BackupPath != "" && backup == nil {
				return fmt.Errorf("requires review: interrupted Include backup missing")
			}
			document.ConfigEdits[index].Complete = true
			fact.Outcome = OutcomeUnknown
			if err := options.saveReport(store, *document, Report{Kind: ReportInclude, Outcome: OutcomeReconciled, Path: path}); err != nil {
				return err
			}
			fmt.Fprintln(options.Output, "Interrupted Include reconciled from exact config and backup digests; no files rewritten")
			return nil
		}
		if configDigest(original) != edit.BeforeSHA256 || (original == nil) != (edit.BackupPath == "") || configDigest(append([]byte(directive), original...)) != edit.AfterSHA256 {
			return fmt.Errorf("requires review: interrupted Include differs from saved intent")
		}
		pendingIndex = index
	}
	if strings.HasPrefix(string(original), directive) {
		options.report(Report{Kind: ReportInclude, Outcome: OutcomeReconciled, Path: path})
		fmt.Fprintln(options.Output, "Managed Include already active at top level; unchanged")
		return nil
	}
	included := false
	if err := inspectAliases(options, "", filepath.Join(state.Directory(options.Home), "config"), func() { included = true }); err != nil {
		return err
	}
	if included {
		options.report(fact)
		fmt.Fprintln(options.Output, "Existing Include may be scoped or ordered differently; manual review required; no duplicate added")
		return nil
	}
	fmt.Fprintf(options.Output, "Include preview %s (prepend before every Host/Match; generated aliases take precedence):\n+ %sManually add this line at the beginning of %s if declined.\n", path, directive, path)
	answer, err := prompt(options, "Type yes to approve changing this existing/unowned config after an exact backup; any other answer keeps manual setup:")
	if err != nil || answer != "yes" {
		fact.Outcome = approvalOutcome(err)
		if err == nil {
			options.report(fact)
		}
		return err
	}
	if _, _, err := ownedConfig(options, document); err != nil {
		return err
	}
	for _, host := range document.Hosts {
		if host.ConfigKeyID == "" {
			continue
		}
		if err := checkAliasExcept(options, host.Alias, filepath.Join(state.Directory(options.Home), "config")); err != nil {
			return err
		}
	}
	current, err := plainConfig(path)
	if err != nil || (current == nil) != (original == nil) || string(current) != string(original) {
		return fmt.Errorf("requires review: Include target changed since preview")
	}
	backupPath := ""
	if pendingIndex < 0 {
		identifier, err := newID("include")
		if err != nil {
			return err
		}
		if original != nil {
			backupPath = path + ".ssh-keyflow-backup-" + identifier
		}
		document.ConfigEdits = append(document.ConfigEdits, state.ConfigEdit{Path: path, BeforeSHA256: configDigest(original), AfterSHA256: configDigest(append([]byte(directive), original...)), BackupPath: backupPath})
		pendingIndex = len(document.ConfigEdits) - 1
	} else {
		backupPath = document.ConfigEdits[pendingIndex].BackupPath
	}
	fact.Outcome = OutcomeUnknown
	if err := options.saveReport(store, *document, Report{Kind: ReportInclude, Outcome: OutcomePending, Path: path}); err != nil {
		return err
	}
	backupBytes := []byte(nil)
	if backupPath != "" {
		backupBytes, err = plainConfig(backupPath)
		if err != nil || (backupBytes != nil && string(backupBytes) != string(original)) {
			return fmt.Errorf("requires review: Include backup changed")
		}
	}
	if original != nil && backupBytes == nil {
		backup, err := os.OpenFile(backupPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		_, writeErr := backup.Write(original)
		syncErr := backup.Sync()
		closeErr := backup.Close()
		if writeErr != nil {
			return writeErr
		}
		if syncErr != nil {
			return syncErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	if err := replaceConfig(path, original, append([]byte(directive), original...)); err != nil {
		return err
	}
	document.ConfigEdits[pendingIndex].Complete = true
	if err := options.saveReport(store, *document, Report{Kind: ReportInclude, Outcome: OutcomeApplied, Path: path}); err != nil {
		return err
	}
	fmt.Fprintln(options.Output, "Include saved; original bytes backed up when present")
	return nil
}
