package main

import (
	"fmt"
	"os"
	"strings"
)

func readCache(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func writeCache(path string, entries []string) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(strings.Join(entries, "\n")+"\n"), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func refreshCache(cfg Config, passphrase string) error {
	entries, err := cfg.listEntries(passphrase)
	if err != nil {
		return fmt.Errorf("refresh: %w", err)
	}
	if err := ensureDir(cfg.CacheDir); err != nil {
		return err
	}
	return writeCache(cfg.cacheFile(), entries)
}
