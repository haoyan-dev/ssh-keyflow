package app

import (
	"fmt"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

func checkAlias(options Options, alias string) error {
	return checkAliasExcept(options, alias, "")
}

func checkAliasExcept(options Options, alias, ownedPath string) error {
	return inspectAliases(options, alias, ownedPath, nil)
}

func inspectAliases(options Options, alias, ownedPath string, included func()) error {
	if ownedPath != "" {
		canonical, err := filepath.EvalSymlinks(ownedPath)
		if err != nil {
			return fmt.Errorf("requires review: cannot resolve owned configuration: %w", err)
		}
		ownedPath = canonical
	}
	skip := func(canonical string) bool {
		same := filepath.Clean(canonical) == filepath.Clean(ownedPath)
		if runtime.GOOS == "windows" {
			same = strings.EqualFold(filepath.Clean(canonical), filepath.Clean(ownedPath))
		}
		if ownedPath == "" || !same {
			return false
		}
		if included != nil {
			included()
		}
		return true
	}
	visit := func(filename string, fields []string) error {
		if strings.EqualFold(fields[0], "host") {
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
		}
		return nil
	}
	report := func(issue configIssue) error {
		switch issue.kind {
		case "cycle":
			return fmt.Errorf("requires review: Include cycle at %s", issue.target)
		case "limit":
			return fmt.Errorf("requires review: configuration discovery limit")
		case "unresolved":
			return fmt.Errorf("requires review: unresolved Include %s", issue.target)
		case "unmatched":
			return fmt.Errorf("requires review: unreadable or unmatched Include %s", issue.target)
		default:
			return fmt.Errorf("requires review: cannot inspect existing aliases: %s: %w", issue.target, issue.err)
		}
	}
	return walkConfiguration(options.Home, skip, visit, report)
}
