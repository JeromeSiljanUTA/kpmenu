package main

import (
	"os/exec"
	"strings"
)

func autotype(cfg Config, username, password string) error {
	seq := username
	if username != "" {
		seq += "\t"
	}
	seq += password
	cmd := exec.Command(cfg.Wtype, "-", "-d", "40")
	cmd.Stdin = strings.NewReader(seq)
	return cmd.Run()
}
