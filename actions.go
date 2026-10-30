package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func cmdBrowse() {
	cfg := loadAndValidate()
	entries, err := getEntries(cfg)
	if err != nil {
		fatal(err)
	}
	if len(entries) == 0 {
		fatal(fmt.Errorf("no entries found"))
	}
	sel, err := wofiSelect(cfg, entries, "keepass")
	if err != nil {
		if err == errCancelled {
			os.Exit(1)
		}
		fatal(err)
	}
	runAction(cfg, sel, false)
}

func cmdBrowseType() {
	cfg := loadAndValidate()
	entries, err := getEntries(cfg)
	if err != nil {
		fatal(err)
	}
	sel, err := wofiSelect(cfg, entries, "keepass (type)")
	if err != nil {
		if err == errCancelled {
			os.Exit(1)
		}
		fatal(err)
	}
	runAction(cfg, sel, true)
}

func cmdCopy() {
	cfg := loadAndValidate()
	entry := ""
	if len(os.Args) >= 3 {
		entry = os.Args[2]
	} else {
		entries, err := getEntries(cfg)
		if err != nil {
			fatal(err)
		}
		entry, err = wofiSelect(cfg, entries, "keepass")
		if err != nil {
			if err == errCancelled {
				os.Exit(1)
			}
			fatal(err)
		}
	}
	runAction(cfg, entry, false)
}

// getEntries serves an unlocked agent's in-memory list instantly, or falls back
// to the on-disk cache so the menu never blocks on keepass decryption.
func getEntries(cfg Config) ([]string, error) {
	if entries, ok := agentList(cfg); ok {
		return entries, nil
	}

	entries := readCache(cfg.cacheFile())
	if len(entries) == 0 {
		pass, err := ensureUnlocked(cfg, true)
		if err != nil {
			return nil, err
		}
		if err := refreshCache(cfg, pass); err != nil {
			return nil, err
		}
		return readCache(cfg.cacheFile()), nil
	}

	if cfg.cacheStale() && agentAlive(cfg) {
		go agentCall(cfg, agentRequest{Op: "refresh"})
	}
	return entries, nil
}

func runAction(cfg Config, entry string, doType bool) {
	pass, err := ensureUnlocked(cfg, true)
	if err != nil {
		fatal(err)
	}

	password, err := cfg.showField(entry, "Password", pass)
	if err != nil {
		fatal(err)
	}

	if doType {
		username, _ := cfg.showField(entry, "UserName", pass)
		if err := autotype(cfg, username, password); err != nil {
			fatal(fmt.Errorf("autotype: %w", err))
		}
		return
	}

	if err := copyClipboard(cfg, password); err != nil {
		fatal(fmt.Errorf("clipboard: %w", err))
	}
	clearClipboardLater(cfg, password, cfg.ClipboardClear)
	notify(cfg, "Password copied: "+strings.TrimSpace(entry))
}

func notify(cfg Config, msg string) {
	if _, err := os.Stat("/usr/bin/notify-send"); err == nil {
		exec.Command("notify-send", "-a", "kpmenu", "kpmenu", msg).Run()
	}
}
