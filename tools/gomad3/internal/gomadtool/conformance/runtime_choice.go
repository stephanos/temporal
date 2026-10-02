package conformance

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad3/choice"
)

const (
	choiceTraceBytes            = 1 << 20
	choiceTraceHeaderBytes      = 64
	choiceTraceNextOffset       = 24
	choiceReplaySelectDecisions = 6
)

var choiceReplayTranscriptPattern = regexp.MustCompile(`^[LR]{6}$`)

type choiceRun struct {
	transcript string
	trace      choice.Trace
	terminal   []byte
}

// requireChoiceReplay records the fixture's Choice Trace under one seed and
// replays the Decision Tape projected from it under a seed that selects
// differently on its own, so an equal transcript can only come from the tape.
func (campaign *runtimeCampaign) requireChoiceReplay(fixture string) error {
	identity, err := campaign.choiceIdentity(fixture)
	if err != nil {
		return err
	}
	recorded, err := campaign.runChoice("choice-replay-record-seed-1", fixture, "1", nil)
	if err != nil {
		return err
	}
	if !choiceReplayTranscriptPattern.MatchString(recorded.transcript) {
		return fmt.Errorf("choice replay fixture transcript is malformed: %q", recorded.transcript)
	}
	plan, err := choice.ProjectReplayPlan(recorded.trace, identity)
	if err != nil {
		return fmt.Errorf("project choice replay fixture tape: %w", err)
	}
	if len(plan.Decisions) < choiceReplaySelectDecisions {
		return fmt.Errorf("choice replay fixture recorded %d branching decisions for %d two-way selects", len(plan.Decisions), choiceReplaySelectDecisions)
	}
	var control string
	for seed := 2; seed <= 9 && control == ""; seed++ {
		value := strconv.Itoa(seed)
		unforced, err := campaign.runChoice("choice-replay-control-seed-"+value, fixture, value, nil)
		if err != nil {
			return err
		}
		if unforced.transcript != recorded.transcript {
			control = value
		}
	}
	if control == "" {
		return errors.New("no control seed selected differently from the recorded choice replay fixture")
	}
	replayed, err := campaign.runChoice("choice-replay-tape-seed-"+control, fixture, control, &plan)
	if err != nil {
		return err
	}
	if replayed.transcript != recorded.transcript {
		return campaign.repeatabilityMismatch("choice tape did not reproduce the recorded transcript under seed "+control, recorded.transcript, replayed.transcript)
	}
	observed, err := choice.ProjectReplayPlan(replayed.trace, identity)
	if err != nil {
		return fmt.Errorf("project replayed choice trace: %w", err)
	}
	if !slices.Equal(observed.Decisions, plan.Decisions) {
		return fmt.Errorf("choice tape replay under seed %s recorded different decisions than the tape", control)
	}
	return nil
}

func (campaign *runtimeCampaign) choiceIdentity(fixture string) (choice.ExecutionIdentity, error) {
	target, err := os.ReadFile(fixture)
	if err != nil {
		return choice.ExecutionIdentity{}, fmt.Errorf("read choice replay fixture: %w", err)
	}
	key, err := os.ReadFile(filepath.Join(campaign.config.Root, ".toolchain", "build-key"))
	if err != nil {
		return choice.ExecutionIdentity{}, fmt.Errorf("read gomad3 build key: %w", err)
	}
	identity := choice.ExecutionIdentity{
		TargetSHA256: sha256.Sum256(target), ToolchainBuildKey: strings.TrimSpace(string(key)), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH,
	}
	identity.ImplementationSHA256, err = choice.ImplementationIdentity(identity.ToolchainBuildKey)
	if err != nil {
		return choice.ExecutionIdentity{}, err
	}
	return identity, nil
}

// runChoice runs the fixture with a Choice Trace and, when a tape is given,
// in exact replay mode. hostexec passes no extra descriptors, so the shell
// opens the trace backing, the terminal frame, and the tape and replaces
// itself with the fixture.
func (campaign *runtimeCampaign) runChoice(name, fixture, seed string, tape *choice.ReplayPlan) (choiceRun, error) {
	mode := choice.ModeRecord
	if tape != nil {
		mode = choice.ModeReplay
	}
	return campaign.runChoiceMode(name, fixture, seed, tape, mode, 0)
}

func (campaign *runtimeCampaign) runChoiceMode(name, fixture, seed string, tape *choice.ReplayPlan, mode choice.Mode, wantExit int, fixtureArgs ...string) (choiceRun, error) {
	run, _, err := campaign.runChoiceAccepting(name, fixture, seed, tape, mode, wantExit, nil, fixtureArgs...)
	return run, err
}

