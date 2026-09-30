package conformance

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/hostexec"
)

const (
	bootstrapFrameBytes     = 212
	bootstrapChecksumOffset = 180
	bootstrapSeedOffset     = 172
)

// requireBootstrapConsumer feeds bootstrap frames to the runtime's actual
// consumer on descriptor 5. The runtime recognizes only the header during early
// startup and exits before user initialization when it is wrong. A frame with a
// valid header passes that phase; without the Runner's other descriptors the
// process then stops in a later Gomad phase (the transcript mapping, or the full
// checksum and identity validation), still before user initialization.
func (campaign *runtimeCampaign) requireBootstrapConsumer(binary string) error {
	directory, err := os.MkdirTemp("", "gomad3-bootstrap-")
	if err != nil {
		return fmt.Errorf("create bootstrap frame directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(directory) }()

	earlyRejection := func(result hostexec.Result) error {
		if len(result.Stdout.RawBytes) != 0 || commandErrorOutput(result) != "runtime: invalid Gomad bootstrap configuration" {
			return errors.New("malformed bootstrap frame reached user initialization or emitted an unexpected diagnostic")
		}
		return nil
	}
	laterRejection := func(result hostexec.Result) error {
		stderr := string(result.Stderr.RawBytes)
		if len(result.Stdout.RawBytes) != 0 || strings.Contains(stderr, "invalid Gomad bootstrap configuration") || !strings.HasPrefix(stderr, "panic: gomad3: ") {
			return errors.New("bootstrap frame with a valid header was refused by the early phase or reached user initialization")
		}
		return nil
	}
	for _, test := range []struct {
		name   string
		frame  []byte
		oracle func(hostexec.Result) error
	}{
		{name: "truncated", frame: bootstrapFrame(0)[:bootstrapFrameBytes-1], oracle: earlyRejection},
		{name: "header-only", frame: bootstrapFrame(0)[:12], oracle: earlyRejection},
		{name: "wrong-magic", frame: mutateBootstrapFrame(0, 0, 'X'), oracle: earlyRejection},
		{name: "wrong-magic-terminator", frame: mutateBootstrapFrame(0, 7, 2), oracle: earlyRejection},
		{name: "wrong-version", frame: mutateBootstrapFrame(0, 9, 2), oracle: earlyRejection},
		{name: "wrong-version-high", frame: mutateBootstrapFrame(0, 8, 1), oracle: earlyRejection},
		{name: "wrong-kind", frame: mutateBootstrapFrame(0, 11, 2), oracle: earlyRejection},
		{name: "valid-header-seed-zero", frame: bootstrapFrame(0), oracle: laterRejection},
		{name: "valid-header-seed-max", frame: bootstrapFrame(^uint64(0)), oracle: laterRejection},
	} {
		path := filepath.Join(directory, test.name)
		if err := os.WriteFile(path, test.frame, 0o600); err != nil {
			return fmt.Errorf("write bootstrap frame %s: %w", test.name, err)
		}
		// hostexec passes no extra descriptors, so the shell opens the frame on
		// descriptor 5 and replaces itself with the fixture.
		if err := campaign.expectedExit(
			"bootstrap-"+test.name, []string{"/bin/sh", "-c", `exec "$0" 5<"$1"`, binary, path}, campaign.testdata, 10*time.Second, 2,
			test.oracle, []string{"GOMADSEED", "GOMAD3_IO_PROFILE"}, "GOMAD3_IO_PROFILE=deterministic",
		); err != nil {
			return err
		}
	}
	return nil
}

// bootstrapFrame builds a frame with a valid header and the given seed, and a
// zero checksum that full validation must reject.
func bootstrapFrame(seed uint64) []byte {
	frame := make([]byte, bootstrapFrameBytes)
	copy(frame, "GOMADIO\x01")
	binary.BigEndian.PutUint16(frame[8:10], 1)
	binary.BigEndian.PutUint16(frame[10:12], 1)
	binary.BigEndian.PutUint64(frame[bootstrapSeedOffset:bootstrapChecksumOffset], seed)
	return frame
}

func mutateBootstrapFrame(seed uint64, index int, value byte) []byte {
	frame := bootstrapFrame(seed)
	frame[index] = value
	return frame
}
