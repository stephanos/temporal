package ir

// The API behavior hints and server steps the lifter wrote from model/irgen/testdata/lifts/Hints.scala
// (fn-118.2): the realizations it admits are in TestLiftedModelsAreAdmitted; these it lifts and the
// reader refuses, each at the line of the Scala declaration concerned.

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLiftedHintsAreRefusedAtTheirLines(t *testing.T) {
	const at = "model/irgen/testdata/lifts/Hints.scala:"
	_, err := Load(filepath.Join("..", "..", "..", "model", "irgen", "testdata", "lifts", "expected", "hintsRefused.json"))
	require.Error(t, err)
	require.ElementsMatch(t, []string{
		at + "82: realization zeroInterval: cause bound cause.delivery looks every 0 milliseconds; an interval is positive",
		at + "93: realization nonPositiveBound: visibility visibility.activityAnswer.describeActivityExecution waits at most -1 milliseconds; a bound is positive",
		at + "96: realization nonPositiveBound: cause bound cause.timer waits at most 0 milliseconds; a bound is positive",
		at + "105: realization intervalOverBound: cause bound cause.timer looks every 2000 milliseconds and waits at most 1000; " +
			"an interval is no greater than its bound",
		at + "111: realization unboundedStep: server step poll is a delivery, and the realization bounds no delivery",
		at + "120: realization timerNoDeadline: server step scheduleToStart is a timer and names no positive deadline",
		at + "125: realization deliveryDeadline: server step poll names a deadline of 2000 milliseconds, and only a timer's step has one",
	}, strings.Split(err.Error(), "\n"))
}
