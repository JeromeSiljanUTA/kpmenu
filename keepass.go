package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// listEntries returns every entry path in the database, newline separated.
func (c Config) listEntries(passphrase string) ([]string, error) {
	args := []string{"ls", "-q", "-R", "-f", c.Database}
	cmd := exec.Command(c.KeepassxcCli, args...)
	cmd.Stdin = strings.NewReader(passphrase + "\n")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("keepassxc-cli ls: %w", err)
	}
	var entries []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimRight(line, "\r")
		line = strings.TrimSpace(line)
		if line == "" || line == "[empty]" || strings.HasSuffix(line, "/") {
			continue
		}
		entries = append(entries, line)
	}
	return entries, nil
}

func (c Config) showField(entry, field, passphrase string) (string, error) {
	args := []string{"show", "-q", "-a", field}
	if c.KeyFile != "" {
		args = append(args, "-k", c.KeyFile)
	}
	args = append(args, c.Database, entry)
	cmd := exec.Command(c.KeepassxcCli, args...)
	cmd.Stdin = strings.NewReader(passphrase + "\n")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("keepassxc-cli show %s: %w", field, err)
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

func (c Config) touchCache() {
	if fi, err := os.Stat(c.Database); err == nil {
		_ = os.Chtimes(c.cacheFile(), fi.ModTime(), fi.ModTime())
	}
}

func (c Config) cacheStale() bool {
	cfi, err := os.Stat(c.cacheFile())
	if err != nil {
		return true
	}
	dfi, err := os.Stat(c.Database)
	if err != nil {
		return false
	}
	return dfi.ModTime().After(cfi.ModTime())
}

func ensureDir(p string) error {
	return os.MkdirAll(p, 0o700)
}
