package local

import (
	"errors"

	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

type isolationDecision uint8

const (
	isolationDecisionFailed isolationDecision = iota
	isolationDecisionCanceled
	isolationDecisionReady
)

type isolationCollection struct {
	prepareCommand       umpireruntime.Command
	realizeCommand       umpireruntime.Command
	observeCommand       umpireruntime.Command
	operationCorrelation string

	operationRecorded bool
	operationCount    uint64
	controlRecorded   bool
	controlCount      uint64
	inputsClosed      bool
	invalid           bool
	isolationCalled   bool
}

func newIsolationCollection(
	prepareCommand umpireruntime.Command,
	realizeCommand umpireruntime.Command,
	observeCommand umpireruntime.Command,
	operationCorrelation string,
) isolationCollection {
	var emptyCommand umpireruntime.Command
	return isolationCollection{
		prepareCommand:       prepareCommand,
		realizeCommand:       realizeCommand,
		observeCommand:       observeCommand,
		operationCorrelation: operationCorrelation,
		invalid: prepareCommand == emptyCommand || realizeCommand == emptyCommand ||
			observeCommand == emptyCommand || operationCorrelation == "",
	}
}

func (c *isolationCollection) recordOperationCount(
	command umpireruntime.Command,
	operationCorrelation string,
	count uint64,
) error {
	if command != c.prepareCommand || operationCorrelation != c.operationCorrelation ||
		c.operationRecorded || c.inputsClosed {
		c.invalid = true
		return errors.New("unsupported isolation operation record")
	}
	c.operationRecorded = true
	c.operationCount = count
	return nil
}

func (c *isolationCollection) recordControlCount(
	command umpireruntime.Command,
	operationCorrelation string,
	count uint64,
) error {
	if command != c.realizeCommand || operationCorrelation != c.operationCorrelation ||
		c.controlRecorded || c.inputsClosed {
		c.invalid = true
		return errors.New("unsupported isolation control record")
	}
	c.controlRecorded = true
	c.controlCount = count
	return nil
}

func (c *isolationCollection) closeInputs(command umpireruntime.Command) error {
	if command != c.observeCommand || c.inputsClosed {
		c.invalid = true
		return errors.New("unsupported isolation collection close")
	}
	c.inputsClosed = true
	return nil
}

func (c *isolationCollection) decision() isolationDecision {
	if c.isolationCalled {
		c.invalid = true
	}
	c.isolationCalled = true
	if c.invalid || c.operationCount > 1 || c.controlCount > 1 {
		return isolationDecisionFailed
	}
	if !c.operationRecorded || c.operationCount != 1 ||
		!c.controlRecorded || c.controlCount != 1 || !c.inputsClosed {
		return isolationDecisionCanceled
	}
	return isolationDecisionReady
}
