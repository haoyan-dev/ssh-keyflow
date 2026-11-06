package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"ssh-keyflow/internal/state"
)

var ErrCancelled = errors.New("cancelled; no further changes made")
var ErrInvalidInput = errors.New("invalid operation input")

// Request contains operation inputs, never credentials or simulated menu answers.
type Request struct {
	Operation    string
	Name         string
	PublicPath   string
	PrivatePath  string
	Key          string
	Destination  Destination
	Host         string
	Registration string
	Resume       string
	Bootstrap    string
	Identity     string
	Configure    bool
}

type Destination = state.Host

type View struct {
	State     state.State `json:"state"`
	Discovery string      `json:"discovery"`
	Review    string      `json:"review,omitempty"`
}

// Snapshot reads existing data without probing dependencies or creating storage.
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
	var output bytes.Buffer
	options.Output = &output
	list(options, document)
	view.Discovery = output.String()
	return view, nil
}

func execute(ctx context.Context, options Options) error {
	request := options.Request
	if err := request.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	if err := request.checkTargets(options.Home); err != nil {
		return err
	}
	switch request.Operation {
	case "list":
		view, err := Snapshot(options)
		if err == nil {
			if view.Review != "" {
				fmt.Fprintln(options.Output, view.Review, "(updates stopped)")
			}
			fmt.Fprint(options.Output, view.Discovery)
		}
		return err
	case "import":
		if _, err := options.Runner.LookPath("ssh-keygen"); err != nil {
			return err
		}
		if err := state.CheckWritable(options.Home); err != nil {
			return err
		}
		return state.WithLock(options.Home, func(store *state.LockedStore) error {
			return importLocked(ctx, options, store)
		})
	case "create":
		if _, err := options.Runner.LookPath("ssh-keygen"); err != nil {
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
			if err := integrity(document); err != nil {
				return err
			}
			_, err = createKey(ctx, options, store, &document, request.Name, "")
			return err
		})
	case "delete":
		return deleteLocal(options)
	case "register":
		return state.WithLock(options.Home, func(store *state.LockedStore) error {
			document, err := store.Load()
			if err != nil {
				return err
			}
			if err := integrity(document); err != nil {
				return err
			}
			hostIndex, err := destinationIndex(document, request.Host)
			if err != nil {
				return err
			}
			keyIndex, err := keyIndex(document, request.Key)
			if err != nil {
				return err
			}
			return registerSelected(ctx, options, store, &document, document.Hosts[hostIndex], document.Keys[keyIndex])
		})
	case "unregister":
		return unregister(ctx, options)
	case "resume":
		return resumeOperation(ctx, options)
	case "add":
		if request.Destination.Port == 0 {
			request.Destination.Port = 22
		}
		if err := state.CheckWritable(options.Home); err != nil {
			return err
		}
		return state.WithLock(options.Home, func(store *state.LockedStore) error {
			document, err := store.Load()
			if err != nil {
				return err
			}
			if err := integrity(document); err != nil {
				return err
			}
			for _, host := range document.Hosts {
				if strings.EqualFold(host.Alias, request.Destination.Alias) {
					return fmt.Errorf("%w: destination alias already managed", ErrInvalidInput)
				}
			}
			if err := checkAlias(options, request.Destination.Alias); err != nil {
				return err
			}
			var key state.Key
			if request.Key != "" {
				index, err := keyIndex(document, request.Key)
				if err != nil {
					return err
				}
				key = document.Keys[index]
			}
			return saveDestination(ctx, options, store, &document, request.Destination, key, request.Key == "")
		})
	default:
		return fmt.Errorf("unknown operation %q", request.Operation)
	}
}

