package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
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
		if c.hiddenEntry(line) {
			continue
		}
		entries = append(entries, line)
	}
	return entries, nil
}

func (c Config) hiddenEntry(entry string) bool {
	for _, g := range c.HideGroups {
		g = strings.TrimSuffix(strings.TrimSpace(g), "/")
		if g == "" {
			continue
		}
		if entry == g || strings.HasPrefix(entry, g+"/") {
			return true
		}
	}
	return false
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

func (c Config) totp(entry, passphrase string) (string, error) {
	args := []string{"show", "-q", "-t"}
	if c.KeyFile != "" {
		args = append(args, "-k", c.KeyFile)
	}
	args = append(args, c.Database, entry)
	cmd := exec.Command(c.KeepassxcCli, args...)
	cmd.Stdin = strings.NewReader(passphrase + "\n")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("no TOTP")
	}
	return strings.TrimSpace(string(out)), nil
}

func (c Config) fieldValue(entry, field, passphrase string) (string, error) {
	if field == "TOTP" {
		return c.totp(entry, passphrase)
	}
	return c.showField(entry, field, passphrase)
}

// Field is a labeled value from show --all, in display order.
type Field struct {
	Key   string
	Value string
}

// showAll returns every field keepassxc-cli knows about, including custom
// attributes. Password comes back as PROTECTED, so it is fetched separately.
func (c Config) showAll(entry, passphrase string) ([]Field, error) {
	args := []string{"show", "-q", "--all"}
	if c.KeyFile != "" {
		args = append(args, "-k", c.KeyFile)
	}
	args = append(args, c.Database, entry)
	cmd := exec.Command(c.KeepassxcCli, args...)
	cmd.Stdin = strings.NewReader(passphrase + "\n")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("keepassxc-cli show --all: %w", err)
	}
	var fields []Field
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimRight(line, "\r")
		k, v, ok := strings.Cut(line, ": ")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		if strings.EqualFold(k, "uuid") || strings.EqualFold(k, "tags") {
			continue
		}
		fields = append(fields, Field{Key: k, Value: v})
	}
	return fields, nil
}

func (c Config) generatePassword(length int) (string, error) {
	args := []string{"generate", "-L", strconv.Itoa(length)}
	out, err := exec.Command(c.KeepassxcCli, args...).Output()
	if err != nil {
		return "", fmt.Errorf("keepassxc-cli generate: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// editField maps a display field name to its keepassxc-cli edit flag.
func (c Config) editField(entry, field, value, passphrase string) error {
	flag := map[string]string{
		"UserName": "-u",
		"URL":      "--url",
		"Notes":    "--notes",
		"Title":    "-t",
	}[field]
	if flag == "" {
		return fmt.Errorf("field %s is not editable", field)
	}
	args := []string{"edit", "-q", flag, value}
	if c.KeyFile != "" {
		args = append(args, "-k", c.KeyFile)
	}
	args = append(args, c.Database, entry)
	cmd := exec.Command(c.KeepassxcCli, args...)
	cmd.Stdin = strings.NewReader(passphrase + "\n")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("edit %s: %s", field, strings.TrimSpace(string(out)))
	}
	return nil
}

func (c Config) setNotes(entry, notes, passphrase string) error {
	return c.editField(entry, "Notes", notes, passphrase)
}

func (c Config) editPassword(entry, newPassword, passphrase string) error {
	args := []string{"edit", "-q", "-p"}
	if c.KeyFile != "" {
		args = append(args, "-k", c.KeyFile)
	}
	args = append(args, c.Database, entry)
	cmd := exec.Command(c.KeepassxcCli, args...)
	cmd.Stdin = strings.NewReader(passphrase + "\n" + newPassword + "\n")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("edit password: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func (c Config) addEntry(parent, title, username, url, notes, password, passphrase string, generate bool) error {
	entryPath := title
	if parent != "" && parent != "/" {
		entryPath = strings.TrimSuffix(parent, "/") + "/" + title
	}
	args := []string{"add", "-q"}
	if username != "" {
		args = append(args, "-u", username)
	}
	if url != "" {
		args = append(args, "--url", url)
	}
	if notes != "" {
		args = append(args, "--notes", notes)
	}
	if generate {
		args = append(args, "-g", "-L", "20")
	} else {
		args = append(args, "-p")
	}
	if c.KeyFile != "" {
		args = append(args, "-k", c.KeyFile)
	}
	args = append(args, c.Database, entryPath)
	cmd := exec.Command(c.KeepassxcCli, args...)
	stdin := passphrase + "\n"
	if !generate {
		stdin += password + "\n"
	}
	cmd.Stdin = strings.NewReader(stdin)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("add entry: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func (c Config) deleteEntry(entry, passphrase string) error {
	args := []string{"rm", "-q"}
	if c.KeyFile != "" {
		args = append(args, "-k", c.KeyFile)
	}
	args = append(args, c.Database, entry)
	cmd := exec.Command(c.KeepassxcCli, args...)
	cmd.Stdin = strings.NewReader(passphrase + "\n")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("delete: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func (c Config) listGroups(passphrase string) ([]string, error) {
	args := []string{"ls", "-q", "-R", "-f"}
	if c.KeyFile != "" {
		args = append(args, "-k", c.KeyFile)
	}
	args = append(args, c.Database)
	cmd := exec.Command(c.KeepassxcCli, args...)
	cmd.Stdin = strings.NewReader(passphrase + "\n")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("keepassxc-cli ls: %w", err)
	}
	groups := []string{"/"}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasSuffix(line, "/") {
			groups = append(groups, line)
		}
	}
	return groups, nil
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
