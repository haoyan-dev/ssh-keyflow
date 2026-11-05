package app

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	sshops "ssh-keyflow/internal/ssh"
	"ssh-keyflow/internal/state"
)

func registerSaved(ctx context.Context, options Options, store *state.LockedStore, document *state.State, hostID, keyID string) error {
	if err := integrity(*document); err != nil {
		return err
	}
	if len(document.Configs) > 0 {
		if _, _, err := ownedConfig(options, document); err != nil {
			return err
		}
	}
	var host state.Host
	var key state.Key
	for _, candidate := range document.Hosts {
		if candidate.ID == hostID {
			host = candidate
		}
	}
	for _, candidate := range document.Keys {
		if candidate.ID == keyID {
			key = candidate
		}
	}
	if host.ID == "" || key.ID == "" {
		return fmt.Errorf("requires review: destination/key unavailable")
	}
	public, err := os.ReadFile(key.PublicPath)
	if err != nil {
		return err
	}
	plan, err := sshops.StandardRegistration(host.OS, host.WindowsAccount, public)
	if err != nil {
		return err
	}
	fallback := func(cause error) error {
		for index, job := range document.Onboardings {
			if job.HostID == host.ID && job.KeyID == key.ID {
				document.Onboardings[index].Stage = "manual"
			}
		}
		if err := store.Save(*document); err != nil {
			return err
		}
		guidance, guideErr := manualGuidance(host, public)
		if guideErr != nil {
			return guideErr
		}
		fmt.Fprintf(options.Output, "Automatic registration stopped: %v\nCheck address, port, network, server, authentication, permissions and standard sshd configuration. Direct SSH uses -F none; configured ProxyJump and Match options are not loaded. Host-key changes require independent review; no known_hosts entry is removed. Manual registration:\n%s\n", cause, guidance)
		return nil
	}
	mode, err := prompt(options, "Bootstrap login: p password/keyboard-interactive; k existing private key; 0 manual:")
	if err != nil {
		return err
	}
	if mode != "p" && mode != "k" {
		return fallback(fmt.Errorf("manual selected"))
	}
	identity := ""
	if mode == "k" {
		identity, err = prompt(options, "Existing private key path for bootstrap login:")
		if err != nil {
			return err
		}
		identity, err = filepath.Abs(expandPath(options.Home, identity))
		if err != nil {
			return err
		}
		info, err := os.Lstat(identity)
		if err != nil || !info.Mode().IsRegular() {
			return fallback(fmt.Errorf("bootstrap key must be a regular file"))
		}
	}
	probe, err := sshops.RegistrationProbe(host.OS, host.WindowsAccount, host.User)
	if err != nil {
		return fallback(err)
	}
	run := func(script string, verify bool, first bool) (string, error) {
		for _, path := range []string{options.Home, identity} {
			if strings.ContainsAny(path, "\r\n\x00\"%$") {
				return "", fmt.Errorf("SSH path requires manual setup: unsupported quoting or expansion token")
			}
		}
		arguments := sshArguments(options.Home, host, identity, mode, verify, first)
		var output bytes.Buffer
		arguments = append(arguments, sshops.RemoteCommand(host.OS, script))
		err := options.Runner.Run(ctx, Command{Name: "ssh", Args: arguments, Stdin: options.Input, Stdout: &output, Stderr: options.Output, Interactive: true})
		return strings.TrimSpace(output.String()), err
	}
	observed, err := run(sshops.OperationScript(host.OS, "KEYFLOW_PROBE", probe), false, true)
	if err != nil {
		return fallback(err)
	}
	if !strings.HasPrefix(observed, "KEYFLOW_TARGET:") || strings.ContainsAny(observed, "\r\n\x00") || len(observed) <= len("KEYFLOW_TARGET:") {
		return fallback(fmt.Errorf("unable to verify OS and effective standard target"))
	}
	fmt.Fprintf(options.Output, "Verified %s login %s; %s\n", host.OS, host.User, observed)
	approval, err := prompt(options, "Type yes to confirm the running server uses its default sshd configuration and approve this exact standard target (custom daemon -f/-o requires manual review):")
	if err != nil {
		return err
	}
	if approval != "yes" {
		return fallback(fmt.Errorf("target not approved"))
	}
	if plan.Shared {
		approval, err = prompt(options, "This exact Windows administrator file is shared by administrator logins. Type yes to approve that shared impact:")
		if err != nil {
			return err
		}
		if approval != "yes" {
			return fallback(fmt.Errorf("shared administrator target not approved"))
		}
	}
	registrationIndex := -1
	for index, registration := range document.Registrations {
		if registration.KeyID == key.ID && registration.HostID == host.ID {
			registrationIndex = index
		}
	}
	if registrationIndex < 0 {
		identifier, err := newID("registration")
		if err != nil {
			return err
		}
		document.Registrations = append(document.Registrations, state.Registration{ID: identifier, KeyID: key.ID, HostID: host.ID, Status: "unknown", ConnectionCheck: "unverified", Operation: "register", Stage: "planned"})
		registrationIndex = len(document.Registrations) - 1
	}
	registration := &document.Registrations[registrationIndex]
	uncertain := registration.Stage == "unknown" || registration.Status == "registered"
	for _, event := range document.History {
		if event.RegistrationID == registration.ID {
			uncertain = true
		}
	}
	guarded := func(body string) string { return sshops.GuardedRegistration(host.OS, probe, observed, body) }
	present := false
	if uncertain {
		reader, err := sshops.RegistrationRead(host.OS, host.WindowsAccount, public)
		if err != nil {
			return err
		}
		result, err := run(sshops.OperationScript(host.OS, "KEYFLOW_RECONCILE", guarded(reader)), false, false)
		if err != nil || (result != "KEYFLOW_PRESENT" && result != "KEYFLOW_ABSENT") {
			return fallback(fmt.Errorf("read-only reconciliation failed; no retry performed: %v", err))
		}
		present = result == "KEYFLOW_PRESENT"
		if !present {
			registration.Status, registration.Stage = "unknown", "planned"
			if err := recordRegistration(store, document, registrationIndex); err != nil {
				return err
			}
			approval, err := prompt(options, "Read-only reconciliation found the key absent. Type yes to retry registration:")
			if err != nil {
				return err
			}
			if approval != "yes" {
				return nil
			}
		}
	}
	if !present {
		registration.Status, registration.Stage, registration.Operation = "unknown", "pending", "register"
		registration.ConnectionCheck = "unverified"
		if err := recordRegistration(store, document, registrationIndex); err != nil {
			return err
		}
		body := sshops.ScriptWithSuccess(host.OS, plan.Script, "KEYFLOW_REGISTERED")
		result, remoteErr := run(sshops.OperationScript(host.OS, "KEYFLOW_REGISTER", guarded(body)), false, false)
		if remoteErr != nil || !strings.HasSuffix(result, "KEYFLOW_REGISTERED") {
			registration.Status, registration.Stage = "unknown", "unknown"
			if err := recordRegistration(store, document, registrationIndex); err != nil {
				return err
			}
			return fallback(fmt.Errorf("result unknown; resume to reconcile before any retry: %v", remoteErr))
		}
	}
	registration.Status, registration.Stage = "registered", "complete"
	registration.ConnectionCheck = "unverified"
	if err := recordRegistration(store, document, registrationIndex); err != nil {
		return err
	}
	if key.PrivatePath != "" {
		if err := integrity(*document); err != nil {
			return err
		}
		identity = key.PrivatePath
		verification := sshops.ScriptWithSuccess(host.OS, "", "KEYFLOW_VERIFIED")
		result, verifyErr := run(verification, true, false)
		registration.ConnectionCheck = "failed"
		if verifyErr == nil && result == "KEYFLOW_VERIFIED" {
			registration.ConnectionCheck = "verified"
		}
		if err := recordRegistration(store, document, registrationIndex); err != nil {
			return err
		}
	}
	fmt.Fprintf(options.Output, "%s: registered; connection=%s (specified key only).\n", host.Alias, registration.ConnectionCheck)
	return configureRegistration(options, store, document, host, key)
}

