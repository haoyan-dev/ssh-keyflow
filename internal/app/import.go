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
	fmt.Fprintln(options.Output, label)
	return readLine(options.Input)
}

func fingerprint(data []byte) (string, error) {
	fields := strings.Fields(string(data))
	if len(fields) < 2 || strings.Contains(strings.TrimSpace(string(data)), "\n") {
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

func importKey(ctx context.Context, options Options) error {
	mode, err := prompt(options, "i: Import existing key; c: Create Ed25519 key; r: Resume creation; 0: Back")
	if err != nil {
		return err
	}
	if mode == "r" {
		return state.WithLock(options.Home, func(store *state.LockedStore) error {
			document, err := store.Load()
			if err != nil {
				return err
			}
			if err := integrity(document); err != nil {
				return err
			}
			creationIndex, err := selectCreation(options, document)
			if err != nil {
				return err
			}
			answer, err := prompt(options, "Type yes to resume this recorded path; existing files are preserved:")
			if err != nil {
				return err
			}
			if answer != "yes" {
				return nil
			}
			_, err = finishCreation(ctx, options, store, &document, creationIndex)
			return err
		})
	}
	if mode == "c" {
		return state.WithLock(options.Home, func(store *state.LockedStore) error {
			document, err := store.Load()
			if err != nil {
				return err
			}
			if err := integrity(document); err != nil {
				return err
			}
			name, err := prompt(options, "Management name:")
			if err != nil {
				return err
			}
			_, err = createKey(ctx, options, store, &document, name, "")
			return err
		})
	}
	if mode != "i" {
		fmt.Fprintln(options.Output, "No changes made")
		return nil
	}
	return state.WithLock(options.Home, func(store *state.LockedStore) error { return importLocked(ctx, options, store) })
}

func importLocked(ctx context.Context, options Options, store *state.LockedStore) error {
	document, err := store.Load()
	if err != nil {
		return err
	}
	if err := integrity(document); err != nil {
		return err
	}
	selected, err := prompt(options, "Existing public or private key path:")
	if err != nil {
		return err
	}
	selected, err = filepath.Abs(expandPath(options.Home, selected))
	if err != nil {
		return err
	}
	selectedData, err := os.ReadFile(selected)
	if err != nil {
		return err
	}
	publicPath, privatePath := selected, ""
	if strings.Contains(string(selectedData), "PRIVATE KEY-----") {
		privatePath = selected
		publicPath, err = prompt(options, "Public key path (blank uses selected private path + .pub):")
		if publicPath == "" {
			publicPath = selected + ".pub"
		}
	} else {
		privatePath, err = prompt(options, "Private key path (blank for public-only import):")
	}
	if err != nil {
		return err
	}
	publicPath, err = filepath.Abs(expandPath(options.Home, publicPath))
	if err != nil {
		return err
	}
	if privatePath != "" {
		privatePath, err = filepath.Abs(expandPath(options.Home, privatePath))
		if err != nil {
			return err
		}
	}
	name, err := prompt(options, "Management name:")
	if err != nil {
		return err
	}
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
	fmt.Fprintf(options.Output, "Import %q\nPublic: %s\nPrivate: %s\nFingerprint: %s\nConnection: unverified\n", name, publicPath, privatePath, identity)
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
		return store.Save(document)
	}()
	if err == nil {
		fmt.Fprintln(options.Output, "Imported", name)
	}
	return err
}

func integrity(document state.State) error {
	for _, key := range document.Keys {
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
