package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func cmdView() {
	cfg := loadAndValidate()
	pass, err := ensureUnlocked(cfg, true)
	if err != nil {
		exitCancel(err)
	}

	for {
		entry, err := pickEntry(cfg, pass)
		if err != nil {
			exitCancel(err)
		}
		saveLastEntry(cfg, entry)
		if entryMenu(cfg, entry, pass) {
			return
		}
	}
}

// pickEntry offers the last used entry as a shortcut above the full list.
func pickEntry(cfg Config, pass string) (string, error) {
	entries, err := getEntries(cfg)
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("no entries found")
	}
	last := lastEntry(cfg)
	items := entries
	if last != "" && contains(entries, last) {
		items = append([]string{"↻ " + last}, entries...)
	}
	choice, err := wofiSelect(cfg, items, "keepass")
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(choice, "↻ "), nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func lastEntryPath(cfg Config) string {
	return filepath.Join(cfg.CacheDir, "last")
}

func lastEntry(cfg Config) string {
	data, err := os.ReadFile(lastEntryPath(cfg))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func saveLastEntry(cfg Config, entry string) {
	ensureDir(cfg.CacheDir)
	os.WriteFile(lastEntryPath(cfg), []byte(entry), 0o600)
}

// entryMenu reports whether a terminal action ran. A field copy or a note line
// returns true so kpmenu exits and wofi is not popped back up; escape returns
// false so the caller goes back to the entry list.
func entryMenu(cfg Config, entry, pass string) bool {
	for {
		fields, err := agentFields(cfg, entry)
		if err != nil {
			notify(cfg, err.Error())
			return true
		}

		items := make([]string, 0, len(fields)+3)
		for _, f := range fields {
			items = append(items, fieldLabel(f))
		}
		if !hasField(fields, "TOTP") {
			items = append(items, "TOTP")
		}
		items = append(items, "Edit ▸", "Add entry")

		choice, err := wofiSelect(cfg, items, baseName(entry))
		if err != nil {
			return false
		}

		switch choice {
		case "Edit ▸":
			editMenu(cfg, entry, pass)
		case "Add entry":
			addMenu(cfg, pass)
		default:
			key := strings.SplitN(choice, ": ", 2)[0]
			field := findField(fields, key)
			if useField(cfg, entry, field, pass) {
				return true
			}
		}
	}
}

func hasField(fields []Field, key string) bool {
	for _, f := range fields {
		if strings.EqualFold(f.Key, key) {
			return true
		}
	}
	return false
}

func findField(fields []Field, key string) Field {
	for _, f := range fields {
		if strings.EqualFold(f.Key, key) {
			return f
		}
	}
	return Field{Key: key}
}

func fieldLabel(f Field) string {
	if strings.EqualFold(f.Key, "Password") {
		return f.Key + ": " + strings.Repeat("*", len(f.Value))
	}
	return f.Key + ": " + truncate(f.Value)
}

// useField performs the terminal action for a field and reports whether the
// menu should close, so wofi is not popped back up right after a copy.
func useField(cfg Config, entry string, field Field, pass string) bool {
	if strings.EqualFold(field.Key, "Notes") {
		return notesValue(cfg, entry, field.Value, pass)
	}
	if strings.EqualFold(field.Key, "TOTP") && field.Value == "" {
		live, err := cfg.totp(entry, pass)
		if err != nil || live == "" {
			notify(cfg, "no TOTP")
			return false
		}
		field.Value = live
	}
	if field.Value == "" {
		notify(cfg, field.Key+" is empty")
		return false
	}
	copyAndType(cfg, field.Key, field.Value)
	return true
}

// notesValue returns true when a note line was selected, which closes the menu.
func notesValue(cfg Config, entry, notes, pass string) bool {
	if strings.TrimSpace(notes) == "" {
		notify(cfg, "no notes")
		return false
	}
	for {
		lines := strings.Split(notes, "\n")
		items := make([]string, 0, len(lines)+1)
		for i, l := range lines {
			items = append(items, fmt.Sprintf("%d: %s", i+1, l))
		}
		items = append(items, "Edit ▸")

		choice, err := wofiSelect(cfg, items, baseName(entry)+" notes")
		if err != nil {
			return false
		}
		if choice == "Edit ▸" {
			editNotes(cfg, entry, notes, pass)
			agentInvalidate(cfg, entry)
			if fields, err := agentFields(cfg, entry); err == nil {
				notes = findField(fields, "Notes").Value
			}
			continue
		}
		_, line, _ := strings.Cut(choice, ": ")
		if strings.TrimSpace(line) != "" {
			copyAndType(cfg, "note", line)
			return true
		}
		return false
	}
}

func copyAndType(cfg Config, label, value string) {
	if err := copyClipboard(cfg, value); err != nil {
		notify(cfg, "clipboard failed")
		return
	}
	clearClipboardLater(cfg, value, cfg.ClipboardClear)
	autotype(cfg, "", value)
	notify(cfg, label+" copied and typed")
}

func editMenu(cfg Config, entry, pass string) {
	fields, err := agentFields(cfg, entry)
	if err != nil {
		notify(cfg, err.Error())
		return
	}
	editable := []string{}
	for _, f := range fields {
		switch strings.ToLower(f.Key) {
		case "title", "username", "password", "url", "notes":
			editable = append(editable, f.Key)
		}
	}
	editable = append(editable, "Delete entry")

	choice, err := wofiSelect(cfg, editable, "edit "+baseName(entry))
	if err != nil {
		return
	}

	switch choice {
	case "Delete entry":
		confirm, err := wofiSelect(cfg, []string{"Cancel", "Delete " + baseName(entry)}, "confirm delete")
		if err != nil || !strings.HasPrefix(confirm, "Delete") {
			return
		}
		if err := cfg.deleteEntry(entry, pass); err != nil {
			notify(cfg, err.Error())
			return
		}
		refreshCache(cfg, pass)
		notify(cfg, "deleted "+baseName(entry))
		os.Exit(0)
	case "Notes":
		notes := findField(fields, "Notes").Value
		editNotes(cfg, entry, notes, pass)
		agentInvalidate(cfg, entry)
	case "Password":
		editPassword(cfg, entry, pass)
		agentInvalidate(cfg, entry)
	default:
		current := findField(fields, choice).Value
		newVal, err := wofiInput(cfg, "new "+strings.ToLower(choice), current)
		if err != nil {
			return
		}
		if err := cfg.editField(entry, choice, newVal, pass); err != nil {
			notify(cfg, err.Error())
			return
		}
		agentInvalidate(cfg, entry)
		notify(cfg, choice+" updated")
	}
}

func editPassword(cfg Config, entry, pass string) {
	method, err := wofiSelect(cfg, []string{"Type new password", "Generate password"}, "password")
	if err != nil {
		return
	}
	var newPw string
	if method == "Generate password" {
		newPw, err = cfg.generatePassword(20)
		if err != nil {
			notify(cfg, err.Error())
			return
		}
		copyClipboard(cfg, newPw)
		clearClipboardLater(cfg, newPw, cfg.ClipboardClear)
	} else {
		newPw, err = wofiPassword(cfg, "new password")
		if err != nil {
			return
		}
	}
	if err := cfg.editPassword(entry, newPw, pass); err != nil {
		notify(cfg, err.Error())
		return
	}
	notify(cfg, "password updated")
}

func editNotes(cfg Config, entry, notes, pass string) {
	tmp, err := os.CreateTemp("", "kpmenu-notes-*.txt")
	if err != nil {
		notify(cfg, err.Error())
		return
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	tmp.WriteString(notes)
	tmp.Close()

	editor := cfg.Editor
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "nvim"
	}

	var cmd *exec.Cmd
	if cfg.Terminal != "" {
		cmd = exec.Command(cfg.Terminal, "-e", editor, tmpPath)
	} else {
		cmd = exec.Command(editor, tmpPath)
	}
	if err := cmd.Run(); err != nil {
		notify(cfg, "editor failed")
		return
	}

	edited, err := os.ReadFile(tmpPath)
	if err != nil {
		notify(cfg, err.Error())
		return
	}
	if err := cfg.setNotes(entry, string(edited), pass); err != nil {
		notify(cfg, err.Error())
		return
	}
	notify(cfg, "notes updated")
}

func addMenu(cfg Config, pass string) {
	groups, err := cfg.listGroups(pass)
	if err != nil {
		notify(cfg, err.Error())
		return
	}
	parent, err := wofiSelect(cfg, groups, "group for new entry")
	if err != nil {
		return
	}
	title, err := wofiInput(cfg, "title", "")
	if err != nil || title == "" {
		return
	}
	username, _ := wofiInput(cfg, "username", "")
	url, _ := wofiInput(cfg, "url", "")
	notes, _ := wofiInput(cfg, "notes", "")

	method, err := wofiSelect(cfg, []string{"Generate password", "Enter password"}, "password")
	if err != nil {
		return
	}
	generate := method == "Generate password"
	password := ""
	if !generate {
		password, err = wofiPassword(cfg, "entry password")
		if err != nil {
			return
		}
	}

	if err := cfg.addEntry(parent, title, username, url, notes, password, pass, generate); err != nil {
		notify(cfg, err.Error())
		return
	}
	refreshCache(cfg, pass)
	notify(cfg, "added "+title)
}

func baseName(entry string) string {
	if i := strings.LastIndex(entry, "/"); i >= 0 {
		return entry[i+1:]
	}
	return entry
}

func truncate(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 50 {
		return s[:47] + "..."
	}
	return s
}

func exitCancel(err error) {
	if err == errCancelled {
		os.Exit(1)
	}
	fatal(err)
}
