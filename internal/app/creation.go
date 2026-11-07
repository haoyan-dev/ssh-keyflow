package app

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"ssh-keyflow/internal/state"
)

func newID(prefix string) (string, error) {
	identifier := make([]byte, 16)
	if _, err := rand.Read(identifier); err != nil {
		return "", err
	}
	return prefix + "-" + hex.EncodeToString(identifier), nil
}

func createKey(ctx context.Context, options Options, store *state.LockedStore, document *state.State, name, onboardingID string) (state.Key, error) {
	for _, creation := range document.Creations {
		if creation.Stage != "complete" {
			return state.Key{}, fmt.Errorf("unfinished creation %s at %s; resume this saved operation before allocating another pair", creation.ID, creation.PrivatePath)
		}
	}
	if strings.TrimSpace(name) == "" || strings.ContainsAny(name, "\x00\r\n") {
		return state.Key{}, fmt.Errorf("invalid management name")
	}
	keyID, err := newID("key")
	if err != nil {
		return state.Key{}, err
	}
	creationID, err := newID("creation")
	if err != nil {
		return state.Key{}, err
	}
	createdAt := time.Now().UTC()
	var safeName strings.Builder
	for _, character := range name {
		if safeName.Len() >= 48 {
			break
		}
		if unicode.IsLetter(character) || unicode.IsDigit(character) || character == '-' || character == '_' {
			safeName.WriteRune(character)
		} else {
			safeName.WriteByte('_')
		}
	}
	base := "id_ed25519_" + safeName.String() + "_" + createdAt.Format("20060102T150405Z")
	directory, err := filepath.Abs(filepath.Join(state.Directory(options.Home), "keys"))
	if err != nil {
		return state.Key{}, err
	}
	privatePath := ""
	for suffix := 0; ; suffix++ {
		filename := base
		if suffix > 0 {
			filename += fmt.Sprintf("_%d", suffix)
		}
		candidate := filepath.Join(directory, filename)
		occupied := false
		for _, path := range []string{candidate, candidate + ".pub"} {
			if _, err := os.Lstat(path); err == nil {
				occupied = true
			} else if !os.IsNotExist(err) {
				return state.Key{}, err
			}
		}
		for _, creation := range document.Creations {
			if creation.PrivatePath == candidate {
				occupied = true
			}
		}
		if !occupied {
			privatePath = candidate
			break
		}
	}
	fmt.Fprintf(options.Output, "Create Ed25519 %q\nPrivate: %s\nPublic: %s.pub\nEnter passphrase in native ssh-keygen; an empty passphrase is allowed.\n", name, privatePath, privatePath)
	answer, err := prompt(options, "Type yes to create:")
	if err != nil {
		return state.Key{}, err
	}
	if answer != "yes" {
		return state.Key{}, fmt.Errorf("cancelled; no changes made")
	}
	if err := integrity(*document); err != nil {
		return state.Key{}, err
	}
	for _, candidate := range []string{privatePath, privatePath + ".pub"} {
		if _, err := os.Lstat(candidate); err == nil {
			return state.Key{}, fmt.Errorf("requires review: creation path appeared during confirmation; preserved %s", candidate)
		} else if !os.IsNotExist(err) {
			return state.Key{}, err
		}
	}
	document.Creations = append(document.Creations, state.Creation{ID: creationID, KeyID: keyID, Name: name, PrivatePath: privatePath, PublicPath: privatePath + ".pub", CreatedAt: createdAt.Format(time.RFC3339Nano), Stage: "planned"})
	if onboardingID != "" {
		found := false
		for jobIndex := range document.Onboardings {
			job := &document.Onboardings[jobIndex]
			if job.ID == onboardingID && job.KeyID == "" {
				job.KeyID, job.CreationID, job.Stage = keyID, creationID, "key_pending"
				found = true
				break
			}
		}
		if !found {
			return state.Key{}, fmt.Errorf("requires review: creation onboarding identity missing")
		}
	}
	options.target = ReportTarget{KeyID: keyID, CreationID: creationID, OnboardingID: onboardingID}
	if onboardingID != "" {
		for _, job := range document.Onboardings {
			if job.ID == onboardingID {
				options.target.HostID = job.HostID
			}
		}
	}
	if err := options.saveReport(store, *document, Report{Kind: ReportCreation, Outcome: OutcomePending, Stage: "planned", Path: privatePath}); err != nil {
		return state.Key{}, err
	}
	if onboardingID != "" {
		options.report(Report{Kind: ReportOnboarding, Outcome: OutcomePending, Stage: "key_pending", Saved: true})
	}
	return finishCreation(ctx, options, store, document, len(document.Creations)-1)
}

