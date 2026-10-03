// locked_syscall checks the ordering between timer delivery and a goroutine
// locked to its OS thread returning from a syscall. Transport modes hold a
// simulation-time response open so the syscall can return inside that round
// trip, making ownership of the idle P observable.
package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	childEnvironment = "GOMAD3_LOCKED_SYSCALL_CHILD"

	modeArrivalBeforeTimer              = "arrival-before-timer"
	modeArrivalAfterTimerFired          = "arrival-after-timer-fired"
	modeArrivalDuringQuiescence         = "arrival-during-quiescence-round-trip"
	directDataDescriptor                = 3
	timeRequestEnvironment              = "GOMAD3_SIMULATION_TIME_REQUEST_FD"
	timeResponseEnvironment             = "GOMAD3_SIMULATION_TIME_RESPONSE_FD"
	timeRequestDescriptor               = 3
	timeResponseDescriptor              = 4
	dataDescriptor                      = 5
	eventDescriptor                     = 6
	timerDescriptor                     = 7
	simulationTimeRequestBytes          = 40
	simulationTimeResponseBytes         = 32
	simulationTimeResponseAdvance       = 1
	simulationTimeResponseRetry         = 2
	simulationTimeInitial         int64 = 946684800000000000
)

var simulationTimeRequestMagic = [8]byte{'G', 'O', 'M', 'A', 'D', 'T', 'Q', 1}
var simulationTimeResponseMagic = [8]byte{'G', 'O', 'M', 'A', 'D', 'T', 'R', 1}

type simulationTimeRequest struct {
	generation uint64
	current    int64
	deadline   int64
}

//go:linkname runtimeEnterSyscallBlock runtime.entersyscallblock
func runtimeEnterSyscallBlock()

//go:linkname runtimeExitSyscall runtime.exitsyscall
func runtimeExitSyscall()

//go:linkname runtimeGomadBlockingRead runtime.gomadBlockingRead
func runtimeGomadBlockingRead(descriptor int32, destination unsafe.Pointer, bytes int32) int32

