package app

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ssh-keyflow/internal/state"
)

func prompt(options Options, label string) (string, error) {
	if _, err := fmt.Fprintln(options.Output, label); err != nil {
		return "", err
	}
	if options.Confirm == nil {
		return "", fmt.Errorf("explicit confirmation is required")
	}
	approved, err := options.Confirm(label)
	if err != nil {
		options.report(Report{Kind: ReportConfirmation, Outcome: OutcomeUnknown})
		return "", err
	}
	if !approved {
		options.report(Report{Kind: ReportConfirmation, Outcome: OutcomeDeclined})
		return "", ErrCancelled
	}
	return "yes", nil
}

func fingerprint(data []byte) (string, error) {
	fields := strings.Fields(string(data))
	if len(fields) < 2 || strings.ContainsAny(strings.TrimSpace(string(data)), "\r\n\x00") {
		return "", fmt.Errorf("expected one OpenSSH public key")
	}
	blob, err := base64.StdEncoding.DecodeString(fields[1])
	if err != nil || len(blob) < 4 {
		return "", fmt.Errorf("invalid public key encoding")
	}
	length := int(binary.BigEndian.Uint32(blob[:4]))
	if length < 1 || length > len(blob)-4 || string(blob[4:4+length]) != fields[0] {
		return "", fmt.Errorf("public key algorithm does not match its data")
	}
	digest := sha256.Sum256(blob)
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(digest[:]), nil
}

func importLocked(ctx context.Context, options Options, store *state.LockedStore) error {
	document, err := store.Load()
	if err != nil {
		return err
	}
	if err := integrity(document); err != nil {
		return err
	}
	publicPath, err := filepath.Abs(expandPath(options.Home, options.Request.PublicPath))
	if err != nil {
		return err
	}
	privatePath := options.Request.PrivatePath
	if privatePath != "" {
		privatePath, err = filepath.Abs(expandPath(options.Home, privatePath))
		if err != nil {
			return err
		}
	}
	name := options.Request.Name
	data, err := os.ReadFile(publicPath)
	if err != nil {
		return err
	}
	identity, err := fingerprint(data)
	if err != nil {
		return err
	}
	if err := options.Runner.Run(ctx, Command{Name: "ssh-keygen", Args: []string{"-l", "-E", "sha256", "-f", publicPath}, Stdin: options.Input, Stdout: options.Output, Stderr: options.Output}); err != nil {
		return fmt.Errorf("validate public key: %w", err)
	}
	privateHash := ""
	if privatePath != "" {
		privateData, err := os.ReadFile(privatePath)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(privateData)
		privateHash = hex.EncodeToString(digest[:])
		var derived bytes.Buffer
		if err := options.Runner.Run(ctx, Command{Name: "ssh-keygen", Args: []string{"-y", "-f", privatePath}, Stdin: options.Input, Stdout: &derived, Stderr: options.Output, Interactive: true}); err != nil {
			return fmt.Errorf("derive public key; private key unavailable: %w", err)
		}
		derivedIdentity, err := fingerprint(derived.Bytes())
		if err != nil || derivedIdentity != identity {
			return fmt.Errorf("key pair fingerprint mismatch; no changes made")
		}
	}
	if _, err := fmt.Fprintf(options.Output, "Import %q\nPublic: %s\nPrivate: %s\nFingerprint: %s\nConnection: unverified\n", name, publicPath, privatePath, identity); err != nil {
		return err
	}
	answer, err := prompt(options, "Type yes to import; source files stay in place:")
	if err != nil {
		return err
	}
	if answer != "yes" {
		fmt.Fprintln(options.Output, "Cancelled; no changes made")
		return nil
	}
	err = func() error {
		if err := integrity(document); err != nil {
			return err
		}
		for _, existing := range document.Keys {
			if existing.Fingerprint == identity {
				return fmt.Errorf("already managed as %s [%s]", existing.Name, existing.ID)
			}
		}
		identifier := make([]byte, 16)
		if _, err := rand.Read(identifier); err != nil {
			return err
		}
		key := state.Key{ID: "key-" + hex.EncodeToString(identifier), Name: name, Fingerprint: identity, PublicPath: publicPath, PrivatePath: privatePath, PrivateSHA256: privateHash, ConnectionCheck: "unverified", Origin: "imported", ImportedAt: time.Now().UTC().Format(time.RFC3339Nano)}
		if err := integrity(state.State{Keys: []state.Key{key}}); err != nil {
			return err
		}
		document.Keys = append(document.Keys, key)
		options.target = ReportTarget{KeyID: key.ID}
		return options.saveReport(store, document, Report{Kind: ReportImport, Outcome: OutcomeApplied, Path: key.PublicPath})
	}()
	if err == nil {
		fmt.Fprintln(options.Output, "Imported", name)
	}
	return err
}

func integrity(document state.State) error {
	for _, key := range document.Keys {
		if pendingDeletion(document, key.ID) {
			continue
		}
		data, err := os.ReadFile(key.PublicPath)
		if err != nil {
			return fmt.Errorf("requires review: %s public key missing/unreadable: %w", key.ID, err)
		}
		identity, err := fingerprint(data)
		if err != nil || identity != key.Fingerprint {
			return fmt.Errorf("requires review: %s public key fingerprint mismatch", key.ID)
		}
		if key.PrivatePath != "" {
			data, err := os.ReadFile(key.PrivatePath)
			if err != nil {
				return fmt.Errorf("requires review: %s private key missing/unreadable: %w", key.ID, err)
			}
			digest := sha256.Sum256(data)
			if hex.EncodeToString(digest[:]) != key.PrivateSHA256 {
				return fmt.Errorf("requires review: %s private key file mismatch", key.ID)
			}
		}
	}
	return nil
}

func expandPath(home, path string) string {
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, `~\`) {
		return filepath.Join(home, path[2:])
	}
	return path
}
func pendingDeletion(document state.State, keyID string) bool {
	for _, deletion := range document.Deletions {
		if deletion.KeyID == keyID && deletion.Stage != "complete" {
			return true
		}
	}
	return false
}
