package gomadchoicewire_test

import (
	"bytes"
	"fmt"
	"internal/testenv"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"internal/gomadchoicewire"
)

// The fixture prints what a heap allocation or a seeded draw made while
// recording would move: select and run-queue order, map iteration order, and
// the allocation count.
const diagnosticFixtureSource = `package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

func round(workers int) []int {
	var group sync.WaitGroup
	results := make(chan int, workers*8)
	for worker := range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			for step := range 8 {
				left := make(chan int, 1)
				right := make(chan int, 1)
				left <- worker
				right <- 100 + step
				select {
				case value := <-left:
					results <- value
				case value := <-right:
					results <- value
				}
				runtime.Gosched()
			}
		}()
	}
	group.Wait()
	close(results)
	var order []int
	for value := range results {
		order = append(order, value)
	}
	return order
}

func main() {
	first := round(4)
	runtime.GC()
	time.Sleep(time.Millisecond)
	second := round(4)
	keys := map[int]bool{}
	for key := range 16 {
		keys[key] = true
	}
	var iteration []int
	for key := range keys {
		iteration = append(iteration, key)
	}
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	fmt.Println(first, second, iteration)
	fmt.Println("mallocs", stats.Mallocs)
}
`

const (
	diagnosticFixtureChoiceBytes = 1 << 20
	diagnosticFixtureBytes       = 1 << 20
	diagnosticFixtureStart       = 946684800000000000
)

type diagnosticRequest struct {
	withoutChoiceTrace bool
	// choiceBytes is the choice trace capacity; zero takes the fixture default.
	choiceBytes uint64
	// diagnosticBytes is the diagnostic trace capacity; zero leaves it off.
	diagnosticBytes   uint64
	withoutByteBound  bool
	corruptHeader     bool
	extraEnvironment  []string
	wantConfiguration bool
}

type diagnosticResult struct {
	exitCode      int
	stdout        string
	stderr        string
	choiceRecords []byte
	terminal      []byte
	header        gomadchoicewire.DiagnosticHeader
	records       []gomadchoicewire.DiagnosticRecord
}

