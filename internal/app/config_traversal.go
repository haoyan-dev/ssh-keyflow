package app

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type configIssue struct {
	kind   string
	target string
	err    error
}

func walkConfiguration(home string, skip func(string) bool, visit func(string, []string) error, report func(configIssue) error) error {
	root := filepath.Join(home, ".ssh")
	active := map[string]bool{}
	visited := 0
	var walk func(string, int) error
	walk = func(filename string, depth int) error {
		canonical, err := filepath.EvalSymlinks(filename)
		if err != nil {
			return report(configIssue{"unreadable", filename, err})
		}
		if skip != nil && skip(canonical) {
			return nil
		}
		identity := canonical
		if runtime.GOOS == "windows" {
			identity = strings.ToLower(identity)
		}
		if active[identity] {
			return report(configIssue{"cycle", filename, nil})
		}
		if depth > 64 || visited >= 4096 {
			return report(configIssue{"limit", filename, nil})
		}
		visited++
		active[identity] = true
		defer delete(active, identity)
		file, err := os.Open(filename)
		if err != nil {
			return report(configIssue{"unreadable", filename, err})
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
				if err := report(configIssue{"parse", filename, err}); err != nil {
					return err
				}
				continue
			}
			if len(fields) == 0 {
				continue
			}
			if !strings.EqualFold(fields[0], "include") {
				if err := visit(filename, fields); err != nil {
					return err
				}
				continue
			}
			for _, pattern := range fields[1:] {
				if strings.ContainsAny(pattern, "%$") || (strings.HasPrefix(pattern, "~") && !strings.HasPrefix(pattern, "~/") && !strings.HasPrefix(pattern, "~\\")) {
					if err := report(configIssue{"unresolved", pattern, nil}); err != nil {
						return err
					}
					continue
				}
				pattern = expandPath(home, pattern)
				if !filepath.IsAbs(pattern) {
					pattern = filepath.Join(root, pattern)
				}
				matches, err := filepath.Glob(pattern)
				if err != nil || len(matches) == 0 {
					if err := report(configIssue{"unmatched", pattern, err}); err != nil {
						return err
					}
					continue
				}
				for _, match := range matches {
					if err := walk(match, depth+1); err != nil {
						return err
					}
				}
			}
		}
		if err := scanner.Err(); err != nil {
			return report(configIssue{"read", filename, err})
		}
		return nil
	}
	filename := filepath.Join(root, "config")
	if _, err := os.Lstat(filename); os.IsNotExist(err) {
		return nil
	}
	return walk(filename, 0)
}
