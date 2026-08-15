package execution

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomadv3/choice"
	romount "go.temporal.io/server/tools/gomadv3/deterministicio"
)

type Termination string

const maximumIOConfigBytes = 4096

const choiceProfileEnvironmentName = "GOMADV3_CHOICE_PROFILE"
const choiceModeEnvironmentName = "GOMADV3_CHOICE_MODE"
const choiceTraceFDEnvironmentName = "GOMADV3_CHOICE_TRACE_FD"
const choiceTerminalFDEnvironmentName = "GOMADV3_CHOICE_TERMINAL_FD"
const choiceTraceBytesEnvironmentName = "GOMADV3_CHOICE_TRACE_BYTES"
const choiceTapeFDEnvironmentName = "GOMADV3_CHOICE_TAPE_FD"
const choiceTapeBytesEnvironmentName = "GOMADV3_CHOICE_TAPE_BYTES"

const (
	TerminationExit   Termination = "exit"
	TerminationSignal Termination = "signal"
)

type Spec struct {
	SupervisorCommand []string
	BootstrapCommand  []string
	Command           string
	Args              []string
	Argv0             string
	Dir               string
	Env               []string
	RunTimeout        time.Duration
	TerminateGrace    time.Duration
	OutputLimit       uint64
	StdoutHead        io.Writer
	StderrHead        io.Writer
	World             WorldCapability
	IO                *IOCapability
	Choice            *ChoiceCapability
}

type WorldCapability struct {
	RecordLimit     uint64
	TransitionLimit uint64
	Seed            uint64
	ExpectedInitial []byte
	ReplayPlan      []byte
}

type IOCapability struct {
	Config        []byte
	Transcript    *IOTranscriptCapability
	ReadOnlyMount *ReadOnlyMountCapability
}

type IOTranscriptCapability struct {
	Limit    uint64
	Replay   bool
	Expected []byte
}

type ReadOnlyMountCapability struct {
	Mappings []romount.Mapping
	Limits   romount.Limits
	Replay   *romount.Snapshot
}

type ChoiceCapability struct {
	Mode                 choice.Mode
	Profile              string
	ImplementationSHA256 [sha256.Size]byte
	ExecutionIdentity    choice.ExecutionIdentity
	Limit                uint64
	ReplayPlan           *choice.ReplayPlan
}

type Result struct {
	Captured        bool
	Termination     Termination
	ExitCode        int
	Signal          string
	WatchdogTimeout bool
	Cancelled       bool
	Stdout          Output
	Stderr          Output
	PID             int
	PGID            int
	GroupGone       bool
	WorldRecord     []byte
	IOTranscript    IOTranscript
	IOROMounts      romount.Snapshot
	ChoiceTrace     ChoiceTrace
}

type IOTranscript = romount.Transcript

type ChoiceTrace struct {
	Profile              string
	ImplementationSHA256 [sha256.Size]byte
	Limit                uint64
	Trace                choice.Trace
	TapeSHA256           [sha256.Size]byte
	Decisions            uint64
}

