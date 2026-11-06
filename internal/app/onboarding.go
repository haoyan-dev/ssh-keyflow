package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	sshops "ssh-keyflow/internal/ssh"
	"ssh-keyflow/internal/state"
)

func onboard(ctx context.Context, options Options) error {
	return state.WithLock(options.Home, func(store *state.LockedStore) error {
		document, err := store.Load()
		if err != nil {
			return err
		}
		if err := integrity(document); err != nil {
			return err
		}
		mode, err := prompt(options, "n: New destination; r: Resume onboarding/manual guidance; a: Automatic registration of saved destination; 0: Back")
		if err != nil {
			return err
		}
		if mode == "r" || mode == "a" {
			for jobIndex, job := range document.Onboardings {
				fmt.Fprintf(options.Output, "%d: %s host=%s stage=%s\n", jobIndex+1, job.ID, job.HostID, job.Stage)
			}
			answer, err := prompt(options, "Onboarding number (0 cancels):")
			if err != nil {
				return err
			}
			selected, err := strconv.Atoi(answer)
			if err != nil || selected < 1 || selected > len(document.Onboardings) {
				return fmt.Errorf("no onboarding selected")
			}
			answer, err = prompt(options, "Type yes to resume saved destination and key:")
			if err != nil {
				return err
			}
			if answer != "yes" {
				return nil
			}
			if mode == "a" {
				job := document.Onboardings[selected-1]
				return registerSaved(ctx, options, store, &document, job.HostID, job.KeyID)
			}
			return finishOnboarding(ctx, options, store, &document, selected-1)
		}
		if mode != "n" {
			return nil
		}
		host := state.Host{Port: 22}
		fields := []struct {
			label string
			value *string
		}{{"Destination alias:", &host.Alias}, {"Hostname, Tailscale name or IP (IPv6 without brackets):", &host.Hostname}, {"Login user:", &host.User}}
		for _, field := range fields {
			value, err := prompt(options, field.label)
			if err != nil {
				return err
			}
			*field.value = value
		}
		port, err := prompt(options, "Port (blank = 22):")
		if err != nil {
			return err
		}
		if port != "" {
			host.Port, err = strconv.Atoi(port)
			if err != nil {
				return fmt.Errorf("invalid port")
			}
		}
		for {
			host.OS, err = prompt(options, "Destination OS: linux, macos or windows (unknown requires selection):")
			if err != nil {
				return err
			}
			host.OS = strings.ToLower(host.OS)
			if host.OS != "unknown" {
				break
			}
		}
		if host.OS == "windows" {
			host.WindowsAccount, err = prompt(options, "Windows account: standard or administrator (shared administrators_authorized_keys):")
			if err != nil {
				return err
			}
			if host.WindowsAccount != "standard" && host.WindowsAccount != "administrator" {
				return fmt.Errorf("select the confirmed Windows account type")
			}
		}
		if err := host.Validate(); err != nil {
			return err
		}
		for _, existing := range document.Hosts {
			if strings.EqualFold(existing.Alias, host.Alias) {
				return fmt.Errorf("destination alias already managed")
			}
		}
		if err := checkAlias(options, host.Alias); err != nil {
			return err
		}
		for keyIndex, key := range document.Keys {
			fmt.Fprintf(options.Output, "%d: %s [%s] %s origin=%s connection=%s\n", keyIndex+1, key.Name, key.ID, key.Fingerprint, key.Origin, key.ConnectionCheck)
		}
		answer, err := prompt(options, "Managed key number to reuse, or n for a new key for this destination (0 cancels):")
		if err != nil {
			return err
		}
		if answer == "0" {
			return nil
		}
		var key state.Key
		if answer != "n" {
			selected, err := strconv.Atoi(answer)
			if err != nil || selected < 1 || selected > len(document.Keys) {
				return fmt.Errorf("invalid managed key selection")
			}
			key = document.Keys[selected-1]
		}
		return saveDestination(ctx, options, store, &document, host, key, answer == "n")
	})
}

