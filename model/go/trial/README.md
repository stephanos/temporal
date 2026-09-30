# Agent trial results

Task: [TASK.md](TASK.md). Six runs, 2026-09-30, each a fresh Claude Opus 5.5 agent session given
the task, its side's files and its check command. Runs of one side never overlapped a run of the
other side's files; a Scala run never overlapped a Go run, because Scala's views test reads the Go
goldens. Each run's diff is in `runs/`, and each was re-verified by running the check command on the
saved result before the tree was restored.

| Run | Wall time | Check runs | Final | Reviewer |
| --- | --- | --- | --- | --- |
| Go 1 | 188 s | 1 | pass | changes requested |
| Go 2 | 219 s | 1 | pass | changes requested |
| Go 3 | 212 s | 1 | pass | changes requested |
| Scala 1 | 264 s | 1 | pass | accepted, with a follow-up |
| Scala 2 | 292 s | 1 | pass | accepted, with a follow-up |
| Scala 3 | 280 s | 1 | pass | accepted, with a follow-up |
| Lean | not run | | | the activity protocol is past the 256-state bound |

Wall time runs from starting the agent to receiving its report, so it includes the agent reading the
Model before writing anything.

## What the runs had in common

- **Every run passed on its first check.** The agents read the existing Model, claims and pins, then
  wrote the change and the pin updates before running anything.
- **The four-input start split the sides.** The Go framework stops at three inputs, and every Go run
  folded the start-to-close and heartbeat deadlines into one `AttemptTimeouts` struct input. The
  class keys come out as a four-input action's would, but the action now declares an input the
  domain does not have, which a reviewer would send back with a request for `NewAction4` in the
  framework. Every Scala run instead declared a four-input `apply` and `~>` beside the Model, in
  about six lines, and kept `start` a true four-input action; the follow-up is to move those into
  the framework.
- **Every run renamed the timer `heartbeatTimeout`.** An action and a timer both named `heartbeat`
  would bind one class key. Four of the six agents said nothing checks this; both frameworks do
  reject it, with "two steps bind the action class".
- **"A started attempt" was read two ways.** Three runs let the heartbeat and its timer act on every
  attempt a worker holds (started, pause-requested, cancel-requested), as start-to-close does; three
  kept to started. Both readings are defensible; the reviews do not count it against a run.
- **No run changed the product machine.** Each read the heartbeat as a stutter and its timer as the
  product's `timeout`, which keeps the Lean parity dumps valid.
- **The Scala runs had one extra chore.** The heartbeat changes two activity views whose Go
  counterparts the run could not regenerate, so each Scala run exempted those two files from the
  line-by-line comparison with Go, with a reason.

The reviews are the experiment owner's own reading of the diffs, not an independent reviewer's.