func finishCreation(ctx context.Context, options Options, store *state.LockedStore, document *state.State, creationIndex int) (result state.Key, err error) {
	creation := &document.Creations[creationIndex]
	options.target = ReportTarget{KeyID: creation.KeyID, CreationID: creation.ID}
	for _, job := range document.Onboardings {
		if job.CreationID == creation.ID {
			options.target.HostID, options.target.OnboardingID = job.HostID, job.ID
		}
	}
	options.report(Report{Kind: ReportResolvedTarget, Outcome: OutcomeResolved})
	fact := Report{Kind: ReportCreation, Outcome: OutcomeRequiresReview, Path: creation.PrivatePath}
	defer func() {
		if err != nil {
			options.report(fact)
		}
	}()
	if err := integrity(*document); err != nil {
		return state.Key{}, err
	}
	if err := creationLocation(options.Home, *creation); err != nil {
		return state.Key{}, err
	}
	if err := observeCreation(creation); err != nil {
		return state.Key{}, err
	}
	creation.Stage = "pending"
	fact.Outcome = OutcomeUnknown
	if err := options.saveReport(store, *document, Report{Kind: ReportCreation, Outcome: OutcomePending, Stage: creation.Stage, Path: creation.PrivatePath}); err != nil {
		return state.Key{}, err
	}
	if creation.PrivateSHA256 == "" && creation.Fingerprint != "" {
		return state.Key{}, fmt.Errorf("requires review: public-only creation at %s; preserve it and restore the matching private file; no automatic deletion or generation", creation.PublicPath)
	}
	if creation.PrivateSHA256 == "" {
		generationErr := options.Runner.Run(ctx, Command{Name: "ssh-keygen", Args: []string{"-t", "ed25519", "-f", creation.PrivatePath}, Stdin: options.Input, Stdout: options.Output, Stderr: options.Output, Interactive: true})
		if err := observeCreation(creation); err != nil {
			return state.Key{}, err
		}
		if err := options.saveReport(store, *document, Report{Kind: ReportCreation, Outcome: OutcomePending, Stage: creation.Stage, Path: creation.PrivatePath}); err != nil {
			return state.Key{}, err
		}
		if generationErr != nil {
			return state.Key{}, fmt.Errorf("creation interrupted; resume the recorded path: %w", generationErr)
		}
		if creation.PrivateSHA256 == "" {
			return state.Key{}, fmt.Errorf("creation interrupted; no private file; resume the recorded path")
		}
	}
	var derived bytes.Buffer
	if err := options.Runner.Run(ctx, Command{Name: "ssh-keygen", Args: []string{"-y", "-f", creation.PrivatePath}, Stdin: options.Input, Stdout: &derived, Stderr: options.Output, Interactive: true}); err != nil {
		return state.Key{}, err
	}
	derivedIdentity, err := fingerprint(derived.Bytes())
	if err != nil || !strings.HasPrefix(derived.String(), "ssh-ed25519 ") {
		return state.Key{}, fmt.Errorf("requires review: generated key is not Ed25519")
	}
	if err := observeCreation(creation); err != nil {
		return state.Key{}, err
	}
	if creation.Fingerprint != "" && creation.Fingerprint != derivedIdentity {
		return state.Key{}, fmt.Errorf("requires review: generated key pair mismatch")
	}
	if creation.Fingerprint == "" {
		publicFile, err := os.OpenFile(creation.PublicPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return state.Key{}, fmt.Errorf("preserve existing public file; requires review: %w", err)
		}
		_, writeErr := publicFile.Write(append(bytes.TrimSpace(derived.Bytes()), '\n'))
		syncErr := publicFile.Sync()
		closeErr := publicFile.Close()
		if writeErr != nil {
			return state.Key{}, writeErr
		}
		if syncErr != nil {
			return state.Key{}, syncErr
		}
		if closeErr != nil {
			return state.Key{}, closeErr
		}
	}
	if err := observeCreation(creation); err != nil {
		return state.Key{}, err
	}
	creation.Stage = "complete"
	if err := integrity(*document); err != nil {
		return state.Key{}, err
	}
	key := state.Key{ID: creation.KeyID, Name: creation.Name, PublicPath: creation.PublicPath, PrivatePath: creation.PrivatePath, PrivateSHA256: creation.PrivateSHA256, Fingerprint: creation.Fingerprint, Origin: "created", CreatedAt: creation.CreatedAt, ConnectionCheck: "unverified"}
	document.Keys = append(document.Keys, key)
	if err := options.saveReport(store, *document, Report{Kind: ReportCreation, Outcome: OutcomeApplied, Stage: creation.Stage, Path: creation.PrivatePath}); err != nil {
		return state.Key{}, err
	}
	fmt.Fprintln(options.Output, "Created", key.Name, key.Fingerprint, "connection=unverified")
	return key, nil
}

