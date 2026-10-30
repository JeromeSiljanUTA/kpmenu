package main

import (
	"fmt"
	"os"
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
		label := baseName(entry)

		fields := []string{}
		for _, f := range []string{"UserName", "Password", "URL", "Notes", "TOTP"} {
			if v, err := cfg.fieldValue(entry, f, pass); err == nil && v != "" {
				display := v
				if f == "Password" {
					display = strings.Repeat("*", len(v))
				}
				fields = append(fields, fmt.Sprintf("%s: %s", f, truncate(display)))
			}
		}
		fields = append(fields, "Edit ▸", "Add entry", "Back")

		choice, err := wofiSelect(cfg, fields, label)
		if err != nil {
			return
		}

		switch {
		case choice == "Back":
			return
		case choice == "Edit ▸":
			editMenu(cfg, entry, pass)
		case choice == "Add entry":
			addMenu(cfg, pass)
		default:
			field := strings.SplitN(choice, ":", 2)[0]
			useField(cfg, entry, field, pass)
		}
	}
}

func useField(cfg Config, entry, field, pass string) {
	value, err := cfg.fieldValue(entry, field, pass)
	if err != nil || value == "" {
		notify(cfg, field+" is empty")
		return
	}
	if err := copyClipboard(cfg, value); err != nil {
		notify(cfg, "clipboard failed")
		return
	}
	clearClipboardLater(cfg, value, cfg.ClipboardClear)
	autotype(cfg, "", value)
	notify(cfg, field+" copied and typed")
}

func editMenu(cfg Config, entry, pass string) {
	choices := []string{"UserName", "Password", "URL", "Notes", "Title", "Delete entry"}
	choice, err := wofiSelect(cfg, choices, "edit "+baseName(entry))
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
	case "Password":
		newPw, err := wofiPassword(cfg, "new password")
		if err != nil {
			return
		}
		if err := cfg.editPassword(entry, newPw, pass); err != nil {
			notify(cfg, err.Error())
			return
		}
		notify(cfg, "password updated")
	case "Title":
		newTitle, err := wofiInput(cfg, "new title", baseName(entry))
		if err != nil {
			return
		}
		if err := cfg.editField(entry, "Title", newTitle, pass); err != nil {
			notify(cfg, err.Error())
			return
		}
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

	genChoice, _ := wofiSelect(cfg, []string{"Generate password", "Enter password"}, "password")
	generate := genChoice == "Generate password"
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
	if err := refreshCache(cfg, pass); err != nil {
		_ = err
	}
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
	if len(s) > 40 {
		return s[:37] + "..."
	}
	return s
}

func exitCancel(err error) {
	if err == errCancelled {
		os.Exit(1)
	}
	fatal(err)
}