func recordRegistration(store *state.LockedStore, document *state.State, index int) error {
	registration := document.Registrations[index]
	document.History = append(document.History, state.RegistrationEvent{RegistrationID: registration.ID, At: time.Now().UTC().Format(time.RFC3339Nano), Status: registration.Status, ConnectionCheck: registration.ConnectionCheck, Stage: registration.Stage, Operation: registration.Operation})
	return store.Save(*document)
}

func registerAnother(ctx context.Context, options Options) error {
	return state.WithLock(options.Home, func(store *state.LockedStore) error {
		document, err := store.Load()
		if err != nil {
			return err
		}
		if err := integrity(document); err != nil {
			return err
		}
		for index, host := range document.Hosts {
			fmt.Fprintf(options.Output, "%d: %s %s@%s:%d\n", index+1, host.Alias, host.User, host.Hostname, host.Port)
		}
		selectedHost, err := prompt(options, "Destination number (0 cancels):")
		if err != nil || selectedHost == "0" {
			return err
		}
		hostIndex, err := strconv.Atoi(selectedHost)
		if err != nil || hostIndex < 1 || hostIndex > len(document.Hosts) {
			return fmt.Errorf("no destination selected")
		}
		for index, key := range document.Keys {
			fmt.Fprintf(options.Output, "%d: %s %s\n", index+1, key.Name, key.Fingerprint)
		}
		selectedKey, err := prompt(options, "Managed key number (0 cancels):")
		if err != nil || selectedKey == "0" {
			return err
		}
		keyIndex, err := strconv.Atoi(selectedKey)
		if err != nil || keyIndex < 1 || keyIndex > len(document.Keys) {
			return fmt.Errorf("no managed key selected")
		}
		host, key := document.Hosts[hostIndex-1], document.Keys[keyIndex-1]
		fmt.Fprintf(options.Output, "Register %s at %s (%s@%s:%d)\n", key.Fingerprint, host.Alias, host.User, host.Hostname, host.Port)
		answer, err := prompt(options, "Type yes to register this managed public key:")
		if err != nil || answer != "yes" {
			return err
		}
		return registerSaved(ctx, options, store, &document, host.ID, key.ID)
	})
}

