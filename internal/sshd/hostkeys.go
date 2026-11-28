// Package sshd reads this machine's sshd host public keys without changing
// anything. It never opens a private host key.
package sshd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"ssh-keyflow/internal/roster"
)

// HostKey is one host public key and the file it came from.
type HostKey struct {
	Path string
	Key  roster.PublicKey
}

// Directory is sshd's configuration directory on this platform.
func Directory() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(programData(), "ssh")
	}
	return "/etc/ssh"
}

func programData() string {
	if value := os.Getenv("ProgramData"); value != "" {
		return value
	}
	return `C:\ProgramData`
}

// ReadHostKeys reads the public halves of the HostKey entries in config,
// following Include. Relative Include paths resolve against dir, as sshd
// does. Without any HostKey entry it reads sshd's defaults in dir. A
// missing config is allowed only when required is false. Entries whose
// public file is absent are reported in skipped; at least one key is needed.
func ReadHostKeys(config, dir string, required bool) (keys []HostKey, skipped []string, err error) {
	var paths []string
	found, err := scan(config, dir, required, &paths, 0)
	if err != nil {
		return nil, nil, err
	}
	if !found {
		for _, kind := range []string{"rsa", "ecdsa", "ed25519"} {
			paths = append(paths, filepath.Join(dir, "ssh_host_"+kind+"_key"))
		}
	}
	seen := map[string]bool{}
	for _, path := range paths {
		public := path
		if !strings.HasSuffix(public, ".pub") {
			public += ".pub"
		}
		data, err := os.ReadFile(public)
		if os.IsNotExist(err) {
			skipped = append(skipped, public)
			continue
		}
		if err != nil {
			return nil, nil, fmt.Errorf("read host public key: %w", err)
		}
		key, err := roster.ParsePublicKey(string(data), true)
		if err != nil {
			return nil, nil, fmt.Errorf("host public key %s: %w", public, err)
		}
		if !seen[key.Text] {
			seen[key.Text] = true
			keys = append(keys, HostKey{Path: public, Key: key})
		}
	}
	if len(keys) == 0 {
		return nil, skipped, fmt.Errorf("no sshd host public key found (looked for %s); is sshd installed? pass --sshd-config if it uses another file", strings.Join(skipped, ", "))
	}
	if len(keys) > roster.MaxKeys {
		return nil, nil, fmt.Errorf("sshd has %d host keys; at most %d are supported", len(keys), roster.MaxKeys)
	}
	return keys, skipped, nil
}

const maxIncludeDepth = 16

// scan appends HostKey paths from one config file. It reports whether any
// HostKey entry was seen. Lines after a Match apply to that block, where
// sshd does not allow HostKey, so they are ignored.
func scan(config, dir string, required bool, paths *[]string, depth int) (bool, error) {
	if depth > maxIncludeDepth {
		return false, fmt.Errorf("sshd_config Include nesting is too deep at %s", config)
	}
	file, err := os.Open(config)
	if os.IsNotExist(err) && !required {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read sshd config: %w", err)
	}
	defer file.Close()
	found := false
	lines := bufio.NewScanner(file)
	for lines.Scan() {
		keyword, arguments, err := directive(lines.Text())
		if err != nil {
			return false, fmt.Errorf("%s: %w", config, err)
		}
		switch keyword {
		case "match":
			return found, nil
		case "hostkey":
			if len(arguments) != 1 {
				return false, fmt.Errorf("%s: HostKey needs exactly one path", config)
			}
			path, err := hostKeyPath(arguments[0])
			if err != nil {
				return false, fmt.Errorf("%s: %w", config, err)
			}
			*paths = append(*paths, path)
			found = true
		case "include":
			for _, pattern := range arguments {
				if strings.HasPrefix(pattern, "~") {
					return false, fmt.Errorf("%s: Include %q: ~ is not supported", config, pattern)
				}
				if !filepath.IsAbs(pattern) {
					pattern = filepath.Join(dir, pattern)
				}
				matches, err := filepath.Glob(pattern)
				if err != nil {
					return false, fmt.Errorf("%s: Include %q: %w", config, pattern, err)
				}
				for _, match := range matches {
					included, err := scan(match, dir, true, paths, depth+1)
					if err != nil {
						return false, err
					}
					found = found || included
				}
			}
		}
	}
	if err := lines.Err(); err != nil {
		return false, fmt.Errorf("read sshd config %s: %w", config, err)
	}
	return found, nil
}

func hostKeyPath(value string) (string, error) {
	if runtime.GOOS == "windows" {
		value = strings.ReplaceAll(value, "__PROGRAMDATA__", programData())
	}
	if strings.Contains(value, "%") {
		return "", fmt.Errorf("HostKey %q: %% tokens are not supported", value)
	}
	if !filepath.IsAbs(value) {
		return "", fmt.Errorf("HostKey %q must be an absolute path", value)
	}
	return filepath.Clean(value), nil
}

// directive splits one sshd_config line into a lowercase keyword and its
// arguments. "Keyword=value" and double-quoted arguments are accepted.
func directive(line string) (string, []string, error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", nil, nil
	}
	end := strings.IndexAny(line, " \t=")
	if end < 0 {
		return strings.ToLower(line), nil, nil
	}
	keyword := strings.ToLower(line[:end])
	rest := strings.TrimLeft(line[end:], " \t")
	rest = strings.TrimLeft(strings.TrimPrefix(rest, "="), " \t")
	var arguments []string
	for rest != "" {
		if strings.HasPrefix(rest, "#") {
			break
		}
		var argument string
		if rest[0] == '"' {
			closing := strings.IndexByte(rest[1:], '"')
			if closing < 0 {
				return "", nil, fmt.Errorf("unterminated quote in %q", line)
			}
			argument, rest = rest[1:1+closing], rest[2+closing:]
		} else {
			stop := strings.IndexAny(rest, " \t")
			if stop < 0 {
				stop = len(rest)
			}
			argument, rest = rest[:stop], rest[stop:]
		}
		arguments = append(arguments, argument)
		rest = strings.TrimLeft(rest, " \t")
	}
	return keyword, arguments, nil
}
