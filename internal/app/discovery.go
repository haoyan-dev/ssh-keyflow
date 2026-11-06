package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"ssh-keyflow/internal/state"
)

func list(options Options, document state.State) {
	for _, deleted := range document.DeletedKeys {
		fmt.Fprintf(options.Output, "deleted key %s [%s] %s deleted_at=%s; tombstone and registration history retained\n", deleted.Key.Name, deleted.Key.ID, deleted.Key.Fingerprint, deleted.DeletedAt)
	}
	for _, deletion := range document.Deletions {
		if deletion.Stage != "complete" {
			fmt.Fprintf(options.Output, "local deletion %s key=%s stage=%s; resume this saved operation to reconcile\n", deletion.ID, deletion.KeyID, deletion.Stage)
		}
	}
	for _, key := range document.Keys {
		fmt.Fprintf(options.Output, "managed key %s [%s] %s %s connection=%s origin=%s imported_at=%s created_at=%s\n", key.Name, key.ID, key.PublicPath, key.Fingerprint, key.ConnectionCheck, key.Origin, key.ImportedAt, key.CreatedAt)
	}
	for _, host := range document.Hosts {
		for _, job := range document.Onboardings {
			if job.HostID == host.ID {
				fmt.Fprintf(options.Output, "onboarding %s host=%s key=%s stage=%s; resume this saved operation for guidance\n", job.ID, host.Alias, job.KeyID, job.Stage)
			}
		}
		fmt.Fprintf(options.Output, "managed host %s [%s] %s@%s:%d\n", host.Alias, host.ID, host.User, host.Hostname, host.Port)
	}
	for _, registration := range document.Registrations {
		fmt.Fprintf(options.Output, "registration %s key=%s host=%s status=%s connection=%s operation=%s stage=%s (recorded history)\n", registration.ID, registration.KeyID, registration.HostID, registration.Status, registration.ConnectionCheck, registration.Operation, registration.Stage)
	}
	for _, creation := range document.Creations {
		if creation.Stage != "complete" {
			fmt.Fprintf(options.Output, "creation %s [%s] %s stage=%s; resume this saved operation\n", creation.Name, creation.ID, creation.PrivatePath, creation.Stage)
		}
	}
	managed := map[string]bool{}
	for _, creation := range document.Creations {
		managed[filepath.Clean(creation.PrivatePath)], managed[filepath.Clean(creation.PublicPath)] = true, true
	}
	for _, key := range document.Keys {
		managed[filepath.Clean(key.PublicPath)] = true
		if key.PrivatePath != "" {
			managed[filepath.Clean(key.PrivatePath)] = true
		}
	}
	root := filepath.Join(options.Home, ".ssh")
	for _, directory := range []string{root, filepath.Join(state.Directory(options.Home), "keys")} {
		entries, err := os.ReadDir(directory)
		if err != nil {
			fmt.Fprintf(options.Output, "discovery note: %s: %v\n", directory, err)
			continue
		}
		for _, entry := range entries {
			path := filepath.Join(directory, entry.Name())
			if entry.IsDir() || managed[path] {
				continue
			}
			file, err := os.Open(path)
			if err != nil {
				fmt.Fprintf(options.Output, "discovery note: %s: %v\n", path, err)
				continue
			}
			prefix := make([]byte, 256)
			count, _ := file.Read(prefix)
			file.Close()
			text := string(prefix[:count])
			if strings.HasSuffix(path, ".pub") || strings.Contains(text, "PRIVATE KEY-----") || strings.HasPrefix(entry.Name(), "id_") {
				fmt.Fprintf(options.Output, "unmanaged key candidate %s\n", path)
			}
		}
	}
	visit := func(filename string, fields []string) error {
		switch strings.ToLower(fields[0]) {
		case "host":
			fmt.Fprintf(options.Output, "unmanaged Host %s (%s; patterns and conditional scopes require review)\n", strings.Join(fields[1:], " "), filename)
		case "match":
			fmt.Fprintf(options.Output, "Match %s (%s): discovery only; Match exec is never executed\n", strings.Join(fields[1:], " "), filename)
		case "identityfile":
			fmt.Fprintf(options.Output, "unmanaged IdentityFile %s (%s)\n", strings.Join(fields[1:], " "), filename)
		}
		return nil
	}
	report := func(issue configIssue) error {
		switch issue.kind {
		case "cycle":
			fmt.Fprintf(options.Output, "Include cycle: %s\n", issue.target)
		case "limit":
			fmt.Fprintln(options.Output, "Include discovery limit reached; review configuration manually")
		case "unresolved":
			fmt.Fprintf(options.Output, "Include unresolved token or home: %s\n", issue.target)
		case "unmatched":
			fmt.Fprintf(options.Output, "Include unreadable or unmatched: %s\n", issue.target)
		case "parse":
			fmt.Fprintf(options.Output, "configuration parse note %s: %v\n", issue.target, issue.err)
		case "read":
			fmt.Fprintf(options.Output, "configuration read note %s: %v\n", issue.target, issue.err)
		default:
			fmt.Fprintf(options.Output, "Include unreadable: %s: %v\n", issue.target, issue.err)
		}
		return nil
	}
	walkConfiguration(options.Home, nil, visit, report)
}

func expandPath(home, path string) string {
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
		return filepath.Join(home, path[2:])
	}
	return path
}

func configFields(line string) ([]string, error) {
	var fields []string
	var word strings.Builder
	quoted := false
	for _, character := range line {
		if character == '"' {
			quoted = !quoted
			continue
		}
		if character == '#' && !quoted {
			break
		}
		if unicode.IsSpace(character) && !quoted {
			if word.Len() > 0 {
				fields = append(fields, word.String())
				word.Reset()
			}
		} else {
			word.WriteRune(character)
		}
	}
	if quoted {
		return nil, fmt.Errorf("unterminated quote")
	}
	if word.Len() > 0 {
		fields = append(fields, word.String())
	}
	return fields, nil
}
