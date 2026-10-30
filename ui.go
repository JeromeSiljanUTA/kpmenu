package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// wofiArgs returns the shared wofi arguments. Arrow keys keep their default
// behaviour while Ctrl-n and Ctrl-p are added for emacs-style navigation.
func wofiArgs(args ...string) []string {
	base := []string{
		"--define", "key_up=Up,Ctrl-p",
		"--define", "key_down=Down,Ctrl-n",
	}
	return append(base, args...)
}

func wofiRun(cfg Config, args []string, stdin string) (string, error) {
	cmd := exec.Command(cfg.Wofi, args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			return "", errCancelled
		}
		return "", err
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

func wofiSelect(cfg Config, entries []string, prompt string) (string, error) {
	args := wofiArgs("--dmenu", "--insensitive", "--prompt", prompt)
	sel, err := wofiRun(cfg, args, strings.Join(entries, "\n"))
	if err != nil {
		return "", err
	}
	if sel == "" {
		return "", errCancelled
	}
	return sel, nil
}

// wofiPassword shows the prompt as a visible row because wofi's --prompt is
// only placeholder text and is invisible under many themes. exec-search makes
// Enter submit the typed value rather than the highlighted row.
func wofiPassword(cfg Config, prompt string) (string, error) {
	args := wofiArgs("--dmenu", "--password", "--exec-search", "--prompt", prompt)
	return wofiRun(cfg, args, prompt+" - type and press Enter")
}

func wofiInput(cfg Config, prompt, initial string) (string, error) {
	args := wofiArgs("--dmenu", "--prompt", prompt)
	return wofiRun(cfg, args, initial)
}

var errCancelled = fmt.Errorf("cancelled")