func saveDestination(ctx context.Context, options Options, store *state.LockedStore, document *state.State, host state.Host, key state.Key, newKey bool) error {
	fmt.Fprintf(options.Output, "Destination %s: %s@%s:%d OS=%s account=%s\nKey: %s %s\n", host.Alias, host.User, host.Hostname, host.Port, host.OS, host.WindowsAccount, key.Name, key.Fingerprint)
	confirmation, err := prompt(options, "Type yes to save intent and choose automatic or manual registration:")
	if err != nil {
		return err
	}
	if confirmation != "yes" {
		return nil
	}
	if err := integrity(*document); err != nil {
		return err
	}
	if err := checkAlias(options, host.Alias); err != nil {
		return err
	}
	host.ID, err = newID("host")
	if err != nil {
		return err
	}
	jobID, err := newID("onboarding")
	if err != nil {
		return err
	}
	document.Hosts = append(document.Hosts, host)
	document.Onboardings = append(document.Onboardings, state.Onboarding{ID: jobID, HostID: host.ID, KeyID: key.ID, Stage: "planned", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)})
	if newKey {
		name := host.Alias
		if options.Request != nil {
			name = options.Request.Name
		}
		_, err = createKey(ctx, options, store, document, name, jobID)
		if err != nil {
			return err
		}
	} else if err := store.Save(*document); err != nil {
		return err
	}
	return finishOnboarding(ctx, options, store, document, len(document.Onboardings)-1)
}

func finishOnboarding(ctx context.Context, options Options, store *state.LockedStore, document *state.State, jobIndex int) error {
	job := &document.Onboardings[jobIndex]
	if job.Stage == "key_pending" {
		for creationIndex, creation := range document.Creations {
			if creation.ID == job.CreationID && creation.Stage != "complete" {
				if _, err := finishCreation(ctx, options, store, document, creationIndex); err != nil {
					return err
				}
			}
		}
	}
	if err := integrity(*document); err != nil {
		return err
	}
	var host state.Host
	var key state.Key
	for _, candidate := range document.Hosts {
		if candidate.ID == job.HostID {
			host = candidate
		}
	}
	for _, candidate := range document.Keys {
		if candidate.ID == job.KeyID {
			key = candidate
		}
	}
	if key.ID == "" {
		return fmt.Errorf("requires review: onboarding key unavailable")
	}
	found := false
	for _, registration := range document.Registrations {
		if registration.KeyID == key.ID && registration.HostID == host.ID {
			found = true
		}
	}
	if !found {
		identifier, err := newID("registration")
		if err != nil {
			return err
		}
		document.Registrations = append(document.Registrations, state.Registration{ID: identifier, KeyID: key.ID, HostID: host.ID, Status: "unknown", ConnectionCheck: "unverified", Operation: "register", Stage: "pending"})
	}
	job.Stage = "planned"
	if err := store.Save(*document); err != nil {
		return err
	}
	fmt.Fprintf(options.Output, "Intent saved for %s (%s@%s:%d), fingerprint %s; recorded history is unknown, connection=unverified.\n", host.Alias, host.User, host.Hostname, host.Port, key.Fingerprint)
	return registerSaved(ctx, options, store, document, host.ID, key.ID)
}

func manualGuidance(host state.Host, public []byte) (string, error) {
	if _, err := fingerprint(public); err != nil {
		return "", err
	}
	plan, err := sshops.StandardRegistration(host.OS, host.WindowsAccount, public)
	if err != nil {
		return "", err
	}
	portGuidance := ""
	if host.Port != 22 {
		portGuidance = fmt.Sprintf("Selected port: %d. Before connecting, set Port %d in the server sshd_config and restart SSH; review the firewall for that port.\n", host.Port, host.Port)
		if host.OS == "windows" {
			portGuidance += fmt.Sprintf("notepad \"$env:ProgramData/ssh/sshd_config\"\nRestart-Service sshd\nNew-NetFirewallRule -Name 'SSH-KeyFlow-%d' -DisplayName 'SSH KeyFlow port %d' -Enabled True -Direction Inbound -Protocol TCP -Action Allow -LocalPort %d\n", host.Port, host.Port, host.Port)
		} else {
			portGuidance += "sudoedit /etc/ssh/sshd_config\n"
		}
	}
	return plan.Enable + "\n" + portGuidance + "Target: " + plan.Target + "\nBEGIN REGISTRATION SCRIPT\n" + plan.Script + "END REGISTRATION SCRIPT", nil
}
