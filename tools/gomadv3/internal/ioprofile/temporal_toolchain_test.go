package ioprofile

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/process"
	"go.temporal.io/server/tools/gomadv3/internal/target"
)

func TestProfileRunsUnchangedTemporalActivityBatchCancelSuite(t *testing.T) {
	if os.Getenv("GOMADV3_TEMPORAL_QUALIFY") != "1" {
		t.Skip("set GOMADV3_TEMPORAL_QUALIFY=1 to build and run the unchanged Temporal suite")
	}
	toolchainRoot, err := filepath.Abs(filepath.Join("..", "..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	moduleCacheBytes, err := exec.Command(filepath.Join(toolchainRoot, "bin", "go"), "env", "GOMODCACHE").Output()
	if err != nil {
		t.Fatal(err)
	}
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	profile, err := Resolve(TemporalActivityAPIBatchCancel)
	if err != nil {
		t.Fatal(err)
	}
	spec, _, err := profile.PrepareBuildOverlay(target.Spec{
		Kind: target.KindGoTest, Source: "./tests", WorkingDir: repositoryRoot,
		Args: []string{targetArgument}, BuildTags: []string{"test_dep"}, PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot,
	}, strings.TrimSpace(string(moduleCacheBytes)))
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := target.Prepare(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	frame, err := profile.BootstrapFrame(prepared, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 7)
	if err != nil {
		t.Fatal(err)
	}
	results := make([]process.Result, 2)
	for index := range results {
		results[index], err = process.Run(context.Background(), process.Request{
			SupervisorCommand: []string{os.Args[0], "-test.run=TestEntropySupervisorHelper"},
			BootstrapCommand:  []string{os.Args[0], "-test.run=TestEntropyBootstrapHelper"},
			Command:           prepared.Path, Argv0: prepared.Argv[0], Args: prepared.Argv[1:], Dir: t.TempDir(), Env: []string{"GOMADV3_IO_PROFILE=" + profile.Name, "GOMADSEED=7", "TZ=UTC"},
			RunTimeout: 2 * time.Minute, TerminateGrace: 2 * time.Second, OutputLimit: 1 << 20,
			WorldRecordLimit: 1 << 20, WorldTransitionLimit: 1 << 20, WorldSeed: 7, IOConfig: frame,
			IOTranscriptLimit: 64 << 20,
		})
		if err != nil {
			t.Fatal(err)
		}
		if results[index].Termination != process.TerminationExit || results[index].ExitCode != 0 {
			t.Fatalf("result = %#v\nstdout:\n%s\nstderr:\n%s", results[index], results[index].Stdout.Bytes, results[index].Stderr.Bytes)
		}
	}
	if !bytes.Equal(results[0].IOTranscript.Bytes, results[1].IOTranscript.Bytes) {
		t.Fatalf("same-seed I/O transcripts differ: first=%d/%x second=%d/%x; %s", results[0].IOTranscript.Records, results[0].IOTranscript.SHA256, results[1].IOTranscript.Records, results[1].IOTranscript.SHA256, firstTranscriptDifference(results[0].IOTranscript.Bytes, results[1].IOTranscript.Bytes))
	}
}

func firstTranscriptDifference(first, second []byte) string {
	const recordBytes = 128
	limit := min(len(first), len(second)) / recordBytes
	for index := 0; index < limit; index++ {
		left := first[index*recordBytes : (index+1)*recordBytes]
		right := second[index*recordBytes : (index+1)*recordBytes]
		if !bytes.Equal(left, right) {
			return fmt.Sprintf("record %d: %s/%d != %s/%d", index, transcriptOperation(left), binary.BigEndian.Uint64(left[96:104]), transcriptOperation(right), binary.BigEndian.Uint64(right[96:104]))
		}
	}
	return fmt.Sprintf("common records=%d; first tail=%s; second tail=%s", limit, transcriptTail(first[limit*recordBytes:]), transcriptTail(second[limit*recordBytes:]))
}

func transcriptTail(data []byte) string {
	const recordBytes = 128
	values := make([]string, 0, len(data)/recordBytes)
	for len(data) >= recordBytes {
		values = append(values, fmt.Sprintf("%s/%d", transcriptOperation(data[:recordBytes]), binary.BigEndian.Uint64(data[96:104])))
		data = data[recordBytes:]
	}
	return strings.Join(values, ",")
}

func transcriptOperation(record []byte) string {
	length := int(binary.BigEndian.Uint16(record[8:10]))
	return string(record[10 : 10+length])
}