func selectCreation(options Options, document state.State) (int, error) {
	var choices []int
	for creationIndex, creation := range document.Creations {
		if creation.Stage == "complete" {
			continue
		}
		choices = append(choices, creationIndex)
		fmt.Fprintf(options.Output, "%d: Resume %s [%s] %s stage=%s\n", len(choices), creation.Name, creation.ID, creation.PrivatePath, creation.Stage)
	}
	if len(choices) == 0 {
		return 0, fmt.Errorf("no unfinished creation")
	}
	answer, err := prompt(options, "Creation number (0 cancels):")
	if err != nil {
		return 0, err
	}
	selected, err := strconv.Atoi(answer)
	if err != nil || selected < 1 || selected > len(choices) {
		return 0, fmt.Errorf("no creation selected")
	}
	return choices[selected-1], nil
}

func creationLocation(home string, creation state.Creation) error {
	directory, err := filepath.Abs(filepath.Join(state.Directory(home), "keys"))
	if err != nil {
		return err
	}
	if filepath.Dir(creation.PrivatePath) != directory || creation.PublicPath != creation.PrivatePath+".pub" || !strings.HasPrefix(filepath.Base(creation.PrivatePath), "id_ed25519_") {
		return fmt.Errorf("requires review: creation path outside managed keys")
	}
	for current := directory; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("requires review: linked or invalid creation directory %s", current)
		}
		if filepath.Dir(current) == current {
			break
		}
	}
	return nil
}

func observeCreation(creation *state.Creation) error {
	for _, path := range []string{creation.PrivatePath, creation.PublicPath} {
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			if (path == creation.PrivatePath && creation.PrivateSHA256 != "") || (path == creation.PublicPath && creation.Fingerprint != "") {
				return fmt.Errorf("requires review: recorded creation file missing %s", path)
			}
			continue
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("requires review: non-regular creation file %s", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if len(data) == 0 {
			return fmt.Errorf("requires review: empty creation file %s", path)
		}
		if path == creation.PrivatePath {
			digest := sha256.Sum256(data)
			actual := hex.EncodeToString(digest[:])
			if creation.PrivateSHA256 != "" && creation.PrivateSHA256 != actual {
				return fmt.Errorf("requires review: creation private file mismatch")
			}
			creation.PrivateSHA256 = actual
		} else {
			identity, err := fingerprint(data)
			if err != nil || (creation.Fingerprint != "" && identity != creation.Fingerprint) {
				return fmt.Errorf("requires review: creation public fingerprint mismatch")
			}
			creation.Fingerprint = identity
		}
	}
	return nil
}
