package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func cmdView() {
	cfg := loadAndValidate()
	pass, err := ensureUnlocked(cfg, true)
	if err != nil {
		exitCancel(err)
	}

	entry, err := pickEntry(cfg, "keepass")
	if err != nil {
		exitCancel(err)
	}
	entryMenu(cfg, entry, pass)
}

func pickEntry(cfg Config, prompt string) (string, error) {
	entries, err := getEntries(cfg)
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("no entries found")
	}
	return wofiSelect(cfg, entries, prompt)
}

func entryMenu(cfg Config, entry, pass string) {
	for {
		fields, err := cfg.showAll(entry, pass)
		if err != nil {
			notify(cfg, err.Error())
			return
		}

		items := make([]string, 0, len(fields)+3)
		for _, f := range fields {
			items = append(items, fieldLabel(f))
		}
		items = append(items, "Edit ▸", "Add entry", "Back")

		choice, err := wofiSelect(cfg, items, baseName(entry))
		if err != nil {
			return
		}

		switch choice {
		case "Back":
			return
		case "Edit ▸":
			editMenu(cfg, entry, pass)
		case "Add entry":
			addMenu(cfg, pass)
		default:
			key := strings.SplitN(choice, ": ", 2)[0]
			useField(cfg, entry, key, pass)
		}
	}
}

func fieldLabel(f Field) string {
	if strings.EqualFold(f.Key, "Password") {
		return f.Key + ": " + strings.Repeat("*", len(f.Value))
	}
	return f.Key + ": " + truncate(f.Value)
}

func useField(cfg Config, entry, key, pass string) {
	if strings.EqualFold(key, "Notes") {
		notesValue(cfg, entry, pass)
		return
	}
	value, err := cfg.fieldValue(entry, key, pass)
	if err != nil || value == "" {
		notify(cfg, key+" is empty")
		return
	}
	copyAndType(cfg, key, value)
}

func notesValue(cfg Config, entry, pass string) {
	notes, err := cfg.showField(entry, "Notes", pass)
	if err != nil || strings.TrimSpace(notes) == "" {
		notify(cfg, "no notes")
		return
	}
	lines := strings.Split(notes, "\n")
	items := make([]string, 0, len(lines)+2)
	for i, l := range lines {
		items = append(items, fmt.Sprintf("%d: %s", i+1, l))
	}
	items = append(items, "Edit ▸", "Back")

	choice, err := wofiSelect(cfg, items, baseName(entry)+" notes")
	if err != nil {
		return
	}
	if choice == "Back" {
		return
	}
	if choice == "Edit ▸" {
		editNotes(cfg, entry, notes, pass)
		return
	}
	_, line, _ := strings.Cut(choice, ": ")
	if strings.TrimSpace(line) != "" {
		copyAndType(cfg, "note", line)
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
	fields, err := cfg.showAll(entry, pass)
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
	editable = append(editable, "Delete entry", "Back")

	choice, err := wofiSelect(cfg, editable, "edit "+baseName(entry))
	if err != nil || choice == "Back" {
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
		notes, _ := cfg.showField(entry, "Notes", pass)
		editNotes(cfg, entry, notes, pass)
	case "Password":
		editPassword(cfg, entry, pass)
	default:
		current, _ := cfg.showField(entry, choice, pass)
		newVal, err := wofiInput(cfg, "new "+strings.ToLower(choice), current)
		if err != nil {
			return
		}
		if err := cfg.editField(entry, choice, newVal, pass); err != nil {
			notify(cfg, err.Error())
			return
		}
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