// runChoiceAccepting also passes the statuses in acceptExits and reports the
// status observed, for an experiment whose outcome is evidence rather than a
// requirement.
func (campaign *runtimeCampaign) runChoiceAccepting(name, fixture, seed string, tape *choice.ReplayPlan, mode choice.Mode, wantExit int, acceptExits []int, fixtureArgs ...string) (choiceRun, int, error) {
	directory := filepath.Join(campaign.workspace, name)
	if err := os.Mkdir(directory, 0o700); err != nil {
		return choiceRun{}, 0, fmt.Errorf("create choice trace directory: %w", err)
	}
	tracePath, terminalPath := filepath.Join(directory, "trace"), filepath.Join(directory, "terminal")
	if err := os.WriteFile(tracePath, choiceTraceBacking(choiceTraceBytes), 0o600); err != nil {
		return choiceRun{}, 0, fmt.Errorf("write choice trace backing: %w", err)
	}
	if err := os.Truncate(tracePath, choiceTraceBytes); err != nil {
		return choiceRun{}, 0, fmt.Errorf("size choice trace backing: %w", err)
	}
	script, arguments := `exec "$0" 3<>"$1" 4>"$2"`, []string{fixture, tracePath, terminalPath}
	values := []string{
		"GOMADSEED=" + seed, "GOMAD3_CHOICE_TRACE_FD=3", "GOMAD3_CHOICE_TERMINAL_FD=4",
		"GOMAD3_CHOICE_TRACE_BYTES=" + strconv.Itoa(choiceTraceBytes),
	}
	if tape != nil {
		tapePath := filepath.Join(directory, "tape")
		if err := os.WriteFile(tapePath, tape.Bytes, 0o400); err != nil {
			return choiceRun{}, 0, fmt.Errorf("write choice tape: %w", err)
		}
		script, arguments = script+` 5<"$3"`, append(arguments, tapePath)
		values = append(values, "GOMAD3_CHOICE_TAPE_FD=5", "GOMAD3_CHOICE_TAPE_BYTES="+strconv.Itoa(len(tape.Bytes)))
	}
	values = append(values, "GOMAD3_CHOICE_MODE="+strconv.Itoa(int(mode)))
	for index := range fixtureArgs {
		script += fmt.Sprintf(` "${%d}"`, len(arguments)+index)
	}
	arguments = append(arguments, fixtureArgs...)
	result, err := campaign.runCase(runtimeCase{name: name, wantExit: wantExit, acceptExits: acceptExits, request: campaign.request(
		append([]string{"/bin/sh", "-c", script}, arguments...), campaign.testdata, 10*time.Second,
		[]string{"GOMADSEED", "GOMAD3_IO_PROFILE", "GOMAD3_CHOICE_TRACE_FD", "GOMAD3_CHOICE_TERMINAL_FD", "GOMAD3_CHOICE_TRACE_BYTES", "GOMAD3_CHOICE_MODE", "GOMAD3_CHOICE_TAPE_FD", "GOMAD3_CHOICE_TAPE_BYTES"},
		values...,
	)})
	if err != nil {
		return choiceRun{}, result.ExitCode, err
	}
	backing, err := os.ReadFile(tracePath)
	if err != nil {
		return choiceRun{}, result.ExitCode, fmt.Errorf("read choice trace backing: %w", err)
	}
	terminal, err := os.ReadFile(terminalPath)
	if err != nil {
		return choiceRun{}, result.ExitCode, fmt.Errorf("read choice terminal frame: %w", err)
	}
	if len(backing) != choiceTraceBytes {
		return choiceRun{}, result.ExitCode, fmt.Errorf("%s resized its choice trace backing to %d bytes", name, len(backing))
	}
	next := binary.BigEndian.Uint64(backing[choiceTraceNextOffset : choiceTraceNextOffset+8])
	if next < choiceTraceHeaderBytes || next > choiceTraceBytes {
		return choiceRun{}, result.ExitCode, fmt.Errorf("%s published choice trace offset %d", name, next)
	}
	trace, err := choice.DecodeTrace(backing[choiceTraceHeaderBytes:next], terminal, choiceTraceBytes)
	if err != nil && !(result.ExitCode == 125 && errors.Is(err, choice.ErrDiverged)) {
		return choiceRun{}, result.ExitCode, fmt.Errorf("decode %s choice trace: %w", name, err)
	}
	return choiceRun{transcript: commandOutput(result), trace: trace, terminal: terminal}, result.ExitCode, nil
}

// choiceTraceBacking builds the empty v2 trace header the runtime maps: the
// magic, the wire version, the capacity, and a next offset just past the header.
func choiceTraceBacking(capacity uint64) []byte {
	header := make([]byte, choiceTraceHeaderBytes)
	copy(header, "GOMADCH\x02")
	binary.BigEndian.PutUint32(header[8:12], 2)
	binary.BigEndian.PutUint64(header[16:24], capacity)
	binary.BigEndian.PutUint64(header[choiceTraceNextOffset:choiceTraceNextOffset+8], choiceTraceHeaderBytes)
	return header
}
