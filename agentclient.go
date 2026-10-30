package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"
)

func agentAlive(cfg Config) bool {
	conn, err := net.DialTimeout("unix", cfg.SocketPath, 200*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func startAgent(cfg Config) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(self, "agent")
	cmd.Env = os.Environ()
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = detachedSysProcAttr()
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	for i := 0; i < 50; i++ {
		if agentAlive(cfg) {
			return nil
		}
		time.Sleep(40 * time.Millisecond)
	}
	return fmt.Errorf("agent failed to start")
}

func agentCall(cfg Config, req agentRequest) (agentResponse, error) {
	conn, err := net.DialTimeout("unix", cfg.SocketPath, 2*time.Second)
	if err != nil {
		return agentResponse{}, err
	}
	defer conn.Close()
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return agentResponse{}, err
	}
	var resp agentResponse
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return agentResponse{}, err
	}
	return resp, nil
}

// agentFields asks the agent for a decrypted entry, decrypting only on a miss.
func agentFields(cfg Config, entry string) ([]Field, error) {
	resp, err := agentCall(cfg, agentRequest{Op: "entry", Entry: entry})
	if err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("%s", resp.Error)
	}
	return resp.Fields, nil
}

func agentInvalidate(cfg Config, entry string) {
	agentCall(cfg, agentRequest{Op: "invalidate", Entry: entry})
}

func ensureUnlocked(cfg Config, allowPrompt bool) (string, error) {
	if !agentAlive(cfg) {
		if !allowPrompt {
			return "", fmt.Errorf("agent not running")
		}
		if err := startAgent(cfg); err != nil {
			return "", err
		}
	}
	resp, err := agentCall(cfg, agentRequest{Op: "pass"})
	if err == nil && resp.OK && resp.Value != "" {
		return resp.Value, nil
	}
	if !allowPrompt {
		return "", fmt.Errorf("locked")
	}
	resp, err = agentCall(cfg, agentRequest{Op: "unlock", DB: cfg.Database, Refresh: true})
	if err != nil {
		return "", err
	}
	if !resp.OK {
		return "", fmt.Errorf("%s", resp.Error)
	}
	resp, err = agentCall(cfg, agentRequest{Op: "pass"})
	if err != nil || !resp.OK {
		return "", fmt.Errorf("unlock succeeded but no passphrase")
	}
	return resp.Value, nil
}
