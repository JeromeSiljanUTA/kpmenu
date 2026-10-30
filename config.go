package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Database       string
	KeyFile        string
	KeepassxcCli   string
	Wofi           string
	Wtype          string
	WlCopy         string
	Pinentry       string
	Prompt         string
	CacheDir       string
	ClipboardClear int
	LockTimeout    int
	HideGroups     []string
	SocketPath     string
}

func defaultConfig() Config {
	runtime := os.Getenv("XDG_RUNTIME_DIR")
	if runtime == "" {
		runtime = "/tmp"
	}
	cacheDir := filepath.Join(runtime, "kpmenu")
	return Config{
		KeepassxcCli:   "keepassxc-cli",
		Wofi:           "wofi",
		Wtype:          "wtype",
		WlCopy:         "wl-copy",
		Pinentry:       "pinentry",
		Prompt:         "wofi",
		CacheDir:       cacheDir,
		ClipboardClear: 30,
		LockTimeout:    3600,
		HideGroups:     []string{"Recycle Bin"},
		SocketPath:     filepath.Join(cacheDir, "agent.sock"),
	}
}

func configPath() string {
	if p := os.Getenv("KPMENU_CONFIG"); p != "" {
		return p
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "kpmenu", "config")
}

func loadConfig() Config {
	cfg := defaultConfig()
	f, err := os.Open(configPath())
	if err != nil {
		return cfg
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		switch k {
		case "database":
			cfg.Database = expandHome(v)
		case "keyfile":
			cfg.KeyFile = expandHome(v)
		case "keepassxc_cli":
			cfg.KeepassxcCli = v
		case "wofi":
			cfg.Wofi = v
		case "wtype":
			cfg.Wtype = v
		case "wl_copy":
			cfg.WlCopy = v
		case "pinentry":
			cfg.Pinentry = v
		case "passphrase_prompt":
			cfg.Prompt = v
		case "cache_dir":
			cfg.CacheDir = expandHome(v)
		case "clipboard_clear":
			if n, err := strconv.Atoi(v); err == nil {
				cfg.ClipboardClear = n
			}
		case "lock_timeout":
			if n, err := strconv.Atoi(v); err == nil {
				cfg.LockTimeout = n
			}
		case "hide_groups":
			cfg.HideGroups = nil
			for _, g := range strings.Split(v, ",") {
				if g = strings.TrimSpace(g); g != "" {
					cfg.HideGroups = append(cfg.HideGroups, g)
				}
			}
		}
	}
	cfg.SocketPath = filepath.Join(cfg.CacheDir, "agent.sock")
	return cfg
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

func (c Config) cacheFile() string {
	return filepath.Join(c.CacheDir, "entries.tsv")
}
