package app

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	sshops "ssh-keyflow/internal/ssh"
	"ssh-keyflow/internal/state"
)

func unregister(ctx context.Context, options Options) error {
	return state.WithLock(options.Home, func(store *state.LockedStore) error {
		document, err := store.Load()
		if err != nil {
			return err
		}
		if err := integrity(document); err != nil {
			return err
		}
		if _, _, err := ownedConfig(options, &document); err != nil {
			return err
		}
		for index, registration := range document.Registrations {
			fmt.Fprintf(options.Output, "%d: registration %s key=%s host=%s status=%s operation=%s stage=%s\n", index+1, registration.ID, registration.KeyID, registration.HostID, registration.Status, registration.Operation, registration.Stage)
		}
		index := -1
		if options.Request != nil {
			for candidateIndex, registration := range document.Registrations {
				if registration.ID == options.Request.Registration {
					index = candidateIndex
				}
			}
			if index < 0 {
				return fmt.Errorf("%w: registration %q not found", ErrInvalidInput, options.Request.Registration)
			}
		} else {
			choice, err := prompt(options, "Registration number to unregister; 0: Back")
			if err != nil || choice == "0" {
				return err
			}
			selection, err := strconv.Atoi(choice)
			if err != nil || selection < 1 || selection > len(document.Registrations) {
				return fmt.Errorf("invalid registration selection")
			}
			index = selection - 1
		}
		registration := &document.Registrations[index]
		var host state.Host
		var key state.Key
		for _, candidate := range document.Hosts {
			if candidate.ID == registration.HostID {
				host = candidate
			}
		}
		for _, candidate := range document.Keys {
			if candidate.ID == registration.KeyID {
				key = candidate
			}
		}
		if key.ID == "" && registration.Status == "removed" {
			fmt.Fprintln(options.Output, "Already removed; local key has been deleted; history retained")
			return nil
		}
		public, err := plainConfig(key.PublicPath)
		if err != nil || public == nil {
			return fmt.Errorf("requires review: public key path unavailable: %v", err)
		}
		plan, err := sshops.StandardRevocation(host.OS, host.WindowsAccount, public)
		if err != nil {
			return err
		}
		fallback := func(reason string) error {
			registration.Operation = "unregister"
			if registration.Status != "removed" {
				registration.Stage = "unknown"
			}
			if err := recordRegistration(store, &document, index); err != nil {
				return err
			}
			fmt.Fprintf(options.Output, "Automatic revocation stopped: %s\nManual revocation for %s@%s:%d, fingerprint %s, target %s:\nKeep local files and an existing session. Independently authenticate without this fingerprint, confirm the server's effective configuration and account, then run this same script on the server. Re-run menu 4 for read-only reconciliation before local deletion. POSIX requires Perl. Shared administrator targets affect every administrator; do not elevate or reset existing ACLs.\n%s\n", reason, host.User, host.Hostname, host.Port, key.Fingerprint, plan.Target, plan.Script)
			if options.Request != nil && options.Request.Bootstrap != "" && options.Request.Bootstrap != "manual" {
				return fmt.Errorf("%s", reason)
			}
			return nil
		}
		mode, err := "", error(nil)
		if options.Request != nil {
			mode, err = bootstrapMode(options.Request.Bootstrap)
		} else {
			mode, err = prompt(options, "Fresh alternate authentication excluding target fingerprint: p: Password/keyboard-interactive; k: Distinct private key; 0: Manual guidance")
		}
		if err != nil {
			return err
		}
		identity, alternateDigest, alternateFingerprint := "", "", ""
		if mode == "k" {
			if options.Request != nil {
				identity = options.Request.Identity
			} else {
				identity, err = prompt(options, "Alternate private key path (passphrase stays in native ssh-keygen)")
			}
			if err != nil {
				return err
			}
			identity = expandPath(options.Home, identity)
			private, readErr := plainConfig(identity)
			if readErr != nil || len(private) == 0 || strings.ContainsAny(identity, "\r\n\x00\"%$") {
				return fallback("alternate private key requires review")
			}
			alternateDigest = configDigest(private)
			var derived bytes.Buffer
			if err := options.Runner.Run(ctx, Command{Name: "ssh-keygen", Args: []string{"-y", "-f", identity}, Stdin: options.Input, Stdout: &derived, Stderr: options.Output, Interactive: true}); err != nil {
				return fallback("alternate key could not be authenticated")
			}
			alternateFingerprint, err = fingerprint(derived.Bytes())
			if err != nil || alternateFingerprint == key.Fingerprint || strings.Contains(strings.Fields(derived.String())[0], "-cert-") {
				return fallback("alternate key must exclude target fingerprint and certificates, even at another path")
			}
		} else if mode != "p" {
			return fallback("no alternate authentication selected; local files preserved")
		}
		probe, err := sshops.RegistrationProbe(host.OS, host.WindowsAccount, host.User)
		if err != nil {
			return fallback(err.Error())
		}
		run := func(operation, body string, first bool) (string, error) {
			if mode == "k" {
				current, err := plainConfig(identity)
				if err != nil || configDigest(current) != alternateDigest {
					return "", fmt.Errorf("alternate identity changed; requires review")
				}
			}
			arguments := sshArguments(options.Home, host, identity, mode, mode == "k", first)
			arguments = append([]string{"-o", "GSSAPIAuthentication=no", "-o", "HostbasedAuthentication=no", "-o", "PKCS11Provider=none"}, arguments...)
			arguments = append(arguments, sshops.RemoteCommand(host.OS, sshops.OperationScript(host.OS, operation, body)))
			var output bytes.Buffer
			err := options.Runner.Run(ctx, Command{Name: "ssh", Args: arguments, Stdin: options.Input, Stdout: &output, Stderr: options.Output, Interactive: true})
			return strings.TrimSpace(output.String()), err
		}
		observed, err := run("KEYFLOW_ALTERNATE", probe, true)
		if err != nil || !strings.HasPrefix(observed, "KEYFLOW_TARGET:") || strings.ContainsAny(observed, "\r\n") {
			return fallback("fresh alternate login or effective server configuration could not be verified; local files preserved")
		}
		authentication := "password/keyboard-interactive"
		if mode == "k" {
			authentication = alternateFingerprint
		}
		fmt.Fprintf(options.Output, "Verified fresh alternate authentication for %s@%s:%d (%s)\n", host.User, host.Hostname, host.Port, authentication)
		answer, err := prompt(options, "Type yes to confirm the running SSH daemon uses the default configuration and the inspected standard target: "+observed)
		if err != nil {
			return err
		}
		if answer != "yes" {
			return fallback("effective daemon configuration not approved")
		}
		readBody := sshops.RevocationReadBody(host.OS, plan.InspectScript)
		presence, err := run("KEYFLOW_REVOCATION_READ", sshops.GuardedRegistration(host.OS, probe, observed, readBody), false)
		if err != nil || (presence != "KEYFLOW_PRESENT" && presence != "KEYFLOW_ABSENT") {
			return fallback("read-only reconciliation failed; no remote mutation attempted")
		}
		registration.Operation, registration.ConnectionCheck = "unregister", "unverified"
		if presence == "KEYFLOW_ABSENT" {
			registration.Status, registration.Stage = "removed", "complete"
			if err := recordRegistration(store, &document, index); err != nil {
				return err
			}
			fmt.Fprintln(options.Output, "Already absent; read-only reconciliation recorded; local files preserved")
			return removeKeyConfiguration(options, store, &document, key.ID, host.ID)
		}
		registration.Status, registration.Stage = "registered", "planned"
		if err := recordRegistration(store, &document, index); err != nil {
			return err
		}
		if plan.Shared {
			answer, err := prompt(options, "Shared administrator target affects all administrators. Type yes to approve this impact")
			if err != nil || answer != "yes" {
				return fmt.Errorf("shared administrator revocation not approved")
			}
		}
		answer, err = prompt(options, fmt.Sprintf("Final confirmation: unregister only %s from %s@%s:%d (%s). Type yes", key.Fingerprint, host.User, host.Hostname, host.Port, observed))
		if err != nil {
			return err
		}
		if answer != "yes" {
			fmt.Fprintln(options.Output, "Cancelled; local files preserved; no remote write")
			return nil
		}
		registration.Status, registration.Stage = "unknown", "pending"
		if err := recordRegistration(store, &document, index); err != nil {
			return err
		}
		body := sshops.ScriptWithSuccess(host.OS, plan.Script, "KEYFLOW_REMOVED")
		result, err := run("KEYFLOW_UNREGISTER", sshops.GuardedRegistration(host.OS, probe, observed, body), false)
		if err != nil || (result != "KEYFLOW_REMOVED" && !strings.HasSuffix(result, "\nKEYFLOW_REMOVED")) {
			return fallback("remote outcome unknown; retry starts with read-only reconciliation; local files preserved")
		}
		registration.Status, registration.Stage = "removed", "complete"
		if err := recordRegistration(store, &document, index); err != nil {
			return err
		}
		fmt.Fprintln(options.Output, "Public key removed; local files preserved; use separate menu 6 for guarded local deletion")
		return removeKeyConfiguration(options, store, &document, key.ID, host.ID)
	})
}
