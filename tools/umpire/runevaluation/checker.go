package runevaluation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	checkerExecutableName = "temporal-run-evaluation-checker"
	checkerTimeout        = 30 * time.Second
	checkerWaitDelay      = 2 * time.Second
)

var installedCheckerSHA256 string

type checkerFailureCode string

const (
	checkerFailureController      checkerFailureCode = "controller"
	checkerFailureMissing         checkerFailureCode = "missing"
	checkerFailureUnsafe          checkerFailureCode = "unsafe"
	checkerFailureNonRegular      checkerFailureCode = "non-regular"
	checkerFailureStart           checkerFailureCode = "start"
	checkerFailureCanceled        checkerFailureCode = "canceled"
	checkerFailureTimeout         checkerFailureCode = "timeout"
	checkerFailureExit            checkerFailureCode = "exit"
	checkerFailureStderr          checkerFailureCode = "stderr"
	checkerFailureOversized       checkerFailureCode = "oversized"
	checkerFailureInvalidRequest  checkerFailureCode = "invalid-request"
	checkerFailureInvalidResponse checkerFailureCode = "invalid-response"
)

type checkerFailure struct {
	code checkerFailureCode
}

func (failure *checkerFailure) Error() string {
	if failure == nil {
		return ""
	}
	return "checker failure: " + string(failure.code)
}

func (failure *checkerFailure) Is(target error) bool {
	other, ok := target.(*checkerFailure)
	return ok && failure != nil && other != nil && failure.code == other.code
}

type checkerProcess struct {
	controllerExecutable string
	expectedSHA256       string
	timeout              time.Duration
}

func runFixedChecker(ctx context.Context, request checkerRequest) (checkerResponse, error) {
	if installedCheckerSHA256 == "" {
		return checkerResponse{}, &checkerFailure{code: checkerFailureUnsafe}
	}
	controller, err := os.Executable()
	if err != nil {
		return checkerResponse{}, &checkerFailure{code: checkerFailureController}
	}
	return (checkerProcess{
		controllerExecutable: controller,
		expectedSHA256:       installedCheckerSHA256,
		timeout:              checkerTimeout,
	}).run(ctx, request)
}

func (process checkerProcess) run(ctx context.Context, request checkerRequest) (checkerResponse, error) {
	encoded, err := encodeCheckerRequest(request)
	if err != nil {
		return checkerResponse{}, &checkerFailure{code: checkerFailureInvalidRequest}
	}
	checker, err := resolveVerifiedCheckerSibling(process.controllerExecutable, process.expectedSHA256)
	if err != nil {
		return checkerResponse{}, err
	}
	timeout := process.timeout
	if timeout <= 0 {
		timeout = checkerTimeout
	}

	timeoutCause := errors.New("checker timeout")
	outputCause := errors.New("checker output limit")
	stderrCause := errors.New("checker stderr")
	timeoutContext, cancelTimeout := context.WithTimeoutCause(ctx, timeout, timeoutCause)
	defer cancelTimeout()
	runContext, cancelRun := context.WithCancelCause(timeoutContext)
	defer cancelRun(nil)
	stdout := newBoundedCapture(maximumCheckerProtocolBytes, func() { cancelRun(outputCause) })
	stderr := newBoundedCapture(maximumCheckerProtocolBytes, func() { cancelRun(stderrCause) })
	stderr.onWrite = func() { cancelRun(stderrCause) }

	command := exec.CommandContext(runContext, checker)
	command.Dir = filepath.Dir(checker)
	command.Env = []string{}
	command.Stdin = bytes.NewReader(encoded)
	command.Stdout = stdout
	command.Stderr = stderr
	command.WaitDelay = checkerWaitDelay
	if err := command.Start(); err != nil {
		if ctx.Err() != nil {
			return checkerResponse{}, &checkerFailure{code: checkerFailureCanceled}
		}
		if errors.Is(context.Cause(timeoutContext), timeoutCause) {
			return checkerResponse{}, &checkerFailure{code: checkerFailureTimeout}
		}
		return checkerResponse{}, &checkerFailure{code: checkerFailureStart}
	}
	waitErr := command.Wait()

	if ctx.Err() != nil {
		return checkerResponse{}, &checkerFailure{code: checkerFailureCanceled}
	}
	if errors.Is(context.Cause(timeoutContext), timeoutCause) {
		return checkerResponse{}, &checkerFailure{code: checkerFailureTimeout}
	}
	if stderr.exceeded() || stderr.length() != 0 ||
		errors.Is(context.Cause(runContext), stderrCause) {
		return checkerResponse{}, &checkerFailure{code: checkerFailureStderr}
	}
	if stdout.exceeded() || errors.Is(context.Cause(runContext), outputCause) {
		return checkerResponse{}, &checkerFailure{code: checkerFailureOversized}
	}
	if waitErr != nil {
		return checkerResponse{}, &checkerFailure{code: checkerFailureExit}
	}
	response, err := decodeCheckerResponse(stdout.take(), request)
	if err != nil {
		return checkerResponse{}, &checkerFailure{code: checkerFailureInvalidResponse}
	}
	return response, nil
}

