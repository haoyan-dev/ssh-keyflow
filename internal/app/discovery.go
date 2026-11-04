package app

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"ssh-keyflow/internal/state"
)

func list(options Options, document state.State) {
	for _, key := range document.Keys {
		fmt.Fprintf(options.Output, "managed key %s [%s] %s %s connection=%s origin=%s imported_at=%s\n", key.Name, key.ID, key.PublicPath, key.Fingerprint, key.ConnectionCheck, key.Origin, key.ImportedAt)
	}
	for _, host := range document.Hosts {
		fmt.Fprintf(options.Output, "managed host %s [%s] %s@%s:%d\n", host.Alias, host.ID, host.User, host.Hostname, host.Port)
	}
	for _, registration := range document.Registrations {
		fmt.Fprintf(options.Output, "registration %s key=%s host=%s status=%s connection=%s operation=%s stage=%s (recorded history)\n", registration.ID, registration.KeyID, registration.HostID, registration.Status, registration.ConnectionCheck, registration.Operation, registration.Stage)
	}
	managed := map[string]bool{}
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
	active := map[string]bool{}
	visited := 0
	var discover func(string, int)
	discover = func(path string, depth int) {
		canonical, err := filepath.EvalSymlinks(path)
		if err != nil {
			fmt.Fprintf(options.Output, "Include unreadable: %s: %v\n", path, err)
			return
		}
		if active[canonical] {
			fmt.Fprintf(options.Output, "Include cycle: %s\n", path)
			return
		}
		if depth > 64 || visited >= 4096 {
			fmt.Fprintln(options.Output, "Include discovery limit reached; review configuration manually")
			return
		}
		visited++
		active[canonical] = true
		defer delete(active, canonical)
		file, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(options.Output, "Include unreadable: %s: %v\n", path, err)
			return
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 4096), 1024*1024)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if position := strings.IndexAny(line, "= \t"); position >= 0 {
				line = line[:position] + " " + strings.TrimLeft(line[position:], "= \t")
			}
			fields, err := configFields(line)
			if err != nil {
				fmt.Fprintf(options.Output, "configuration parse note %s: %v\n", path, err)
				continue
			}
			if len(fields) == 0 {
				continue
			}
			switch strings.ToLower(fields[0]) {
			case "host":
				fmt.Fprintf(options.Output, "unmanaged Host %s (%s; patterns and conditional scopes require review)\n", strings.Join(fields[1:], " "), path)
			case "match":
				fmt.Fprintf(options.Output, "Match %s (%s): discovery only; Match exec is never executed\n", strings.Join(fields[1:], " "), path)
			case "identityfile":
				fmt.Fprintf(options.Output, "unmanaged IdentityFile %s (%s)\n", strings.Join(fields[1:], " "), path)
			case "include":
				for _, pattern := range fields[1:] {
					if strings.ContainsAny(pattern, "%$") || (strings.HasPrefix(pattern, "~") && !strings.HasPrefix(pattern, "~/") && !strings.HasPrefix(pattern, "~\\")) {
						fmt.Fprintf(options.Output, "Include unresolved token or home: %s\n", pattern)
						continue
					}
					pattern = expandPath(options.Home, pattern)
					if !filepath.IsAbs(pattern) {
						pattern = filepath.Join(root, pattern)
					}
					matches, err := filepath.Glob(pattern)
					if err != nil || len(matches) == 0 {
						fmt.Fprintf(options.Output, "Include unreadable or unmatched: %s\n", pattern)
						continue
					}
					for _, match := range matches {
						discover(match, depth+1)
					}
				}
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(options.Output, "configuration read note %s: %v\n", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "config")); !os.IsNotExist(err) {
		discover(filepath.Join(root, "config"), 0)
	}
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
