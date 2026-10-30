package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func wofiSelect(cfg Config, entries []string, prompt string) (string, error) {
	args := []string{"--dmenu", "--insensitive", "--prompt", prompt}
	cmd := exec.Command(cfg.Wofi, args...)
	cmd.Stdin = strings.NewReader(strings.Join(entries, "\n"))
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			return "", errCancelled
		}
		return "", err
	}
	sel := strings.TrimRight(string(out), "\r\n")
	if sel == "" {
		return "", errCancelled
	}
	return sel, nil
}

func wofiPassword(cfg Config, prompt string) (string, error) {
	args := []string{"--dmenu", "--password", "--prompt", prompt}
	cmd := exec.Command(cfg.Wofi, args...)
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

func wofiInput(cfg Config, prompt, initial string) (string, error) {
	args := []string{"--dmenu", "--prompt", prompt}
	cmd := exec.Command(cfg.Wofi, args...)
	cmd.Stdin = strings.NewReader(initial)
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

var errCancelled = fmt.Errorf("cancelled")