func (request Request) checkTargets(home string) error {
	switch request.Operation {
	case "list", "create", "import":
		return nil
	case "add":
		if request.Key == "" {
			return nil
		}
	}
	document, err := state.Load(home)
	if err != nil {
		return err
	}
	switch request.Operation {
	case "add", "delete":
		_, err = keyIndex(document, request.Key)
		return err
	case "register":
		if _, err := destinationIndex(document, request.Host); err != nil {
			return err
		}
		_, err = keyIndex(document, request.Key)
		return err
	case "unregister":
		for _, registration := range document.Registrations {
			if registration.ID == request.Registration {
				return nil
			}
		}
		return fmt.Errorf("%w: registration %q not found", ErrInvalidInput, request.Registration)
	case "resume":
		for _, creation := range document.Creations {
			if creation.ID == request.Resume {
				return nil
			}
		}
		for _, job := range document.Onboardings {
			if job.ID == request.Resume {
				return nil
			}
		}
		for _, registration := range document.Registrations {
			if registration.ID == request.Resume {
				return nil
			}
		}
		for _, deletion := range document.Deletions {
			if deletion.ID == request.Resume && deletion.Stage != "complete" {
				return nil
			}
		}
		return fmt.Errorf("%w: resumable operation %q not found", ErrInvalidInput, request.Resume)
	}
	return nil
}

// Validate checks operation arguments without accessing disk or invoking SSH.
func (request Request) Validate() error {
	switch request.Operation {
	case "list":
		return nil
	case "create":
		if strings.TrimSpace(request.Name) == "" || strings.ContainsAny(request.Name, "\x00\r\n") {
			return fmt.Errorf("create requires a valid management name")
		}
	case "import":
		if strings.TrimSpace(request.Name) == "" || strings.ContainsAny(request.Name, "\x00\r\n") || request.PublicPath == "" {
			return fmt.Errorf("import requires a valid management name and public key path")
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
		if (request.Key == "") == (strings.TrimSpace(request.Name) == "") {
			return fmt.Errorf("destination requires exactly one managed key or new key name")
		}
		if strings.ContainsAny(request.Name, "\x00\r\n") {
			return fmt.Errorf("invalid new key management name")
		}
	case "delete":
		if request.Key == "" {
			return fmt.Errorf("delete requires a managed key")
		}
	case "register":
		if request.Host == "" || request.Key == "" {
			return fmt.Errorf("register requires a destination and managed key")
		}
	case "unregister":
		if request.Registration == "" {
			return fmt.Errorf("unregister requires a registration ID")
		}
	case "resume":
		if request.Resume == "" {
			return fmt.Errorf("resume requires an operation ID")
		}
	default:
		return fmt.Errorf("unknown operation %q", request.Operation)
	}
	if request.Operation == "add" || request.Operation == "register" || request.Operation == "unregister" || request.Operation == "resume" {
		if _, err := bootstrapMode(request.Bootstrap); err != nil {
			return err
		}
		if request.Bootstrap == "key" && request.Identity == "" {
			return fmt.Errorf("key authentication requires an identity path")
		}
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

func bootstrapMode(value string) (string, error) {
	switch value {
	case "", "manual":
		return "0", nil
	case "password":
		return "p", nil
	case "key":
		return "k", nil
	default:
		return "", fmt.Errorf("authentication must be manual, password or key")
	}
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

func resumeOperation(ctx context.Context, options Options) error {
	document, err := state.Load(options.Home)
	if err != nil {
		return err
	}
	id := options.Request.Resume
	for _, deletion := range document.Deletions {
		if deletion.ID == id && deletion.Stage != "complete" {
			options.Request.Key = deletion.KeyID
			return deleteLocal(options)
		}
	}
	for _, registration := range document.Registrations {
		if registration.ID == id {
			options.Request.Registration = id
			if registration.Operation == "unregister" {
				return unregister(ctx, options)
			}
			options.Request.Operation = "register"
			options.Request.Host, options.Request.Key = registration.HostID, registration.KeyID
			return execute(ctx, options)
		}
	}
	return state.WithLock(options.Home, func(store *state.LockedStore) error {
		document, err := store.Load()
		if err != nil {
			return err
		}
		if err := integrity(document); err != nil {
			return err
		}
		for index, creation := range document.Creations {
			if creation.ID == id {
				if creation.Stage == "complete" {
					return fmt.Errorf("creation is already complete")
				}
				if _, err := prompt(options, "Type yes to resume this recorded creation; existing files are preserved:"); err != nil {
					return err
				}
				_, err := finishCreation(ctx, options, store, &document, index)
				return err
			}
		}
		for index, job := range document.Onboardings {
			if job.ID == id {
				if _, err := prompt(options, "Type yes to resume saved destination and key:"); err != nil {
					return err
				}
				return finishOnboarding(ctx, options, store, &document, index)
			}
		}
		return fmt.Errorf("%w: resumable operation %q not found", ErrInvalidInput, id)
	})
}
