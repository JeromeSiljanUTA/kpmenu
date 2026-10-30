package main

import (
	"fmt"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) < 2 {
		cmdBrowse()
		return
	}
	switch os.Args[1] {
	case "type":
		cmdBrowseType()
	case "view":
		cmdView()
	case "refresh":
		cmdRefresh()
	case "unlock":
		cmdUnlock()
	case "lock":
		cmdLock()
	case "run", "agent":
		cmdAgent()
	case "clipwatch":
		cmdClipWatch()
	case "menu":
		cmdBrowse()
	case "copy":
		cmdCopy()
	case "version", "-v", "--version":
		fmt.Println("kpmenu 0.1.0")
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "kpmenu: unknown command %q\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Print(`kpmenu - fast wofi frontend for keepass databases

usage:
  kpmenu [menu]         pick an entry and copy its password
  kpmenu type           pick an entry and autotype username/password
  kpmenu view           browse, read, edit and add entries interactively
  kpmenu copy [entry]   copy an entry without the menu
  kpmenu refresh        rebuild the entry cache
  kpmenu unlock         unlock the agent / cache the passphrase
  kpmenu lock           forget the cached passphrase
`)
}

func loadAndValidate() Config {
	cfg := loadConfig()
	if cfg.Database == "" {
		fmt.Fprintln(os.Stderr, "kpmenu: no database configured (set 'database' in ~/.config/kpmenu/config)")
		os.Exit(1)
	}
	return cfg
}

func cmdRefresh() {
	cfg := loadAndValidate()
	pass, err := ensureUnlocked(cfg, true)
	if err != nil {
		fatal(err)
	}
	if err := refreshCache(cfg, pass); err != nil {
		fatal(err)
	}
}

func cmdUnlock() {
	cfg := loadAndValidate()
	if _, err := ensureUnlocked(cfg, true); err != nil {
		fatal(err)
	}
}

func cmdLock() {
	cfg := loadConfig()
	if agentAlive(cfg) {
		agentCall(cfg, agentRequest{Op: "lock"})
	}
}

func cmdAgent() {
	cfg := loadAndValidate()
	if err := runAgent(cfg); err != nil {
		fatal(err)
	}
}

func cmdClipWatch() {
	cfg := loadConfig()
	if len(os.Args) < 4 {
		os.Exit(2)
	}
	seconds, _ := strconv.Atoi(os.Args[2])
	value := os.Args[3]
	runClipWatch(cfg, seconds, value)
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "kpmenu: %v\n", err)
	os.Exit(1)
}
