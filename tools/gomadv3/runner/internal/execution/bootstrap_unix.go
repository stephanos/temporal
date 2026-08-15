//go:build unix

package execution

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

	"go.temporal.io/server/tools/gomadv3/choice"
)

func BootstrapMain() (retErr error) {
	defer func() {
		capabilities := launchCapabilities{ioTranscript: true, readOnlyMount: true, choiceTrace: true, choiceReplayPlan: true}
		retErr = errors.Join(retErr, closeDescriptors(bootstrapIOTranscriptFD, bootstrapIOTerminalFD, bootstrapIOExpectedFD, bootstrapIOROMountRequestFD, bootstrapIOROMountResponseFD, descriptorFor(bootstrapStage, capabilities, choiceTraceResource), descriptorFor(bootstrapStage, capabilities, choiceTerminalResource), descriptorFor(bootstrapStage, capabilities, choiceTapeResource)))
	}()
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
	if err := validateChoiceEnvironment(request.Env); err != nil {
		return errors.Join(err, closeDescriptors(bootstrapActivationFD, bootstrapReadinessFD, bootstrapWorldConfigFD, bootstrapWorldRecordFD))
	}
	if len(request.IOConfig) > maximumIOConfigBytes {
		return errors.Join(errors.New("target I/O configuration exceeds its bound"), closeDescriptors(bootstrapActivationFD, bootstrapReadinessFD, bootstrapWorldConfigFD, bootstrapWorldRecordFD))
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
	capabilities := launchCapabilities{ioTranscript: request.IOTranscriptLimit != 0, readOnlyMount: request.IOROMounts, choiceTrace: request.ChoiceTrace, choiceReplayPlan: request.ChoiceTapeBytes != 0}
	if err := installTargetStage(capabilities); err != nil {
		return err
	}
	if err := installIOConfig(request.IOConfig); err != nil {
		return errors.Join(err, closeDescriptors(targetWorldConfigFD, targetWorldRecordFD))
	}
	if err := os.Chdir(request.Dir); err != nil {
		return fmt.Errorf("change target working directory: %w", err)
	}
	argv := make([]string, 1, len(request.Args)+1)
	argv[0] = request.Argv0
	argv = append(argv, request.Args...)
	if request.ChoiceTrace {
		choiceTraceFD := descriptorFor(targetStage, capabilities, choiceTraceResource)
		choiceTerminalFD := descriptorFor(targetStage, capabilities, choiceTerminalResource)
		request.Env = append(request.Env,
			choiceProfileEnvironmentName+"="+choice.Profile,
			fmt.Sprintf("%s=%d", choiceModeEnvironmentName, request.ChoiceMode),
			fmt.Sprintf("%s=%d", choiceTraceFDEnvironmentName, choiceTraceFD),
			fmt.Sprintf("%s=%d", choiceTerminalFDEnvironmentName, choiceTerminalFD),
			fmt.Sprintf("%s=%d", choiceTraceBytesEnvironmentName, request.ChoiceTraceLimit),
		)
		if request.ChoiceTapeBytes != 0 {
			choiceTapeFD := descriptorFor(targetStage, capabilities, choiceTapeResource)
			request.Env = append(request.Env,
				fmt.Sprintf("%s=%d", choiceTapeFDEnvironmentName, choiceTapeFD),
				fmt.Sprintf("%s=%d", choiceTapeBytesEnvironmentName, request.ChoiceTapeBytes),
			)
		}
	}
	return syscall.Exec(request.Command, argv, request.Env)
}

func installIOConfig(configuration []byte) error {
	var descriptors [2]int
	if err := syscall.Pipe(descriptors[:]); err != nil {
		return fmt.Errorf("create target I/O configuration pipe: %w", err)
	}
	readDescriptor, writeDescriptor := descriptors[0], descriptors[1]
	if len(configuration) != 0 {
		written, err := syscall.Write(writeDescriptor, configuration)
		if err != nil || written != len(configuration) {
			return errors.Join(fmt.Errorf("write target I/O configuration: wrote %d: %w", written, err), closeDescriptors(readDescriptor, writeDescriptor))
		}
	}
	if err := syscall.Close(writeDescriptor); err != nil {
		return errors.Join(fmt.Errorf("close target I/O configuration writer: %w", err), closeDescriptors(readDescriptor))
	}
	if readDescriptor == targetIOConfigFD {
		return nil
	}
	if err := syscall.Dup2(readDescriptor, targetIOConfigFD); err != nil {
		return errors.Join(fmt.Errorf("install target I/O configuration descriptor: %w", err), closeDescriptors(readDescriptor))
	}
	if err := syscall.Close(readDescriptor); err != nil {
		return errors.Join(fmt.Errorf("close target I/O configuration reader: %w", err), closeDescriptors(targetIOConfigFD))
	}
	return nil
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
