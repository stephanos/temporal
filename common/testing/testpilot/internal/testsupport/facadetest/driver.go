// Package facadetest holds the test support that needs the public facade: a fake Driver and the
// prepared runtime fixture the Temporal Driver tests share. The facade's own in-package tests cannot
// import it.
package facadetest

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
)

// Driver is a fake Driver that reports DriverIdentity and records what each Run hands it. Validate
// runs OnValidate when set; Open runs OnOpen when set and otherwise opens an unscripted
// testsupport.Session.
type Driver struct {
	DriverIdentity testpilot.DriverIdentity
	OnValidate     func(context.Context, testpilot.PreparedProgram) error
	OnOpen         func(context.Context, string, testpilot.PreparedProgram) (testpilot.Session, error)

	mu        sync.Mutex
	validated int
	program   testpilot.PreparedProgram
	runIDs    []string
	sessions  []testpilot.Session
}

func (d *Driver) Identity(context.Context) (testpilot.DriverIdentity, error) {
	return d.DriverIdentity, nil
}

func (d *Driver) Validate(ctx context.Context, program testpilot.PreparedProgram) error {
	d.mu.Lock()
	d.validated++
	d.program = program
	d.mu.Unlock()
	if d.OnValidate != nil {
		return d.OnValidate(ctx, program)
	}
	return nil
}

func (d *Driver) Open(ctx context.Context, runID string, program testpilot.PreparedProgram) (testpilot.Session, error) {
	d.mu.Lock()
	d.program = program
	d.runIDs = append(d.runIDs, runID)
	d.mu.Unlock()
	var session testpilot.Session = &testsupport.Session{}
	if d.OnOpen != nil {
		var err error
		if session, err = d.OnOpen(ctx, runID, program); err != nil {
			return nil, err
		}
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.sessions = append(d.sessions, session)
	return session, nil
}

// Validated is how many times Validate was called.
func (d *Driver) Validated() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.validated
}

// Program is the Program the last Validate or Open received.
func (d *Driver) Program() testpilot.PreparedProgram {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.program
}

// RunIDs are the Run IDs Open received, in order.
func (d *Driver) RunIDs() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.runIDs...)
}

// Closed is how many opened testsupport Sessions have been closed.
func (d *Driver) Closed() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	closed := 0
	for _, session := range d.sessions {
		if scripted, ok := session.(*testsupport.Session); ok && scripted.Calls("Close") > 0 {
			closed++
		}
	}
	return closed
}

var errProgramCaptured = errors.New("prepared Program captured")

// Capture is the Program prepared hands its Driver, taken from a Run that stops at Open.
func Capture(t testing.TB, prepared *testpilot.PreparedCase) testpilot.PreparedProgram {
	t.Helper()
	driver := &Driver{DriverIdentity: prepared.Identity(), OnOpen: func(context.Context, string, testpilot.PreparedProgram) (testpilot.Session, error) {
		return nil, errProgramCaptured
	}}
	_, _, err := prepared.Run(t.Context(), driver)
	require.ErrorIs(t, err, errProgramCaptured)
	return driver.Program()
}

var _ testpilot.Driver = (*Driver)(nil)