func main() {
	if os.Getenv(childEnvironment) != "" {
		child()
		return
	}
	targetProcess := len(os.Args) == 3 && os.Args[1] == "--target"
	if len(os.Args) != 2 && !targetProcess {
		fmt.Fprintln(os.Stderr, "usage: locked_syscall <mode>")
		os.Exit(2)
	}
	mode := os.Args[len(os.Args)-1]
	var output string
	var err error
	if targetProcess {
		output, err = target(mode)
	} else {
		err = harness(mode)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if output != "" {
		fmt.Println(output)
	}
}

func harness(mode string) error {
	switch mode {
	case modeArrivalBeforeTimer:
		return runDirectTarget(mode)
	case modeArrivalAfterTimerFired, modeArrivalDuringQuiescence:
		return runTransportTarget(mode)
	default:
		return fmt.Errorf("unknown locked syscall mode %q", mode)
	}
}

func runDirectTarget(mode string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		return err
	}
	target := exec.CommandContext(ctx, executable, "--target", mode)
	target.Env = targetEnv(false)
	target.ExtraFiles = []*os.File{reader}
	target.Stdout = os.Stdout
	target.Stderr = os.Stderr
	if err := target.Start(); err != nil {
		closeFiles(reader, writer)
		return fmt.Errorf("start direct target: %w", err)
	}
	externalWriter := exec.CommandContext(ctx, executable, mode)
	externalWriter.Env = []string{childEnvironment + "=1"}
	externalWriter.ExtraFiles = []*os.File{writer}
	externalWriter.Stdout = os.Stdout
	externalWriter.Stderr = os.Stderr
	if err := externalWriter.Start(); err != nil {
		_ = target.Process.Kill()
		_ = target.Wait()
		closeFiles(reader, writer)
		return fmt.Errorf("start external writer: %w", err)
	}
	closeFiles(reader, writer)
	targetErr := target.Wait()
	writerErr := externalWriter.Wait()
	if targetErr != nil {
		return fmt.Errorf("run direct target: %w", targetErr)
	}
	if writerErr != nil {
		return fmt.Errorf("run external writer: %w", writerErr)
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}

func runTransportTarget(mode string) error {
	requestRead, requestWrite, err := os.Pipe()
	if err != nil {
		return err
	}
	responseRead, responseWrite, err := os.Pipe()
	if err != nil {
		requestRead.Close()
		requestWrite.Close()
		return err
	}
	dataRead, dataWrite, err := os.Pipe()
	if err != nil {
		closeFiles(requestRead, requestWrite, responseRead, responseWrite)
		return err
	}
	eventRead, eventWrite, err := os.Pipe()
	if err != nil {
		closeFiles(requestRead, requestWrite, responseRead, responseWrite, dataRead, dataWrite)
		return err
	}
	timerRead, timerWrite, err := os.Pipe()
	if err != nil {
		closeFiles(requestRead, requestWrite, responseRead, responseWrite, dataRead, dataWrite, eventRead, eventWrite)
		return err
	}
	defer closeFiles(requestRead, responseWrite, dataWrite, eventRead, timerRead)

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	command := exec.CommandContext(ctx, executable, "--target", mode)
	command.Env = targetEnv(true)
	command.ExtraFiles = []*os.File{requestWrite, responseRead, dataRead, eventWrite, timerWrite}
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		return fmt.Errorf("start transport target: %w", err)
	}
	closeFiles(requestWrite, responseRead, dataRead, eventWrite, timerWrite)

	protocolErr := runTransportProtocol(mode, requestRead, responseWrite, dataWrite, eventRead, timerRead)
	if protocolErr != nil {
		_ = command.Process.Kill()
	}
	waitErr := command.Wait()
	if protocolErr != nil {
		return protocolErr
	}
	if waitErr != nil {
		return fmt.Errorf("wait for transport target: %w", waitErr)
	}
	return nil
}

func runTransportProtocol(mode string, requests, responses, data, events, timers *os.File) error {
	if err := requireByte(events, 'W', "transport syscall wait"); err != nil {
		return err
	}
	first, err := readSimulationTimeRequest(requests)
	if err != nil {
		return err
	}

	switch mode {
	case modeArrivalAfterTimerFired:
		if err := writeSimulationTimeResponse(responses, first, simulationTimeResponseAdvance, first.deadline); err != nil {
			return err
		}
		if err := requireByte(timers, 'T', "timer firing"); err != nil {
			return err
		}
		second, err := readSimulationTimeRequest(requests)
		if err != nil {
			return err
		}
		if err := writeByte(data, 1); err != nil {
			return fmt.Errorf("complete locked syscall: %w", err)
		}
		if err := requireNoByte(events, 'R', "syscall resumed before the post-timer quiescence response"); err != nil {
			return err
		}
		if err := writeSimulationTimeResponse(responses, second, simulationTimeResponseRetry, second.current); err != nil {
			return err
		}
		return requireByte(events, 'R', "syscall resumption")

	case modeArrivalDuringQuiescence:
		if err := writeByte(data, 1); err != nil {
			return fmt.Errorf("complete locked syscall: %w", err)
		}
		if err := requireNoByte(events, 'R', "syscall resumed before the quiescence response"); err != nil {
			return err
		}
		if err := writeSimulationTimeResponse(responses, first, simulationTimeResponseRetry, first.current); err != nil {
			return err
		}
		if err := requireByte(events, 'R', "syscall resumption after the quiescence response"); err != nil {
			return err
		}
		second, err := readSimulationTimeRequest(requests)
		if err != nil {
			return err
		}
		if err := writeSimulationTimeResponse(responses, second, simulationTimeResponseAdvance, second.deadline); err != nil {
			return err
		}
		return requireByte(timers, 'T', "eventual timer firing")
	default:
		return fmt.Errorf("unknown transport mode %q", mode)
	}
}

