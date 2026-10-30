package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type agentState struct {
	passphrase string
	db         string
	lastUsed   time.Time
}

type agentRequest struct {
	Op      string `json:"op"`
	DB      string `json:"db"`
	Refresh bool   `json:"refresh"`
}

type agentResponse struct {
	OK    bool   `json:"ok"`
	Value string `json:"value"`
	Error string `json:"error"`
}

func runAgent(cfg Config) error {
	if err := ensureDir(filepath.Dir(cfg.SocketPath)); err != nil {
		return err
	}
	os.Remove(cfg.SocketPath)

	ln, err := net.Listen("unix", cfg.SocketPath)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer ln.Close()
	os.Chmod(cfg.SocketPath, 0o600)

	st := &agentState{db: cfg.Database, lastUsed: time.Now()}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	expired := make(chan struct{})
	go func() {
		tick := time.NewTicker(30 * time.Second)
		defer tick.Stop()
		for range tick.C {
			if cfg.LockTimeout > 0 && time.Since(st.lastUsed) > time.Duration(cfg.LockTimeout)*time.Second {
				close(expired)
				return
			}
		}
	}()

	go func() {
		<-expired
		ln.Close()
		os.Remove(cfg.SocketPath)
		os.Exit(0)
	}()

	go func() {
		<-sig
		ln.Close()
		os.Remove(cfg.SocketPath)
		os.Exit(0)
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-expired:
				return nil
			default:
				return err
			}
		}
		go handleConn(cfg, st, conn)
	}
}

func handleConn(cfg Config, st *agentState, conn net.Conn) {
	defer conn.Close()
	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)

	var req agentRequest
	if err := dec.Decode(&req); err != nil {
		enc.Encode(agentResponse{Error: "bad request"})
		return
	}
	st.lastUsed = time.Now()

	switch req.Op {
	case "unlock":
		if st.passphrase != "" && req.DB == st.db {
			enc.Encode(agentResponse{OK: true})
			return
		}
		pw, err := promptPassphrase(cfg)
		if err != nil {
			enc.Encode(agentResponse{Error: err.Error()})
			return
		}
		if _, err := cfg.listEntries(pw); err != nil {
			enc.Encode(agentResponse{Error: "unlock failed: " + err.Error()})
			return
		}
		st.passphrase = pw
		st.db = req.DB
		if req.Refresh {
			refreshCache(cfg, pw)
		}
		enc.Encode(agentResponse{OK: true})
	case "pass":
		if st.passphrase == "" {
			enc.Encode(agentResponse{Error: "locked"})
			return
		}
		enc.Encode(agentResponse{OK: true, Value: st.passphrase})
	case "lock":
		st.passphrase = ""
		enc.Encode(agentResponse{OK: true})
	case "refresh":
		if st.passphrase == "" {
			enc.Encode(agentResponse{Error: "locked"})
			return
		}
		if err := refreshCache(cfg, st.passphrase); err != nil {
			enc.Encode(agentResponse{Error: err.Error()})
			return
		}
		enc.Encode(agentResponse{OK: true})
	default:
		enc.Encode(agentResponse{Error: "unknown op"})
	}
}

func promptPassphrase(cfg Config) (string, error) {
	if _, err := exec.LookPath(cfg.Pinentry); err == nil {
		cmd := exec.Command(cfg.Pinentry)
		return pinentryDialog(cmd, "Passphrase for "+filepath.Base(cfg.Database))
	}
	if fi, _ := os.Stdin.Stat(); fi != nil && (fi.Mode()&os.ModeCharDevice) != 0 {
		fmt.Fprint(os.Stderr, "Passphrase: ")
	}
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func pinentryDialog(cmd *exec.Cmd, prompt string) (string, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}
	reader := bufio.NewReader(stdout)
	// Drain the OK/ERR acknowledgement for each negotiation command before
	// issuing getpin; otherwise the first OK is mistaken for the pin reply.
	negotiate := []string{
		"setdesc " + prompt,
		"setprompt Passphrase:",
		"setok OK",
		"setcancel Cancel",
	}
	for _, c := range negotiate {
		fmt.Fprintln(stdin, c)
		if _, err := reader.ReadString('\n'); err != nil {
			stdin.Close()
			cmd.Wait()
			return "", err
		}
	}
	fmt.Fprintln(stdin, "getpin")

	var result string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimRight(line, "\r\n")
		if strings.HasPrefix(line, "D ") {
			result += line[2:]
		} else if strings.TrimSpace(line) == "OK" && result != "" {
			break
		} else if strings.HasPrefix(line, "ERR") {
			stdin.Close()
			cmd.Wait()
			return "", fmt.Errorf("pinentry cancelled")
		}
	}
	stdin.Close()
	cmd.Wait()
	if result == "" {
		return "", fmt.Errorf("no passphrase provided")
	}
	return result, nil
}