func TestDiagnosticTrace(t *testing.T) {
	testenv.MustHaveGoBuild(t)
	directory := t.TempDir()
	source := filepath.Join(directory, "main.go")
	if err := os.WriteFile(source, []byte(diagnosticFixtureSource), 0o600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(directory, "fixture")
	build := exec.Command(testenv.GoToolPath(t), "build", "-o", binary, source)
	build.Dir = directory
	build.Env = append(os.Environ(), "CGO_ENABLED=0", "GOFLAGS=", "GOWORK=off")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build fixture: %v\n%s", err, output)
	}

	t.Run("digests every choice point", func(t *testing.T) {
		run := runDiagnosticFixture(t, binary, diagnosticRequest{diagnosticBytes: diagnosticFixtureBytes})
		if run.exitCode != 0 || run.header.State != gomadchoicewire.DiagnosticComplete {
			t.Fatalf("exit = %d, header = %+v, stderr = %q", run.exitCode, run.header, run.stderr)
		}
		terminal, err := gomadchoicewire.DecodeTerminal(run.terminal)
		if err != nil {
			t.Fatal(err)
		}
		if count := uint64(len(run.records)); count == 0 || count != terminal.Records || count != uint64(len(run.choiceRecords))/gomadchoicewire.RecordBytes {
			t.Fatalf("diagnostic records = %d, choice records = %d, terminal records = %d", count, len(run.choiceRecords)/gomadchoicewire.RecordBytes, terminal.Records)
		}
		queued := false
		for index, record := range run.records {
			if record.Ordinal != uint64(index) {
				t.Fatalf("record %d has ordinal %d", index, record.Ordinal)
			}
			queued = queued || record.RunQueueLength != 0
			if index == 0 {
				continue
			}
			previous := run.records[index-1]
			if record.VirtualTime < previous.VirtualTime || record.Allocations < previous.Allocations || record.GCCycle < previous.GCCycle ||
				record.RunqDraws < previous.RunqDraws || record.SchedulerDraws < previous.SchedulerDraws || record.SelectDraws < previous.SelectDraws ||
				record.RuntimeRandDraws < previous.RuntimeRandDraws || record.RuntimeCheapRandDraws < previous.RuntimeCheapRandDraws ||
				record.TimerDraws < previous.TimerDraws || record.ClockTickDraws < previous.ClockTickDraws {
				t.Fatalf("record %d moved backwards: %+v after %+v", index, record, previous)
			}
		}
		first, last := run.records[0], run.records[len(run.records)-1]
		mallocs := diagnosticFixtureMallocs(t, run.stdout)
		if first.Allocations == 0 || last.Allocations <= first.Allocations || last.Allocations > mallocs {
			t.Fatalf("allocations run from %d to %d, target counted %d", first.Allocations, last.Allocations, mallocs)
		}
		if first.VirtualTime != diagnosticFixtureStart || last.VirtualTime < diagnosticFixtureStart+1_000_000 {
			t.Fatalf("virtual time runs from %d to %d", first.VirtualTime, last.VirtualTime)
		}
		if last.GCCycle == 0 || !queued || last.RunqDraws == 0 || last.SelectDraws == 0 || last.TimerDraws == 0 {
			t.Fatalf("digest fields did not follow the run: first %+v, last %+v, queued %t", first, last, queued)
		}
		repeated := runDiagnosticFixture(t, binary, diagnosticRequest{diagnosticBytes: diagnosticFixtureBytes})
		if !slices.Equal(run.records, repeated.records) {
			t.Fatal("two same-seed runs recorded different digests")
		}
	})

	t.Run("recording leaves the run unchanged", func(t *testing.T) {
		off := runDiagnosticFixture(t, binary, diagnosticRequest{})
		on := runDiagnosticFixture(t, binary, diagnosticRequest{diagnosticBytes: diagnosticFixtureBytes})
		if off.exitCode != 0 || on.exitCode != 0 || len(off.choiceRecords) == 0 || len(on.records) == 0 {
			t.Fatalf("exit off = %d, on = %d, choice bytes = %d, digests = %d", off.exitCode, on.exitCode, len(off.choiceRecords), len(on.records))
		}
		if off.stdout != on.stdout || off.stderr != on.stderr {
			t.Fatalf("output moved with diagnostics on:\noff %q %q\non  %q %q", off.stdout, off.stderr, on.stdout, on.stderr)
		}
		if !bytes.Equal(off.choiceRecords, on.choiceRecords) || !bytes.Equal(off.terminal, on.terminal) {
			t.Fatal("choice trace bytes moved with diagnostics on")
		}
	})

	t.Run("perturbation adds one draw at the chosen ordinal", func(t *testing.T) {
		const ordinal = 5
		plain := runDiagnosticFixture(t, binary, diagnosticRequest{diagnosticBytes: diagnosticFixtureBytes})
		perturbed := runDiagnosticFixture(t, binary, diagnosticRequest{diagnosticBytes: diagnosticFixtureBytes, extraEnvironment: []string{"GOMAD3_DIAGNOSTIC_PERTURB_DRAW=" + strconv.Itoa(ordinal)}})
		if plain.exitCode != 0 || perturbed.exitCode != 0 || len(plain.records) <= ordinal || len(perturbed.records) <= ordinal {
			t.Fatalf("exit plain = %d, perturbed = %d, digests = %d and %d", plain.exitCode, perturbed.exitCode, len(plain.records), len(perturbed.records))
		}
		if !slices.Equal(plain.records[:ordinal], perturbed.records[:ordinal]) {
			t.Fatal("digests before the perturbed ordinal differ")
		}
		want := plain.records[ordinal]
		want.RuntimeCheapRandDraws++
		if perturbed.records[ordinal] != want {
			t.Fatalf("perturbed digest = %+v, want %+v", perturbed.records[ordinal], want)
		}
	})

	t.Run("overflow stops the target", func(t *testing.T) {
		const capacity = gomadchoicewire.DiagnosticHeaderBytes + 2*gomadchoicewire.DiagnosticRecordBytes
		run := runDiagnosticFixture(t, binary, diagnosticRequest{diagnosticBytes: capacity})
		if run.exitCode != 125 || !strings.Contains(run.stderr, "runtime: Gomad diagnostic trace overflow") || run.stdout != "" {
			t.Fatalf("exit = %d, stdout = %q, stderr = %q", run.exitCode, run.stdout, run.stderr)
		}
		if want := (gomadchoicewire.DiagnosticHeader{Capacity: capacity, NextOffset: capacity, RecordCount: 2, State: gomadchoicewire.DiagnosticOverflow}); run.header != want {
			t.Fatalf("header = %+v, want %+v", run.header, want)
		}
	})

	t.Run("choice trace overflow leaves the diagnostic trace marked truncated", func(t *testing.T) {
		const choiceCapacity = gomadchoicewire.HeaderBytes + 2*gomadchoicewire.RecordBytes
		run := runDiagnosticFixture(t, binary, diagnosticRequest{choiceBytes: choiceCapacity, diagnosticBytes: diagnosticFixtureBytes})
		terminal, err := gomadchoicewire.DecodeTerminal(run.terminal)
		if err != nil {
			t.Fatal(err)
		}
		if run.exitCode != 0 || terminal.State != gomadchoicewire.TerminalOverflow || terminal.Records != 2 {
			t.Fatalf("exit = %d, choice terminal = %+v, stderr = %q", run.exitCode, terminal, run.stderr)
		}
		if want := (gomadchoicewire.DiagnosticHeader{Capacity: diagnosticFixtureBytes, NextOffset: gomadchoicewire.DiagnosticHeaderBytes + 2*gomadchoicewire.DiagnosticRecordBytes, RecordCount: 2, State: gomadchoicewire.DiagnosticOverflow}); run.header != want {
			t.Fatalf("header = %+v, want %+v", run.header, want)
		}
	})

	t.Run("invalid configuration stops before user code", func(t *testing.T) {
		for name, request := range map[string]diagnosticRequest{
			"no choice trace":                {diagnosticBytes: diagnosticFixtureBytes, withoutChoiceTrace: true},
			"no byte bound":                  {diagnosticBytes: diagnosticFixtureBytes, withoutByteBound: true},
			"corrupt header":                 {diagnosticBytes: diagnosticFixtureBytes, corruptHeader: true},
			"malformed perturbation ordinal": {diagnosticBytes: diagnosticFixtureBytes, extraEnvironment: []string{"GOMAD3_DIAGNOSTIC_PERTURB_DRAW=first"}},
			"perturbation without a trace":   {extraEnvironment: []string{"GOMAD3_DIAGNOSTIC_PERTURB_DRAW=1"}},
		} {
			request.wantConfiguration = true
			run := runDiagnosticFixture(t, binary, request)
			if run.exitCode != 2 || !strings.HasPrefix(run.stderr, "runtime: invalid Gomad diagnostic trace") || run.stdout != "" {
				t.Errorf("%s: exit = %d, stdout = %q, stderr = %q", name, run.exitCode, run.stdout, run.stderr)
			}
		}
	})
}