func target(mode string) (string, error) {
	switch mode {
	case modeArrivalBeforeTimer:
		return arrivalBeforeTimer()
	case modeArrivalAfterTimerFired, modeArrivalDuringQuiescence:
		return transportTarget(mode)
	default:
		return "", fmt.Errorf("unknown locked syscall target mode %q", mode)
	}
}

func arrivalBeforeTimer() (string, error) {
	timerDone := make(chan struct{})
	time.AfterFunc(20*time.Millisecond, func() { close(timerDone) })
	runtime.LockOSThread()
	var buffer [1]byte
	read, err := blockingRead(directDataDescriptor, buffer[:])
	runtime.UnlockOSThread()
	if err != nil {
		return "", err
	}
	if read != 1 || buffer[0] != 1 {
		return "", fmt.Errorf("locked read = (%d, %d), want (1, 1)", read, buffer[0])
	}
	select {
	case <-timerDone:
		return "", errors.New("virtual timer fired before the locked syscall arrived")
	default:
	}
	<-timerDone
	return "locked syscall arrived before timer; timer fired", nil
}

func transportTarget(mode string) (string, error) {
	if err := syscall.SetNonblock(dataDescriptor, false); err != nil {
		return "", fmt.Errorf("make data descriptor blocking: %w", err)
	}

	timerDone := make(chan struct{})
	timerResult := make(chan error, 1)
	time.AfterFunc(20*time.Millisecond, func() {
		timerResult <- rawWriteByte(timerDescriptor, 'T')
		close(timerDone)
	})
	syscallResult := make(chan error, 1)
	go func() {
		if err := rawWriteByte(eventDescriptor, 'W'); err != nil {
			syscallResult <- fmt.Errorf("announce transport syscall wait: %w", err)
			return
		}
		runtime.LockOSThread()
		var buffer [1]byte
		read := int(runtimeGomadBlockingRead(dataDescriptor, unsafe.Pointer(&buffer[0]), int32(len(buffer))))
		err := rawWriteByte(eventDescriptor, 'R')
		runtime.UnlockOSThread()
		if err == nil && (read != 1 || buffer[0] != 1) {
			err = fmt.Errorf("locked read = (%d, %d), want (1, 1)", read, buffer[0])
		}
		syscallResult <- err
	}()

	switch mode {
	case modeArrivalAfterTimerFired:
		<-timerDone
		if err := <-timerResult; err != nil {
			return "", err
		}
		if err := <-syscallResult; err != nil {
			return "", err
		}
		return "timer fired before locked syscall arrived", nil
	case modeArrivalDuringQuiescence:
		if err := <-syscallResult; err != nil {
			return "", err
		}
		select {
		case <-timerDone:
			return "", errors.New("virtual timer fired before the quiescent syscall arrival resumed")
		default:
		}
		<-timerDone
		if err := <-timerResult; err != nil {
			return "", err
		}
		return "locked syscall arrived during quiescence; timer fired", nil
	default:
		return "", fmt.Errorf("unknown transport target mode %q", mode)
	}
}

