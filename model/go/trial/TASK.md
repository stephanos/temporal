# Agent trial: add a heartbeat timeout to the standalone activity Model

One task, identical for every side except the file paths and the check command. An agent gets the
repository, this file, and its side's check command, and nothing else. Each run starts in a fresh
session.

## The task

Add a heartbeat timeout to the standalone activity Model.

1. Add a `heartbeat` action class the worker performs on a started attempt.
2. Add a `heartbeat` timer the system owns. It fires only on a started attempt whose start request
   set a heartbeat timeout; add the state field that records whether the start set one, and a start
   input for it.
3. Add the rows for both: a heartbeat keeps the attempt started and records nothing new; the timer
   times the attempt out and records the timed-out status with the heartbeat timeout type.
4. Add a Property that the heartbeat timer times the attempt out with the heartbeat timeout type
   recorded, a Scenario that reaches it, and a `find` Query for it.
5. Add pins for the new rows.
6. Run the check command until it passes. Update any existing pin whose number the new field or
   rows change, and say which ones and why.

## Per side

| Side | Files | Check command |
| --- | --- | --- |
| Go | `model/go/standaloneactivity/`, `model/go/views/testdata/` | `model/go/run.sh` |
| Scala | `model/scala/src/standaloneactivity/`, `model/scala/src/test/StandaloneActivity*.test.scala`, `model/scala/goldens/views/`, `model/scala/src/test/CheckedViews.test.scala` | `model/scala/run.sh --no-prove` |
| Lean | `.plans/cmp/lean/StandaloneActivity.lean`, `.plans/cmp/lean/ActivityPins.lean` | none: see below |

The new rows change the activity's generated views, which each side checks against goldens. Go
regenerates them with `go run ./model/go/views/cmd/render -out model/go/views/testdata`; Scala with
`cd model/scala && ./scala.sh run src --main-class views.renderViews -- goldens/views`. Scala also
compares its views with the Go views line by line; a line the new rows make differ is added to that
test's expected differences with its reason.

The Lean side cannot run as specified. The activity protocol machine already has 288 states, past
`Umpire.Command.elaborationBound` (256), so the Lean elaborator refuses it before the task begins,
and a heartbeat field doubles it. The sample file has also never compiled (its `case` block names a
missing realization). The report records this instead of a Lean run.

## What is recorded per run

Wall time, the number of check-command runs, the final diff, whether every check passed at the end,
and whether a reviewer accepts the diff without changes.