func validateSpec(request Spec) error {
	if len(request.SupervisorCommand) == 0 || request.SupervisorCommand[0] == "" {
		return fmt.Errorf("supervisor command is required")
	}
	if len(request.BootstrapCommand) == 0 || request.BootstrapCommand[0] == "" {
		return fmt.Errorf("target bootstrap command is required")
	}
	if request.Command == "" {
		return fmt.Errorf("target command is required")
	}
	if request.Argv0 == "" {
		return fmt.Errorf("target argv[0] is required")
	}
	if request.Dir == "" {
		return fmt.Errorf("target working directory is required")
	}
	if request.RunTimeout <= 0 {
		return fmt.Errorf("run timeout must be positive")
	}
	if request.TerminateGrace < 0 {
		return fmt.Errorf("termination grace must not be negative")
	}
	if request.OutputLimit == 0 {
		return fmt.Errorf("output limit must be positive")
	}
	if request.World.RecordLimit == 0 || request.World.TransitionLimit == 0 {
		return fmt.Errorf("World record and transition limits must be positive")
	}
	if len(request.World.ReplayPlan) != 0 && len(request.World.ExpectedInitial) == 0 {
		return errors.New("world replay plan requires an expected initial snapshot")
	}
	if err := validateChoiceEnvironment(request.Env); err != nil {
		return err
	}
	if choiceCapability := request.Choice; choiceCapability != nil {
		if choiceCapability.Profile != choice.Profile {
			return fmt.Errorf("unsupported choice trace profile %q", choiceCapability.Profile)
		}
		if choice.ValidateTraceLimit(choiceCapability.Limit) != nil {
			return fmt.Errorf("invalid choice trace limit %d", choiceCapability.Limit)
		}
		if choiceCapability.ImplementationSHA256 == ([sha256.Size]byte{}) {
			return errors.New("choice trace implementation identity is required")
		}
		if choiceCapability.Mode != choice.ModeRecord && choiceCapability.Mode != choice.ModeReplay && choiceCapability.Mode != choice.ModePrefix {
			return errors.New("choice controller mode is invalid")
		}
		if choiceCapability.ExecutionIdentity.ImplementationSHA256 != ([sha256.Size]byte{}) && choiceCapability.ExecutionIdentity.ImplementationSHA256 != choiceCapability.ImplementationSHA256 {
			return errors.New("choice controller implementation identities disagree")
		}
		if choiceCapability.Mode == choice.ModeRecord {
			if choiceCapability.ReplayPlan != nil {
				return errors.New("choice record mode cannot include a decision tape")
			}
		} else {
			if choiceCapability.ReplayPlan == nil {
				return errors.New("choice replay and prefix modes require a decision tape")
			}
			if len(choiceCapability.ReplayPlan.Bytes) > MaximumChoiceReplayPlanBytes {
				return errors.New("choice decision tape exceeds its bound")
			}
			if _, err := validateChoiceReplayPlan(*choiceCapability.ReplayPlan, choiceCapability.ExecutionIdentity, choiceCapability.Mode); err != nil {
				return fmt.Errorf("validate choice decision tape: %w", err)
			}
		}
	}
	if request.IO == nil {
		return nil
	}
	ioCapability := request.IO
	if len(ioCapability.Config) > maximumIOConfigBytes {
		return errors.New("I/O configuration exceeds its bound")
	}
	if ioCapability.Transcript == nil {
		if ioCapability.ReadOnlyMount != nil {
			return errors.New("read-only mount broker requires a deterministic I/O transcript")
		}
		return nil
	}
	transcript := ioCapability.Transcript
	if len(ioCapability.Config) == 0 && transcript.Limit != 0 {
		return errors.New("I/O transcript limit requires an I/O configuration")
	}
	if transcript.Limit > romount.MaximumTranscriptBytes {
		return errors.New("I/O transcript limit exceeds its bound")
	}
	if transcript.Replay && transcript.Limit == 0 {
		return errors.New("I/O replay requires a transcript")
	}
	if mounts := ioCapability.ReadOnlyMount; mounts != nil {
		if len(ioCapability.Config) == 0 || transcript.Limit == 0 {
			return errors.New("read-only mount broker requires a deterministic I/O transcript")
		}
		if mounts.Limits == (romount.Limits{}) {
			return errors.New("read-only mount broker requires limits")
		}
	}
	if err := romount.ValidateSessionSpec(romount.SessionSpec{Limit: transcript.Limit, Replay: transcript.Replay, Expected: transcript.Expected}); err != nil {
		return err
	}
	return nil
}

func validateChoiceReplayPlan(tape choice.ReplayPlan, identity choice.ExecutionIdentity, mode choice.Mode) (choice.ReplayPlan, error) {
	if mode == choice.ModePrefix {
		return choice.ValidatePrefixReplayPlan(tape, identity)
	}
	return choice.ValidateReplayPlan(tape, identity)
}

func validateChoiceEnvironment(environment []string) error {
	for _, entry := range environment {
		name, _, _ := strings.Cut(entry, "=")
		if name == choiceProfileEnvironmentName || name == choiceModeEnvironmentName || name == choiceTraceFDEnvironmentName || name == choiceTerminalFDEnvironmentName || name == choiceTraceBytesEnvironmentName || name == choiceTapeFDEnvironmentName || name == choiceTapeBytesEnvironmentName {
			return fmt.Errorf("target environment name %q is reserved", name)
		}
	}
	return nil
}

func effectiveTimeout(ctx context.Context, configured time.Duration) (time.Duration, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return configured, nil
	}
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return 0, context.DeadlineExceeded
	}
	if remaining < configured {
		return remaining, nil
	}
	return configured, nil
}
