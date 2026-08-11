//go:build unix

package process

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
)

const (
	targetWorldConfigFD = 3 + iota
	targetWorldRecordFD
)

func BootstrapMain() error {
	signal.Reset(syscall.SIGTERM)
	if err := reportTargetIdentity(); err != nil {
		return errors.Join(err, closeDescriptors(bootstrapRequestFD, bootstrapActivationFD, bootstrapReadinessFD, bootstrapWorldConfigFD, bootstrapWorldRecordFD, bootstrapIdentityFD))
	}
	requestBytes, err := readBootstrapRequest(bootstrapRequestFD)
	if err != nil {
		return errors.Join(err, closeDescriptors(bootstrapRequestFD, bootstrapActivationFD, bootstrapReadinessFD, bootstrapWorldConfigFD, bootstrapWorldRecordFD))
	}
	if err := syscall.Close(bootstrapRequestFD); err != nil {
		return errors.Join(fmt.Errorf("close target bootstrap request: %w", err), closeDescriptors(bootstrapActivationFD, bootstrapReadinessFD, bootstrapWorldConfigFD, bootstrapWorldRecordFD))
	}
	decoder := json.NewDecoder(bytes.NewReader(requestBytes))
	decoder.DisallowUnknownFields()
	var request targetBootstrapRequest
	if err := decoder.Decode(&request); err != nil {
		return errors.Join(fmt.Errorf("decode target bootstrap request: %w", err), closeDescriptors(bootstrapActivationFD, bootstrapReadinessFD, bootstrapWorldConfigFD, bootstrapWorldRecordFD))
	}
	if token, err := decoder.Token(); err != io.EOF {
		return errors.Join(fmt.Errorf("target bootstrap request has trailing data: %v: %w", token, err), closeDescriptors(bootstrapActivationFD, bootstrapReadinessFD, bootstrapWorldConfigFD, bootstrapWorldRecordFD))
	}
	if request.Command == "" || request.Argv0 == "" || request.Dir == "" {
		return errors.Join(fmt.Errorf("target bootstrap request is incomplete"), closeDescriptors(bootstrapActivationFD, bootstrapReadinessFD, bootstrapWorldConfigFD, bootstrapWorldRecordFD))
	}
	written, err := syscall.Write(bootstrapReadinessFD, []byte{1})
	if err != nil {
		return errors.Join(fmt.Errorf("report target bootstrap readiness: %w", err), closeDescriptors(bootstrapActivationFD, bootstrapReadinessFD, bootstrapWorldConfigFD, bootstrapWorldRecordFD))
	}
	if written != 1 {
		return errors.Join(fmt.Errorf("report target bootstrap readiness: wrote %d: %w", written, io.ErrShortWrite), closeDescriptors(bootstrapActivationFD, bootstrapReadinessFD, bootstrapWorldConfigFD, bootstrapWorldRecordFD))
	}
	if err := syscall.Close(bootstrapReadinessFD); err != nil {
		return errors.Join(fmt.Errorf("close target bootstrap readiness: %w", err), closeDescriptors(bootstrapActivationFD, bootstrapWorldConfigFD, bootstrapWorldRecordFD))
	}
	var activated [1]byte
	read, err := syscall.Read(bootstrapActivationFD, activated[:])
	if err != nil {
		return errors.Join(fmt.Errorf("read target activation: %w", err), closeDescriptors(bootstrapActivationFD, bootstrapWorldConfigFD, bootstrapWorldRecordFD))
	}
	if read != 1 {
		return errors.Join(fmt.Errorf("read target activation: read %d: %w", read, io.ErrUnexpectedEOF), closeDescriptors(bootstrapActivationFD, bootstrapWorldConfigFD, bootstrapWorldRecordFD))
	}
	if activated[0] != 1 {
		return errors.Join(fmt.Errorf("invalid target activation"), closeDescriptors(bootstrapActivationFD, bootstrapWorldConfigFD, bootstrapWorldRecordFD))
	}
	if err := syscall.Close(bootstrapActivationFD); err != nil {
		return errors.Join(fmt.Errorf("close target activation: %w", err), closeDescriptors(bootstrapWorldConfigFD, bootstrapWorldRecordFD))
	}
	if err := syscall.Dup2(bootstrapWorldConfigFD, targetWorldConfigFD); err != nil {
		return errors.Join(fmt.Errorf("install target World configuration descriptor: %w", err), closeDescriptors(bootstrapWorldConfigFD, bootstrapWorldRecordFD))
	}
	if err := syscall.Dup2(bootstrapWorldRecordFD, targetWorldRecordFD); err != nil {
		return errors.Join(fmt.Errorf("install target World record descriptor: %w", err), closeDescriptors(bootstrapWorldConfigFD, bootstrapWorldRecordFD, targetWorldConfigFD))
	}
	if err := closeDescriptors(bootstrapWorldConfigFD, bootstrapWorldRecordFD); err != nil {
		return fmt.Errorf("close target bootstrap descriptors: %w", err)
	}
	if err := os.Chdir(request.Dir); err != nil {
		return fmt.Errorf("change target working directory: %w", err)
	}
	argv := make([]string, 1, len(request.Args)+1)
	argv[0] = request.Argv0
	argv = append(argv, request.Args...)
	return syscall.Exec(request.Command, argv, request.Env)
}

func reportTargetIdentity() error {
	pid := syscall.Getpid()
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		return fmt.Errorf("read target bootstrap process group: %w", err)
	}
	var encoded [16]byte
	binary.BigEndian.PutUint64(encoded[:8], uint64(pid))
	binary.BigEndian.PutUint64(encoded[8:], uint64(pgid))
	written, err := syscall.Write(bootstrapIdentityFD, encoded[:])
	if err != nil {
		return fmt.Errorf("report target bootstrap identity: %w", err)
	}
	if written != len(encoded) {
		return fmt.Errorf("report target bootstrap identity: wrote %d: %w", written, io.ErrShortWrite)
	}
	if err := syscall.Close(bootstrapIdentityFD); err != nil {
		return fmt.Errorf("close target bootstrap identity: %w", err)
	}
	return nil
}

func readBootstrapRequest(descriptor int) ([]byte, error) {
	const maximumBootstrapRequestBytes = 16 << 20
	result := make([]byte, 0, 4096)
	var buffer [4096]byte
	for {
		read, err := syscall.Read(descriptor, buffer[:])
		if read > 0 {
			if len(result) > maximumBootstrapRequestBytes-read {
				return nil, fmt.Errorf("target bootstrap request exceeds its bound")
			}
			result = append(result, buffer[:read]...)
		}
		if err != nil {
			return nil, fmt.Errorf("read target bootstrap request: %w", err)
		}
		if read == 0 {
			return result, nil
		}
	}
}

func closeDescriptors(descriptors ...int) error {
	var result error
	for _, descriptor := range descriptors {
		if err := syscall.Close(descriptor); err != nil && !errors.Is(err, syscall.EBADF) {
			result = errors.Join(result, err)
		}
	}
	return result
}
