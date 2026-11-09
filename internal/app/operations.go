package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ssh-keyflow/internal/state"
)

var ErrCancelled = errors.New("cancelled; no further changes made")
var ErrInvalidInput = errors.New("invalid operation input")

type Request struct {
	Operation, Name, PublicPath, PrivatePath, Key, Host, Resume string
	Destination                                                 Destination
	GuideOptions                                                GuideOptions
}
type Destination = state.Host

type View struct {
	State  state.State `json:"state"`
	Review string      `json:"review,omitempty"`
}

// Snapshot never initializes storage, probes commands or changes legacy records.
func Snapshot(options Options) (View, error) {
	if options.Home == "" {
		return View{}, fmt.Errorf("Home is required")
	}
	document, err := state.Load(options.Home)
	if err != nil {
		return View{}, err
	}
	view := View{State: document}
	if err := integrity(document); err != nil {
		view.Review = err.Error()
	}
	for _, deletion := range document.Deletions {
		if deletion.Stage != "complete" {
			view.Review += fmt.Sprintf("\nLegacy deletion %s for key %s requires manual review; this key cannot be used. Files and record are preserved.", deletion.ID, deletion.KeyID)
		}
	}
	return view, nil
}

func (request Request) Validate() error {
	validName := func(name string) bool { return strings.TrimSpace(name) != "" && !strings.ContainsAny(name, "\x00\r\n") }
	switch request.Operation {
	case "list":
	case "create":
		if !validName(request.Name) {
			return fmt.Errorf("create requires a valid management name")
		}
	case "import":
		if !validName(request.Name) || request.PublicPath == "" {
			return fmt.Errorf("import requires a valid name and public key path")
		}
	case "add":
		host := request.Destination
		if host.Port == 0 {
			host.Port = 22
		}
		if err := host.Validate(); err != nil {
			return err
		}
		if host.OS == "unknown" || (host.OS == "windows" && host.WindowsAccount == "") {
			return fmt.Errorf("select the confirmed destination OS and Windows account type")
		}
		if request.Key == "" || request.Name != "" {
			return fmt.Errorf("destination requires an existing managed key; create or import it first")
		}
	case "associate", "dissociate", "guide":
		if request.Host == "" || request.Key == "" {
			return fmt.Errorf("%s requires a destination and managed key", request.Operation)
		}
		if request.Operation == "guide" {
			return request.GuideOptions.Validate()
		}
	case "public":
		if request.Key == "" {
			return fmt.Errorf("public requires a managed key")
		}
	case "remove-key":
		if request.Key == "" {
			return fmt.Errorf("remove-key requires a managed key")
		}
	case "remove-host":
		if request.Host == "" {
			return fmt.Errorf("remove-host requires a destination")
		}
	case "resume":
		if request.Resume == "" {
			return fmt.Errorf("resume requires an unfinished local creation ID")
		}
	default:
		return fmt.Errorf("unknown operation %q; remote operations and local file deletion are no longer supported", request.Operation)
	}
	return nil
}

func execute(ctx context.Context, options Options) error {
	request := options.Request
	if err := request.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	switch request.Operation {
	case "list":
		view, err := Snapshot(options)
		if err != nil {
			return err
		}
		for _, key := range view.State.Keys {
			fmt.Fprintf(options.Output, "Key %s %q %s\nPublic: %s\nPrivate: %s\n", key.ID, key.Name, key.Fingerprint, key.PublicPath, key.PrivatePath)
		}
		for _, host := range view.State.Hosts {
			fmt.Fprintf(options.Output, "Destination %s %s %s@%s:%d\n", host.ID, host.Alias, host.User, host.Hostname, host.Port)
		}
		if view.Review != "" {
			fmt.Fprintln(options.Output, view.Review)
		}
		return nil
	case "public", "guide":
		var text string
		var err error
		if request.Operation == "public" {
			text, err = PublicKey(options.Home, request.Key)
		} else {
			var guide SetupGuide
			guide, err = PrepareGuideWithOptions(options.Home, request.Host, request.Key, request.GuideOptions)
			text = guide.Text
		}
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(options.Output, text)
		return err
	}
	document, err := state.Load(options.Home)
	if err != nil {
		return err
	}
	if err := request.checkTargets(document); err != nil {
		return err
	}
	if request.Operation == "create" || request.Operation == "import" || request.Operation == "resume" {
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
		switch request.Operation {
		case "import":
			return importLocked(ctx, options, store)
		case "create":
			_, err := createKey(ctx, options, store, &document, request.Name)
			return err
		case "resume":
			for index, creation := range document.Creations {
				if creation.ID != request.Resume {
					continue
				}
				options.target = ReportTarget{KeyID: creation.KeyID, CreationID: creation.ID}
				if _, err := fmt.Fprintf(options.Output, "Resume %s %q stage=%s\nPrivate: %s\nPublic: %s\nExisting files are preserved; only this recorded local key creation will run.\n", creation.ID, creation.Name, creation.Stage, creation.PrivatePath, creation.PublicPath); err != nil {
					return err
				}
				if _, err := prompt(options, "Type yes to resume this creation:"); err != nil {
					return err
				}
				_, err := finishCreation(ctx, options, store, &document, index)
				return err
			}
		case "add", "associate":
			return saveAssociation(options, store, document)
		case "remove-key", "remove-host", "dissociate":
			return removeManaged(options, store, document)
		}
		return fmt.Errorf("%w: operation target disappeared", ErrInvalidInput)
	})
}