func resolveCheckerSibling(controllerExecutable string) (string, error) {
	controller, err := filepath.Abs(controllerExecutable)
	if err != nil {
		return "", &checkerFailure{code: checkerFailureController}
	}
	controller, err = filepath.EvalSymlinks(controller)
	if err != nil {
		return "", &checkerFailure{code: checkerFailureController}
	}
	controllerInfo, err := os.Stat(controller)
	if err != nil || !controllerInfo.Mode().IsRegular() {
		return "", &checkerFailure{code: checkerFailureController}
	}
	directory := filepath.Dir(controller)
	candidate := filepath.Join(directory, checkerExecutableName)
	info, err := os.Lstat(candidate)
	if errors.Is(err, os.ErrNotExist) {
		return "", &checkerFailure{code: checkerFailureMissing}
	}
	if err != nil {
		return "", &checkerFailure{code: checkerFailureUnsafe}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", &checkerFailure{code: checkerFailureUnsafe}
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil || filepath.Clean(resolved) != filepath.Clean(candidate) {
		return "", &checkerFailure{code: checkerFailureUnsafe}
	}
	relative, err := filepath.Rel(directory, resolved)
	if err != nil || relative != checkerExecutableName || strings.ContainsRune(relative, filepath.Separator) {
		return "", &checkerFailure{code: checkerFailureUnsafe}
	}
	resolvedInfo, err := os.Stat(resolved)
	if err != nil || !resolvedInfo.Mode().IsRegular() {
		return "", &checkerFailure{code: checkerFailureNonRegular}
	}
	return resolved, nil
}

func resolveVerifiedCheckerSibling(controllerExecutable string, expectedSHA256 string) (string, error) {
	checker, file, err := openVerifiedCheckerSibling(controllerExecutable, expectedSHA256)
	if err != nil {
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", &checkerFailure{code: checkerFailureUnsafe}
	}
	return checker, nil
}

func openVerifiedCheckerSibling(
	controllerExecutable string,
	expectedSHA256 string,
) (string, *os.File, error) {
	checker, err := resolveCheckerSibling(controllerExecutable)
	if err != nil {
		return "", nil, err
	}
	before, err := os.Lstat(checker)
	if err != nil || !before.Mode().IsRegular() {
		return "", nil, &checkerFailure{code: checkerFailureUnsafe}
	}
	file, err := os.Open(checker)
	if err != nil {
		return "", nil, &checkerFailure{code: checkerFailureUnsafe}
	}
	failed := func() (string, *os.File, error) {
		_ = file.Close()
		return "", nil, &checkerFailure{code: checkerFailureUnsafe}
	}
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return failed()
	}
	if expectedSHA256 != "" {
		digest := sha256.New()
		if _, err := io.Copy(digest, file); err != nil {
			return failed()
		}
		if "sha256:"+hex.EncodeToString(digest.Sum(nil)) != expectedSHA256 {
			return failed()
		}
	}
	after, err := os.Lstat(checker)
	if err != nil || !os.SameFile(before, after) {
		return failed()
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return failed()
	}
	return checker, file, nil
}

type boundedCapture struct {
	mu        sync.Mutex
	data      []byte
	limit     int
	overLimit bool
	onLimit   func()
	onWrite   func()
	limitOnce sync.Once
	writeOnce sync.Once
}

func newBoundedCapture(limit int, onLimit func()) *boundedCapture {
	return &boundedCapture{limit: limit, onLimit: onLimit}
}

func (capture *boundedCapture) Write(value []byte) (int, error) {
	capture.mu.Lock()
	remaining := max(capture.limit-len(capture.data), 0)
	accepted := min(len(value), remaining)
	if accepted != 0 {
		capture.grow(accepted)
		capture.data = append(capture.data, value[:accepted]...)
	}
	exceeded := accepted != len(value)
	if exceeded {
		capture.overLimit = true
	}
	capture.mu.Unlock()

	if len(value) != 0 && capture.onWrite != nil {
		capture.writeOnce.Do(capture.onWrite)
	}
	if exceeded && capture.onLimit != nil {
		capture.limitOnce.Do(capture.onLimit)
	}
	return len(value), nil
}

func (capture *boundedCapture) grow(additional int) {
	required := len(capture.data) + additional
	if required <= cap(capture.data) {
		return
	}
	capacity := max(cap(capture.data)*2, 64)
	capacity = min(max(capacity, required), capture.limit)
	next := make([]byte, len(capture.data), capacity)
	copy(next, capture.data)
	capture.data = next
}

func (capture *boundedCapture) length() int {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	return len(capture.data)
}

func (capture *boundedCapture) take() []byte {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	data := capture.data
	capture.data = nil
	return data
}

func (capture *boundedCapture) capacity() int {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	return cap(capture.data)
}

func (capture *boundedCapture) exceeded() bool {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	return capture.overLimit
}
