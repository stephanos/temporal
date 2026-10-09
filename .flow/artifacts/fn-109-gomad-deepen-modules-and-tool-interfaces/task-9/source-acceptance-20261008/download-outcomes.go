package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"go.temporal.io/server/tools/gomad3/target"
)

type cause struct {
	Type string
	Text string
	Children []cause
}

func chain(err error) cause {
	if err == nil { return cause{} }
	v := cause{Type: fmt.Sprintf("%T", err), Text: err.Error()}
	if multi, ok := err.(interface{ Unwrap() []error }); ok {
		for _, child := range multi.Unwrap() { v.Children = append(v.Children, chain(child)) }
	} else if child := errors.Unwrap(err); child != nil { v.Children = append(v.Children, chain(child)) }
	return v
}

func main() {
	base, err := os.MkdirTemp("", "task9-download-contract-")
	if err != nil { panic(err) }
	defer func() { if err := os.RemoveAll(base); err != nil { panic(err) } }()
	for _, tc := range []struct{name, output, stderr, mode string; exit int}{
		{"success", `{"Sum":"h1:accepted"}`, "", "", 0},
		{"checksum", `{"Sum":"h1:other"}`, "", "", 0},
		{"reported_error_before_exit", `{"Error":"module refused"}`, "diagnostic", "", 7},
		{"nonzero", `{"Sum":"h1:accepted"}`, "diagnostic", "", 7},
		{"malformed_nonzero", `{broken`, "diagnostic", "", 7},
		{"malformed", `{broken`, "diagnostic", "", 0},
		{"before_cancel", `{"Sum":"h1:accepted"}`, "diagnostic", "before", 0},
		{"cancel_after_ack", `{"Error":"module refused"}`, "diagnostic", "cancel", 0},
		{"deadline_after_ack", `{"Error":"module refused"}`, "diagnostic", "deadline", 0},
	} {
		root := filepath.Join(base, tc.name)
		if err := os.MkdirAll(filepath.Join(root, "bin"), 0o700); err != nil { panic(err) }
		marker := filepath.Join(root, "started")
		body := fmt.Sprintf("#!/bin/sh\nprintf '%%s' '%s'\nprintf '%%s' '%s' >&2\n", tc.output, tc.stderr)
		if tc.mode == "cancel" || tc.mode == "deadline" {
			body += fmt.Sprintf("touch '%s'\nexec /bin/sleep 30\n", marker)
		} else { body += fmt.Sprintf("exit %d\n", tc.exit) }
		if err := os.WriteFile(filepath.Join(root,"bin/go"), []byte(body), 0o700); err != nil { panic(err) }
		ctx, cancel := context.WithCancel(context.Background())
		if tc.mode == "before" { cancel() }
		if tc.mode == "deadline" { cancel(); ctx, cancel = context.WithTimeout(context.Background(), 500*time.Millisecond) }
		result := make(chan error, 1)
		go func() { result <- target.DownloadModule(ctx, root, target.ModuleIdentity{Path:"example.com/module",Version:"v1.2.3",Sum:"h1:accepted"}) }()
		ack := false
		if tc.mode == "cancel" || tc.mode == "deadline" {
			ticker := time.NewTicker(time.Millisecond)
			watchdog := time.NewTimer(2*time.Second)
			for !ack {
				select {
				case <-ticker.C:
					_, statErr := os.Stat(marker)
					ack = statErr == nil
					if statErr != nil && !errors.Is(statErr, os.ErrNotExist) { panic(statErr) }
				case <-watchdog.C: panic("child acknowledgement missing")
				}
			}
			ticker.Stop(); watchdog.Stop()
			if tc.mode == "cancel" { cancel() }
		}
		err := <-result
		cancel()
		var exitErr *exec.ExitError
		var syntax *json.SyntaxError
		asExit := errors.As(err,&exitErr)
		exit, signal := 0, ""
		if asExit { exit = exitErr.ExitCode(); if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signaled() { signal = status.Signal().String() } }
		row := struct{Name string; Cause cause; Canceled, Deadline, ExecExit, JSONSyntax bool; Exit int; Signal string; Acknowledged bool}{tc.name,chain(err),errors.Is(err,context.Canceled),errors.Is(err,context.DeadlineExceeded),asExit,errors.As(err,&syntax),exit,signal,ack}
		if err := json.NewEncoder(os.Stdout).Encode(row); err != nil { panic(err) }
	}
}