func saveAssociation(options Options, store *state.LockedStore, document state.State) error {
	request := options.Request
	key, err := usableKey(document, request.Key)
	if err != nil {
		return err
	}
	host := request.Destination
	if request.Operation == "add" {
		host.ID, err = newID("host")
		if err != nil {
			return err
		}
		host.ConfigKeyID = ""
		if host.Port == 0 {
			host.Port = 22
		}
		for _, existing := range document.Hosts {
			if strings.EqualFold(existing.Alias, host.Alias) {
				return fmt.Errorf("%w: destination alias already managed", ErrInvalidInput)
			}
		}
	} else {
		index, err := destinationIndex(document, request.Host)
		if err != nil {
			return err
		}
		host = document.Hosts[index]
		for _, association := range document.Associations {
			if association.HostID == host.ID && association.KeyID == key.ID {
				return fmt.Errorf("%w: key already associated with destination", ErrInvalidInput)
			}
		}
	}
	options.target = ReportTarget{KeyID: key.ID, HostID: host.ID}
	if _, err := fmt.Fprintf(options.Output, "Associate %q [%s] %s\nDestination %s [%s]: %s@%s:%d\nOnly local metadata is saved. Remote registration and connection are not checked.\n", key.Name, key.ID, key.Fingerprint, host.Alias, host.ID, host.User, host.Hostname, host.Port); err != nil {
		return err
	}
	if _, err := prompt(options, "Type yes to save this association:"); err != nil {
		return err
	}
	if _, err := usableKey(document, key.ID); err != nil {
		return err
	}
	if request.Operation == "add" {
		document.Hosts = append(document.Hosts, host)
	}
	document.Associations = append(document.Associations, state.Association{HostID: host.ID, KeyID: key.ID})
	if err := options.saveReport(store, document, Report{Kind: ReportDestination, Outcome: OutcomeApplied}); err != nil {
		return err
	}
	fmt.Fprintln(options.Output, "Association saved. Open the copy-paste guide to place the public key.")
	return nil
}

func (request Request) checkTargets(document state.State) error {
	switch request.Operation {
	case "remove-key":
		index, err := keyIndex(document, request.Key)
		if err != nil {
			return err
		}
		if pendingDeletion(document, document.Keys[index].ID) {
			return fmt.Errorf("requires manual review: unfinished legacy deletion; files and record are preserved")
		}
		return nil
	case "remove-host":
		_, err := destinationIndex(document, request.Host)
		return err
	case "dissociate":
		_, _, err := associationTargets(document, request.Host, request.Key)
		return err
	case "add":
		_, err := usableKey(document, request.Key)
		return err
	case "associate":
		if _, err := destinationIndex(document, request.Host); err != nil {
			return err
		}
		_, err := usableKey(document, request.Key)
		return err
	case "resume":
		for _, creation := range document.Creations {
			if creation.ID == request.Resume && creation.Stage != "complete" {
				return nil
			}
		}
		return fmt.Errorf("%w: unfinished local creation %q not found; legacy remote/deletion records require manual review", ErrInvalidInput, request.Resume)
	}
	return nil
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
	if pendingDeletion(document, key.ID) {
		return state.Key{}, fmt.Errorf("requires manual review: key %s has an unfinished legacy deletion; files and record are preserved", key.ID)
	}
	if err := integrity(state.State{Keys: []state.Key{key}}); err != nil {
		return state.Key{}, err
	}
	return key, nil
}

func destinationIndex(document state.State, reference string) (int, error) {
	for index, host := range document.Hosts {
		if host.ID == reference {
			return index, nil
		}
	}
	for index, host := range document.Hosts {
		if reference != "" && strings.EqualFold(host.Alias, reference) {
			return index, nil
		}
	}
	return 0, fmt.Errorf("%w: destination %q not found", ErrInvalidInput, reference)
}
