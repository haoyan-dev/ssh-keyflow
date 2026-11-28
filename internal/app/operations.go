package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"ssh-keyflow/internal/state"
)

var ErrCancelled = errors.New("cancelled; no further changes made")
var ErrInvalidInput = errors.New("invalid operation input")

// Request is one explicit change: create, import, remove or resume.
type Request struct {
	Operation, Name, PublicPath, PrivatePath, Key, Resume string
}

type View struct {
	State  state.State `json:"state"`
	Review string      `json:"review,omitempty"`
}

// Snapshot never initializes storage, probes commands or migrates state.
func Snapshot(options Options) (View, error) {
	if options.Home == "" {
		return View{}, fmt.Errorf("Home is required")
	}
	document, err := state.Load(options.Home)
	if err != nil {
		return View{}, err
	}
	view := View{State: document}
	var notes []string
	if err := integrity(document); err != nil {
		notes = append(notes, err.Error())
	}
	if document.Migration != nil {
		notes = append(notes, document.Migration.Notice())
	}
	view.Review = strings.Join(notes, "\n")
	return view, nil
}

func (request Request) Validate() error {
	validName := func(name string) bool { return strings.TrimSpace(name) != "" && !strings.ContainsAny(name, "\x00\r\n") }
	switch request.Operation {
	case "create":
		if !validName(request.Name) {
			return fmt.Errorf("create requires a valid management name")
		}
	case "import":
		if !validName(request.Name) || request.PublicPath == "" || request.PrivatePath == "" {
			return fmt.Errorf("import requires a valid name and both public and private key paths")
		}
	case "remove":
		if request.Key == "" {
			return fmt.Errorf("remove requires a managed key")
		}
	case "resume":
		if request.Resume == "" {
			return fmt.Errorf("resume requires an unfinished local creation ID")
		}
	default:
		return fmt.Errorf("unknown operation %q", request.Operation)
	}
	return nil
}

func execute(ctx context.Context, options Options) error {
	request := options.Request
	if err := request.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	document, err := state.Load(options.Home)
	if err != nil {
		return err
	}
	if err := request.checkTargets(document); err != nil {
		return err
	}
	if request.Operation != "remove" {
		if _, err := options.Runner.LookPath("ssh-keygen"); err != nil {
			return err
		}
	}
	if err := state.CheckWritable(options.Home); err != nil {
		return err
	}
	return state.WithLock(options.Home, func(store *state.LockedStore) error {
		document, err := store.Load()
		if err != nil {
			return err
		}
		if err := request.checkTargets(document); err != nil {
			return err
		}
		if document.Migration != nil {
			// Shown before the preview so the confirmation covers the migration too.
			if _, err := fmt.Fprintln(options.Output, document.Migration.Notice()); err != nil {
				return err
			}
		}
		switch request.Operation {
		case "import":
			return importLocked(ctx, options, store)
		case "create":
			_, err := createKey(ctx, options, store, &document, request.Name)
			return err
		case "remove":
			return removeKey(options, store, document)
		case "resume":
			index := slices.IndexFunc(document.Creations, func(creation state.Creation) bool { return creation.ID == request.Resume })
			if index < 0 {
				return fmt.Errorf("%w: unfinished local creation %q not found", ErrInvalidInput, request.Resume)
			}
			creation := document.Creations[index]
			if _, err := fmt.Fprintf(options.Output, "Resume %s %q stage=%s\nPrivate: %s\nPublic: %s\nExisting files are preserved; only this recorded local key creation will run.\n", creation.ID, creation.Name, creation.Stage, creation.PrivatePath, creation.PublicPath); err != nil {
				return err
			}
			if _, err := prompt(options, "Type yes to resume this creation:"); err != nil {
				return err
			}
			_, err := finishCreation(ctx, options, store, &document, index)
			return err
		}
		return fmt.Errorf("%w: operation target disappeared", ErrInvalidInput)
	})
}

func (request Request) checkTargets(document state.State) error {
	switch request.Operation {
	case "remove":
		index, err := keyIndex(document, request.Key)
		if err != nil {
			return err
		}
		if document.Node != nil && slices.Contains(document.Node.ClientKeyIDs, document.Keys[index].ID) {
			return fmt.Errorf("managed key %q is this node's client key; change it with node set first", document.Keys[index].Name)
		}
		return nil
	case "resume":
		if slices.ContainsFunc(document.Creations, func(creation state.Creation) bool { return creation.ID == request.Resume }) {
			return nil
		}
		return fmt.Errorf("%w: unfinished local creation %q not found", ErrInvalidInput, request.Resume)
	}
	return nil
}

func removeKey(options Options, store *state.LockedStore, document state.State) error {
	index, err := keyIndex(document, options.Request.Key)
	if err != nil {
		return err
	}
	key := document.Keys[index]
	if _, err := fmt.Fprintf(options.Output, "Remove managed key %q [%s]\nFingerprint: %s\nPublic: %s\nPrivate: %s\nOnly local management metadata changes. Key files and SSH access are preserved.\n", key.Name, key.ID, key.Fingerprint, key.PublicPath, key.PrivatePath); err != nil {
		return err
	}
	if _, err := prompt(options, "Type yes to confirm:"); err != nil {
		return err
	}
	document.Keys = slices.Delete(document.Keys, index, index+1)
	if err := store.Save(document); err != nil {
		return err
	}
	_, err = fmt.Fprintln(options.Output, "Managed key removed. Key files and SSH access are unchanged.")
	return err
}

func keyIndex(document state.State, reference string) (int, error) {
	for index, key := range document.Keys {
		if key.ID == reference {
			return index, nil
		}
	}
	found := -1
	for index, key := range document.Keys {
		if reference != "" && key.Name == reference {
			if found >= 0 {
				return 0, fmt.Errorf("%w: ambiguous managed key %q; use its fixed ID", ErrInvalidInput, reference)
			}
			found = index
		}
	}
	if found < 0 {
		return 0, fmt.Errorf("%w: managed key %q not found", ErrInvalidInput, reference)
	}
	return found, nil
}

func usableKey(document state.State, reference string) (state.Key, error) {
	index, err := keyIndex(document, reference)
	if err != nil {
		return state.Key{}, err
	}
	key := document.Keys[index]
	if err := integrity(state.State{Keys: []state.Key{key}}); err != nil {
		return state.Key{}, err
	}
	return key, nil
}

// PublicKey returns only the public algorithm and blob, never a file comment.
func PublicKey(home, reference string) (string, error) {
	if home == "" {
		return "", fmt.Errorf("Home is required")
	}
	document, err := state.Load(home)
	if err != nil {
		return "", err
	}
	key, err := usableKey(document, reference)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(key.PublicPath)
	if err != nil {
		return "", err
	}
	if identity, err := fingerprint(data); err != nil || identity != key.Fingerprint {
		return "", fmt.Errorf("requires review: public key fingerprint mismatch")
	}
	fields := strings.Fields(string(data))
	return fields[0] + " " + fields[1], nil
}