func runDiagnosticFixture(t *testing.T, binary string, request diagnosticRequest) diagnosticResult {
	t.Helper()
	directory := t.TempDir()
	command := exec.Command(binary)
	command.Env = []string{"GOMADSEED=7"}
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr

	var trace, diagnostic, terminalReader, terminalWriter *os.File
	if !request.withoutChoiceTrace {
		choiceBytes := uint64(diagnosticFixtureChoiceBytes)
		if request.choiceBytes != 0 {
			choiceBytes = request.choiceBytes
		}
		header := gomadchoicewire.EncodeHeader(choiceBytes)
		trace = diagnosticBacking(t, filepath.Join(directory, "choices"), header[:], choiceBytes)
		reader, writer, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		terminalReader, terminalWriter = reader, writer
		t.Cleanup(func() { reader.Close() })
		command.Env = append(command.Env,
			fmt.Sprintf("GOMAD3_CHOICE_TRACE_FD=%d", 3+len(command.ExtraFiles)), fmt.Sprintf("GOMAD3_CHOICE_TERMINAL_FD=%d", 4+len(command.ExtraFiles)),
			fmt.Sprintf("GOMAD3_CHOICE_TRACE_BYTES=%d", choiceBytes), fmt.Sprintf("GOMAD3_CHOICE_MODE=%d", gomadchoicewire.ModeRecord))
		command.ExtraFiles = append(command.ExtraFiles, trace, writer)
	}
	if request.diagnosticBytes != 0 {
		header := gomadchoicewire.EncodeDiagnosticHeader(request.diagnosticBytes)
		if request.corruptHeader {
			header[0]++
		}
		diagnostic = diagnosticBacking(t, filepath.Join(directory, "diagnostics"), header[:], request.diagnosticBytes)
		command.Env = append(command.Env, fmt.Sprintf("GOMAD3_DIAGNOSTIC_TRACE_FD=%d", 3+len(command.ExtraFiles)))
		if !request.withoutByteBound {
			command.Env = append(command.Env, fmt.Sprintf("GOMAD3_DIAGNOSTIC_TRACE_BYTES=%d", request.diagnosticBytes))
		}
		command.ExtraFiles = append(command.ExtraFiles, diagnostic)
	}
	command.Env = append(command.Env, request.extraEnvironment...)

	err := command.Run()
	if terminalWriter != nil {
		if closeErr := terminalWriter.Close(); closeErr != nil {
			t.Fatal(closeErr)
		}
	}
	if err != nil && command.ProcessState == nil {
		t.Fatal(err)
	}
	result := diagnosticResult{exitCode: command.ProcessState.ExitCode(), stdout: stdout.String(), stderr: stderr.String()}
	if request.wantConfiguration {
		return result
	}
	if trace != nil {
		written := diagnosticContents(t, trace)
		header, err := gomadchoicewire.DecodeHeader(written[:gomadchoicewire.HeaderBytes])
		if err != nil {
			t.Fatal(err)
		}
		result.choiceRecords = written[gomadchoicewire.HeaderBytes:header.NextOffset]
		if result.terminal, err = io.ReadAll(terminalReader); err != nil {
			t.Fatal(err)
		}
	}
	if diagnostic != nil {
		written := diagnosticContents(t, diagnostic)
		header, err := gomadchoicewire.DecodeDiagnosticHeader(written[:gomadchoicewire.DiagnosticHeaderBytes])
		if err != nil {
			t.Fatal(err)
		}
		result.header = header
		for offset := uint64(gomadchoicewire.DiagnosticHeaderBytes); offset < header.NextOffset; offset += gomadchoicewire.DiagnosticRecordBytes {
			record, err := gomadchoicewire.DecodeDiagnosticRecord(written[offset : offset+gomadchoicewire.DiagnosticRecordBytes])
			if err != nil {
				t.Fatal(err)
			}
			result.records = append(result.records, record)
		}
	}
	return result
}

func diagnosticBacking(t *testing.T, path string, header []byte, capacity uint64) *os.File {
	t.Helper()
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { file.Close() })
	if _, err := file.Write(header); err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(int64(capacity)); err != nil {
		t.Fatal(err)
	}
	return file
}

func diagnosticContents(t *testing.T, file *os.File) []byte {
	t.Helper()
	contents, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	return contents
}

func diagnosticFixtureMallocs(t *testing.T, stdout string) uint64 {
	t.Helper()
	_, value, found := strings.Cut(strings.TrimSpace(stdout), "mallocs ")
	if !found {
		t.Fatalf("fixture output has no allocation count: %q", stdout)
	}
	mallocs, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	return mallocs
}
