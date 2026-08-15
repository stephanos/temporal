package process

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/choicewire"
	"go.temporal.io/server/tools/gomadv3/internal/romount"
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

type Request struct {
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
	Mode                 choicewire.Mode
	Profile              string
	ImplementationSHA256 [sha256.Size]byte
	ExecutionIdentity    choicewire.ExecutionIdentity
	Limit                uint64
	Tape                 *choicewire.Tape
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

type IOTranscript struct {
	Bytes            []byte
	SHA256           [sha256.Size]byte
	Records          uint64
	Complete         bool
	ReplayDivergence *uint64
}

type ChoiceTrace struct {
	Profile              string
	ImplementationSHA256 [sha256.Size]byte
	Limit                uint64
	Trace                choicewire.Trace
	TapeSHA256           [sha256.Size]byte
	Decisions            uint64
}

func validateRequest(request Request) error {
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
	if choice := request.Choice; choice != nil {
		if choice.Profile != choicewire.Profile {
			return fmt.Errorf("unsupported choice trace profile %q", choice.Profile)
		}
		if choice.Limit < minimumChoiceTraceBytes || choice.Limit > maximumChoiceTraceBytes {
			return fmt.Errorf("invalid choice trace limit %d", choice.Limit)
		}
		if choice.ImplementationSHA256 == ([sha256.Size]byte{}) {
			return errors.New("choice trace implementation identity is required")
		}
		if choice.Mode != choicewire.ModeRecord && choice.Mode != choicewire.ModeReplay && choice.Mode != choicewire.ModePrefix {
			return errors.New("choice controller mode is invalid")
		}
		if choice.ExecutionIdentity.ImplementationSHA256 != ([sha256.Size]byte{}) && choice.ExecutionIdentity.ImplementationSHA256 != choice.ImplementationSHA256 {
			return errors.New("choice controller implementation identities disagree")
		}
		if choice.Mode == choicewire.ModeRecord {
			if choice.Tape != nil {
				return errors.New("choice record mode cannot include a decision tape")
			}
		} else {
			if choice.Tape == nil {
				return errors.New("choice replay and prefix modes require a decision tape")
			}
			if len(choice.Tape.Bytes) > maximumChoiceTapeBytes {
				return errors.New("choice decision tape exceeds its bound")
			}
			if _, err := validateChoiceTape(*choice.Tape, choice.ExecutionIdentity, choice.Mode); err != nil {
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
	if transcript.Limit > maximumIOTranscriptBytes {
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
	if !transcript.Replay && len(transcript.Expected) != 0 {
		return errors.New("expected I/O transcript requires replay mode")
	}
	if len(transcript.Expected)%ioTranscriptRecordBytes != 0 || uint64(len(transcript.Expected)) > transcript.Limit-ioTranscriptHeaderBytes {
		return fmt.Errorf("invalid expected I/O transcript length %d", len(transcript.Expected))
	}
	return nil
}

func validateChoiceTape(tape choicewire.Tape, identity choicewire.ExecutionIdentity, mode choicewire.Mode) (choicewire.Tape, error) {
	if mode == choicewire.ModePrefix {
		return choicewire.ValidatePrefixTape(tape, identity)
	}
	return choicewire.ValidateDecisionTape(tape, identity)
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
