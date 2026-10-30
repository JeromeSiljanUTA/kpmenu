package main

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func copyClipboard(cfg Config, value string) error {
	cmd := exec.Command(cfg.WlCopy)
	cmd.Stdin = strings.NewReader(value)
	return cmd.Run()
}

func clearClipboardLater(cfg Config, value string, seconds int) {
	if seconds <= 0 {
		return
	}
	self, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command(self, "clipwatch", strconv.Itoa(seconds), value)
	cmd.Env = os.Environ()
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = detachedSysProcAttr()
	cmd.Start()
	go cmd.Wait()
}

func runClipWatch(cfg Config, seconds int, value string) {
	time.Sleep(time.Duration(seconds) * time.Second)
	clearCmd := exec.Command("sh", "-c", cfg.WlCopy+" -c")
	if err := clearCmd.Run(); err != nil {
		current := readClipboard(cfg)
		if current == value {
			exec.Command("sh", "-c", cfg.WlCopy+" -c").Run()
		}
	}
}

func readClipboard(cfg Config) string {
	out, err := exec.Command("sh", "-c", "wl-paste -n 2>/dev/null").Output()
	if err != nil {
		return ""
	}
	return string(out)
}
