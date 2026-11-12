package app

import (
	"fmt"
	"slices"

	"ssh-keyflow/internal/state"
)

func associationTargets(document state.State, hostRef, keyRef string) (state.Host, state.Key, error) {
	hostIndex, err := destinationIndex(document, hostRef)
	if err != nil {
		return state.Host{}, state.Key{}, err
	}
	keyPosition, err := keyIndex(document, keyRef)
	if err != nil {
		return state.Host{}, state.Key{}, err
	}
	host, key := document.Hosts[hostIndex], document.Keys[keyPosition]
	for _, association := range document.Associations {
		if association.HostID == host.ID && association.KeyID == key.ID {
			return host, key, nil
		}
	}
	return state.Host{}, state.Key{}, fmt.Errorf("%w: key is not associated with destination", ErrInvalidInput)
}

func removalPinNotice(home, hostID string) string {
	pins, err := HostKeyPins(home)
	if err != nil {
		return fmt.Sprintf("Host key pins: unknown (%v). Removal does not change the pin file.", err)
	}
	if count := len(pins[hostID]); count > 0 {
		return fmt.Sprintf("%d pinned host key(s) stay in %s. To remove them, cancel and run ssh-keyflow hosts unpin --host %s first.", count, state.PinPath(home), hostID)
	}
	return ""
}

func removeManaged(options Options, store *state.LockedStore, document state.State) error {
	request := options.Request
	keyPosition, hostPosition := -1, -1
	var err error
	switch request.Operation {
	case "remove-key":
		keyPosition, err = keyIndex(document, request.Key)
		if err != nil {
			return err
		}
		key := document.Keys[keyPosition]
		options.target.KeyID = key.ID
		if _, err := fmt.Fprintf(options.Output, "Remove managed key %q [%s]\nFingerprint: %s\nPublic: %s\nPrivate: %s\nAll local associations for this key will be removed.\n", key.Name, key.ID, key.Fingerprint, key.PublicPath, key.PrivatePath); err != nil {
			return err
		}
	case "remove-host":
		hostPosition, err = destinationIndex(document, request.Host)
		if err != nil {
			return err
		}
		host := document.Hosts[hostPosition]
		options.target.HostID = host.ID
		if _, err := fmt.Fprintf(options.Output, "Remove destination %s [%s]: %s@%s:%d\nAll local associations for this destination will be removed.\n", host.Alias, host.ID, host.User, host.Hostname, host.Port); err != nil {
			return err
		}
		// Removal keeps pinned host keys; unpinning needs a managed destination.
		if notice := removalPinNotice(options.Home, host.ID); notice != "" {
			if _, err := fmt.Fprintln(options.Output, notice); err != nil {
				return err
			}
		}
	case "dissociate":
		host, key, err := associationTargets(document, request.Host, request.Key)
		if err != nil {
			return err
		}
		options.target = ReportTarget{HostID: host.ID, KeyID: key.ID}
		if _, err := fmt.Fprintf(options.Output, "Disconnect key %q [%s] from destination %s [%s].\nThe key and destination will remain managed.\n", key.Name, key.ID, host.Alias, host.ID); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(options.Output, "Only local management metadata changes. Key files, SSH configuration, remote authorized keys and legacy history are preserved."); err != nil {
		return err
	}
	if _, err := prompt(options, "Type yes to confirm:"); err != nil {
		return err
	}
	document.Associations = slices.DeleteFunc(document.Associations, func(association state.Association) bool {
		return (options.target.KeyID == "" || association.KeyID == options.target.KeyID) &&
			(options.target.HostID == "" || association.HostID == options.target.HostID)
	})
	// Retain identities so creation results and legacy history remain valid.
	if keyPosition >= 0 {
		document.RemovedKeys = append(document.RemovedKeys, document.Keys[keyPosition])
		document.Keys = slices.Delete(document.Keys, keyPosition, keyPosition+1)
	}
	if hostPosition >= 0 {
		document.RemovedHosts = append(document.RemovedHosts, document.Hosts[hostPosition])
		document.Hosts = slices.Delete(document.Hosts, hostPosition, hostPosition+1)
	}
	if err := options.saveReport(store, document, Report{Kind: ReportRemoval, Outcome: OutcomeApplied}); err != nil {
		return err
	}
	_, err = fmt.Fprintln(options.Output, "Local management metadata removed. Key files and SSH access are unchanged.")
	return err
}