func sshArguments(home string, host state.Host, identity, mode string, verify, first bool) []string {
	arguments := []string{"-F", "none", "-o", "ControlMaster=no", "-o", "ControlPath=none", "-o", "IdentityAgent=none", "-o", "IdentitiesOnly=yes", "-o", "CertificateFile=none", "-o", "IdentityFile=none", "-o", "ForwardAgent=no", "-o", "ClearAllForwardings=yes", "-o", "PermitLocalCommand=no", "-o", "UserKnownHostsFile=" + strconv.Quote(filepath.ToSlash(filepath.Join(home, ".ssh", "known_hosts"))) + " " + strconv.Quote(filepath.ToSlash(filepath.Join(home, ".ssh", "known_hosts2")))}
	checking := "yes"
	if first {
		checking = "ask"
	}
	arguments = append(arguments, "-o", "StrictHostKeyChecking="+checking)
	if verify || mode == "k" {
		arguments = append(arguments, "-i", identity, "-o", "PreferredAuthentications=publickey", "-o", "PubkeyAuthentication=yes", "-o", "PasswordAuthentication=no", "-o", "KbdInteractiveAuthentication=no", "-o", "GSSAPIAuthentication=no", "-o", "HostbasedAuthentication=no")
	} else {
		arguments = append(arguments, "-o", "PubkeyAuthentication=no", "-o", "PreferredAuthentications=keyboard-interactive,password", "-o", "PasswordAuthentication=yes", "-o", "KbdInteractiveAuthentication=yes")
	}
	return append(arguments, "-p", strconv.Itoa(host.Port), "-l", host.User, "--", host.Hostname)
}