func blockingRead(descriptor int, buffer []byte) (int, error) {
	runtimeEnterSyscallBlock()
	count, _, errno := syscall.RawSyscall(syscall.SYS_READ, uintptr(descriptor), uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	runtimeExitSyscall()
	if errno != 0 {
		return int(count), errno
	}
	return int(count), nil
}

func rawWriteByte(descriptor int, value byte) error {
	_, _, errno := syscall.RawSyscall(syscall.SYS_WRITE, uintptr(descriptor), uintptr(unsafe.Pointer(&value)), 1)
	if errno != 0 {
		return errno
	}
	return nil
}

func child() {
	// The locked parent enters its read immediately after starting this child.
	time.Sleep(250 * time.Millisecond)
	buffer := []byte{1}
	if written, err := syscall.Write(3, buffer); err != nil || written != len(buffer) {
		fmt.Fprintf(os.Stderr, "write = (%d, %v), want (%d, nil)\n", written, err, len(buffer))
		os.Exit(1)
	}
}

func targetEnv(transport bool) []string {
	environment := filterEnvironment(os.Environ(), "GOMADSEED", "GOMAD3_CHILD_SEED", "GOMAD3_IO_PROFILE", childEnvironment, timeRequestEnvironment, timeResponseEnvironment)
	environment = append(environment, "GOMADSEED=1")
	if transport {
		environment = append(environment,
			fmt.Sprintf("%s=%d", timeRequestEnvironment, timeRequestDescriptor),
			fmt.Sprintf("%s=%d", timeResponseEnvironment, timeResponseDescriptor),
		)
	}
	return environment
}

func filterEnvironment(environment []string, names ...string) []string {
	filtered := make([]string, 0, len(environment))
	for _, entry := range environment {
		keep := true
		for _, name := range names {
			if strings.HasPrefix(entry, name+"=") {
				keep = false
				break
			}
		}
		if keep {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func readSimulationTimeRequest(source *os.File) (simulationTimeRequest, error) {
	var encoded [simulationTimeRequestBytes]byte
	if _, err := io.ReadFull(source, encoded[:]); err != nil {
		return simulationTimeRequest{}, fmt.Errorf("read simulation time request: %w", err)
	}
	if !bytes.Equal(encoded[:8], simulationTimeRequestMagic[:]) || !zero(encoded[36:]) {
		return simulationTimeRequest{}, errors.New("simulation time request is malformed")
	}
	request := simulationTimeRequest{
		generation: binary.BigEndian.Uint64(encoded[8:16]),
		current:    int64(binary.BigEndian.Uint64(encoded[16:24])),
		deadline:   int64(binary.BigEndian.Uint64(encoded[24:32])),
	}
	if request.generation == 0 || request.current < simulationTimeInitial || request.deadline < request.current {
		return simulationTimeRequest{}, errors.New("simulation time request values are invalid")
	}
	return request, nil
}

func writeSimulationTimeResponse(destination *os.File, request simulationTimeRequest, kind byte, current int64) error {
	var encoded [simulationTimeResponseBytes]byte
	copy(encoded[:8], simulationTimeResponseMagic[:])
	binary.BigEndian.PutUint64(encoded[8:16], request.generation)
	binary.BigEndian.PutUint64(encoded[16:24], uint64(current))
	encoded[24] = kind
	if _, err := destination.Write(encoded[:]); err != nil {
		return fmt.Errorf("write simulation time response: %w", err)
	}
	return nil
}

func requireByte(source *os.File, want byte, label string) error {
	var actual [1]byte
	if _, err := io.ReadFull(source, actual[:]); err != nil {
		return fmt.Errorf("read %s: %w", label, err)
	}
	if actual[0] != want {
		return fmt.Errorf("%s marker = %q, want %q", label, actual[0], want)
	}
	return nil
}

func requireNoByte(source *os.File, forbidden byte, label string) error {
	if err := source.SetReadDeadline(time.Now().Add(200 * time.Millisecond)); err != nil {
		return fmt.Errorf("set %s deadline: %w", label, err)
	}
	var actual [1]byte
	_, err := source.Read(actual[:])
	if clearErr := source.SetReadDeadline(time.Time{}); clearErr != nil {
		return fmt.Errorf("clear %s deadline: %w", label, clearErr)
	}
	if err == nil {
		if actual[0] == forbidden {
			return errors.New(label)
		}
		return fmt.Errorf("unexpected syscall marker %q before the quiescence response", actual[0])
	}
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return nil
	}
	return fmt.Errorf("observe %s: %w", label, err)
}

func writeByte(destination *os.File, value byte) error {
	_, err := destination.Write([]byte{value})
	return err
}

func zero(value []byte) bool {
	for _, current := range value {
		if current != 0 {
			return false
		}
	}
	return true
}

func closeFiles(files ...*os.File) {
	for _, file := range files {
		if file != nil {
			_ = file.Close()
		}
	}
}
