package app

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func checkAlias(options Options, alias string) error {
	return checkAliasExcept(options, alias, "")
}

func checkAliasExcept(options Options, alias, ownedPath string) error {
	return inspectAliases(options, alias, ownedPath, nil)
}

func inspectAliases(options Options, alias, ownedPath string, included func()) error {
	home := options.Home
	root := filepath.Join(home, ".ssh")
	active := map[string]bool{}
	visited := 0
	var check func(string, int) error
	check = func(filename string, depth int) error {
		canonical, err := filepath.EvalSymlinks(filename)
		if err != nil {
			return fmt.Errorf("requires review: cannot inspect existing aliases: %w", err)
		}
		if ownedPath != "" && filepath.Clean(canonical) == filepath.Clean(ownedPath) {
			if included != nil {
				included()
			}
			return nil
		}
		if active[canonical] {
			return fmt.Errorf("requires review: Include cycle at %s", filename)
		}
		if depth > 64 || visited >= 4096 {
			return fmt.Errorf("requires review: configuration discovery limit")
		}
		visited++
		active[canonical] = true
		defer delete(active, canonical)
		file, err := os.Open(filename)
		if err != nil {
			return err
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
				return fmt.Errorf("requires review: %s: %w", filename, err)
			}
			if len(fields) == 0 {
				continue
			}
			switch strings.ToLower(fields[0]) {
			case "host":
				matched := false
				literal := false
				for _, pattern := range fields[1:] {
					negative := strings.HasPrefix(pattern, "!")
					pattern = strings.TrimPrefix(pattern, "!")
					matches, err := path.Match(strings.ToLower(pattern), strings.ToLower(alias))
					if err != nil {
						return fmt.Errorf("requires review: existing Host pattern %s", pattern)
					}
					if matches && negative {
						matched = false
						break
					}
					if matches {
						matched = true
					}
					if matches && !negative && !strings.ContainsAny(pattern, "*?") {
						literal = true
					}
				}
				if matched && literal {
					return fmt.Errorf("requires review: alias %s matches existing Host %s in %s; choose another alias", alias, strings.Join(fields[1:], " "), filename)
				}
				if matched {
					fmt.Fprintf(options.Output, "Wildcard Host %s in %s affects %s: review inherited options and precedence before using an alias. Manual registration remains available; no Include or Host configuration is written.\n", strings.Join(fields[1:], " "), filename, alias)
				}
			case "include":
				for _, pattern := range fields[1:] {
					if strings.ContainsAny(pattern, "%$") || (strings.HasPrefix(pattern, "~") && !strings.HasPrefix(pattern, "~/") && !strings.HasPrefix(pattern, "~\\")) {
						return fmt.Errorf("requires review: unresolved Include %s", pattern)
					}
					pattern = expandPath(home, pattern)
					if !filepath.IsAbs(pattern) {
						pattern = filepath.Join(root, pattern)
					}
					matches, err := filepath.Glob(pattern)
					if err != nil || len(matches) == 0 {
						return fmt.Errorf("requires review: unreadable or unmatched Include %s", pattern)
					}
					for _, match := range matches {
						if err := check(match, depth+1); err != nil {
							return err
						}
					}
				}
			}
		}
		return scanner.Err()
	}
	filename := filepath.Join(root, "config")
	if _, err := os.Lstat(filename); os.IsNotExist(err) {
		return nil
	}
	return check(filename, 0)
}
