// Package child connects an explicitly modeled World to the Runner's inherited
// recording pipe. Applications finish the session only after modeled work has
// stopped, so transport readiness never participates in event ordering.
package child

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"go.temporal.io/server/tools/gomadv3/internal/worldpipe"
	"go.temporal.io/server/tools/gomadv3/world"
)

const (
	configFD = 3
	recordFD = 4
)

type Session struct {
	recorder *world.Recorder
	output   *os.File
	core     *world.World
}

func Open(core *world.World) (*Session, error) {
	if core == nil {
		return nil, fmt.Errorf("World child session requires a World")
	}
	config := os.NewFile(configFD, "gomadv3-world-config")
	output := os.NewFile(recordFD, "gomadv3-world-record")
	if config == nil || output == nil {
		var cleanupErr error
		if config != nil {
			cleanupErr = errors.Join(cleanupErr, config.Close())
		}
		if output != nil {
			cleanupErr = errors.Join(cleanupErr, output.Close())
		}
		return nil, errors.Join(fmt.Errorf("World child file descriptors are unavailable"), cleanupErr)
	}
	header := world.RecordingHeader()
	if err := writeAll(output, header[:]); err != nil {
		return nil, errors.Join(err, config.Close(), output.Close())
	}
	childConfig, err := worldpipe.Read(config)
	closeErr := config.Close()
	if err != nil || closeErr != nil {
		return nil, errors.Join(err, closeErr, output.Close())
	}
	seedText, found := os.LookupEnv("GOMADSEED")
	seed, err := strconv.ParseUint(seedText, 10, 64)
	if !found || err != nil || strconv.FormatUint(seed, 10) != seedText || seed != childConfig.Seed {
		return nil, errors.Join(fmt.Errorf("World seed does not match the canonical GOMADSEED input"), output.Close())
	}
	if len(childConfig.ExpectedInitial) != 0 {
		initial, decodeErr := world.DecodeSnapshot(childConfig.ExpectedInitial)
		if decodeErr != nil {
			return nil, errors.Join(fmt.Errorf("decode trusted replay initial World snapshot: %w", decodeErr), output.Close())
		}
		restored, restoreErr := world.Restore(initial, nil)
		if restoreErr != nil {
			return nil, errors.Join(fmt.Errorf("restore trusted replay initial World snapshot: %w", restoreErr), output.Close())
		}
		if uint64(initial.Config.Seed) != seed {
			return nil, errors.Join(fmt.Errorf("trusted replay initial World snapshot seed does not match GOMADSEED"), output.Close())
		}
		core = restored
	} else if world.Seed(seed) != core.Seed() {
		return nil, errors.Join(fmt.Errorf("World seed does not match the canonical GOMADSEED input"), output.Close())
	}
	recorder, err := core.StartRecording(childConfig.TransitionLimit)
	if err != nil {
		return nil, errors.Join(err, output.Close())
	}
	return &Session{recorder: recorder, output: output, core: core}, nil
}

func (session *Session) World() *world.World {
	if session == nil {
		return nil
	}
	return session.core
}

func (session *Session) Finish() error {
	return session.finish(nil)
}

func (session *Session) FinishError(terminalErr error) error {
	if terminalErr == nil {
		return fmt.Errorf("World terminal error is required")
	}
	return session.finish(terminalErr)
}

func (session *Session) finish(terminalErr error) error {
	if session == nil || session.recorder == nil || session.output == nil {
		return fmt.Errorf("World child session is invalid")
	}
	var recording world.Recording
	var err error
	if terminalErr == nil {
		recording, err = session.recorder.Finish()
	} else {
		recording, err = session.recorder.FinishError(terminalErr)
	}
	if err != nil {
		return errors.Join(err, session.output.Close())
	}
	encoded, err := world.EncodeRecording(recording)
	if err != nil {
		return errors.Join(err, session.output.Close())
	}
	header := world.RecordingHeader()
	if len(encoded) < len(header) {
		return errors.Join(fmt.Errorf("World recording omitted its header"), session.output.Close())
	}
	if err := writeAll(session.output, encoded[len(header):]); err != nil {
		return errors.Join(err, session.output.Close())
	}
	err = session.output.Close()
	session.recorder = nil
	session.output = nil
	session.core = nil
	return err
}

func writeAll(output *os.File, data []byte) error {
	for len(data) > 0 {
		written, err := output.Write(data)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrNoProgress
		}
		data = data[written:]
	}
	return nil
}
