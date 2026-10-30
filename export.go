package main

import (
	"encoding/xml"
	"fmt"
	"os/exec"
	"strings"
)

type xmlKeePassFile struct {
	Root xmlRoot `xml:"Root"`
}

type xmlRoot struct {
	Groups []xmlGroup `xml:"Group"`
}

type xmlGroup struct {
	Name    string     `xml:"Name"`
	Groups  []xmlGroup `xml:"Group"`
	Entries []xmlEntry `xml:"Entry"`
}

type xmlEntry struct {
	Strings []xmlString `xml:"String"`
}

type xmlString struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

// exportAll decrypts the database once and returns every entry keyed by its
// "Group/Sub/Title" path, with all fields including notes, custom attributes
// and cleartext protected values. One decrypt replaces a per-entry decrypt.
func (c Config) exportAll(passphrase string) (map[string][]Field, error) {
	args := []string{"export", "-q"}
	if c.KeyFile != "" {
		args = append(args, "-k", c.KeyFile)
	}
	args = append(args, c.Database)
	cmd := exec.Command(c.KeepassxcCli, args...)
	cmd.Stdin = strings.NewReader(passphrase + "\n")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("keepassxc-cli export: %w", err)
	}

	var doc xmlKeePassFile
	if err := xml.Unmarshal(out, &doc); err != nil {
		return nil, fmt.Errorf("parse export: %w", err)
	}

	entries := map[string][]Field{}
	var walk func(g xmlGroup, path string, root bool)
	walk = func(g xmlGroup, path string, root bool) {
		full := path
		// The database's top group name is not part of entry paths reported by
		// keepassxc-cli ls, so it is skipped when building keys.
		if !root && g.Name != "" {
			full = path + "/" + g.Name
		}
		full = strings.TrimPrefix(full, "/")
		for _, e := range g.Entries {
			key := strings.TrimPrefix(full+"/"+entryTitle(e), "/")
			entries[key] = entryFields(e)
		}
		for _, sub := range g.Groups {
			walk(sub, full, false)
		}
	}
	for _, g := range doc.Root.Groups {
		walk(g, "", true)
	}
	return entries, nil
}

func entryTitle(e xmlEntry) string {
	for _, s := range e.Strings {
		if s.Key == "Title" {
			return s.Value
		}
	}
	return ""
}

// entryFields returns fields in a stable display order, standard fields first.
func entryFields(e xmlEntry) []Field {
	order := []string{"Title", "UserName", "Password", "URL", "Notes"}
	byKey := map[string]string{}
	for _, s := range e.Strings {
		byKey[s.Key] = s.Value
	}
	var fields []Field
	seen := map[string]bool{}
	for _, k := range order {
		if v, ok := byKey[k]; ok {
			fields = append(fields, Field{Key: k, Value: v})
			seen[k] = true
		}
	}
	for _, s := range e.Strings {
		if seen[s.Key] || s.Key == "Title" {
			continue
		}
		fields = append(fields, Field{Key: s.Key, Value: s.Value})
	}
	return fields
}
