# Programmed workers

Design proposal, written 2026-10-23 after the owner deferred the fn-228.5 Shim. Status: proposal,
awaiting the owner's placement. The open questions below carry the conductor's recommendations,
taken under the owner's standing instruction to follow them when in doubt.

## Problem

R40 connects every SDK worker to the Model while it runs: in-process Go workers call Answer, and
workers in other languages reach it through a local Shim. The owner's direction (2026-10-23):
workers should know what to do *before* they execute, because in a cloud setup there is no
connection between a worker and Umpire. The Model still decides every worker response; it decides
earlier, and Umpire judges what happened afterwards.

## Proposal

**A worker program is a policy, not a script.** Runtime order is not predictable (retries,
timeouts, redelivery), so a program is a lookup keyed by what a worker can see when a task
arrives: activity type, activity id, attempt number and, where the Model needs it, a digest of the
input. Each entry names one action: complete with a result payload, fail (retryable or not, with a
failure payload), heartbeat n times with details, or delay. Task tokens and wall-clock values are
never keys. A task the program does not cover takes one fixed fallback (fail non-retryable with an
"unprogrammed" marker), and the Run reports it as unprogrammed, never as a verdict about the system.

**The Model issues the program with its proposal.** `ProposeResponse` gains an optional program
(an opaque `Any`, like the request): the Realization fills it from the World and the seed, for
every attempt its World admits. Determinism holds: the same state and seed give the same request
and the same program. Go stays dumb: the Driver hands the program to the worker and never reads it.

**Evidence is collected after the fact.** The primary source is the server's own history of the
activity (Describe and the activity's events), which needs no worker cooperation and works for
remote workers. A worker-side log of each task received and action taken is an optional
cross-check. The Runner feeds the evidence to the Model in order, as today's Observation and
Callback events. For each callback, the Model's Answer in the state reached at that point must
equal the programmed action; a mismatch is a Model/program inconsistency and refuses the Run.

**One path for all workers.** The in-process Go worker follows programs too, so the live Answer
path and the Shim disappear. Callbacks remain in the Run as recorded events, so replay and
reduction are unchanged in kind.

## Recommendations on the open questions

1. Evidence source: server history first, the worker log as an optional cross-check.
2. Program scope: one program per proposal; batch runs compose them by activity id.
3. The in-process Go worker switches to programs (one path), retiring live Answer for workers.

## Consequences

- R40 is rewritten: a worker in any language follows a Model-issued program and the Run records
  what happened. The Shim (fn-228.5, partial start on `work/fn2285-shim-deferred`) is superseded.
- The Temporal Driver's worker gate (fn-228.4) becomes "deliver the program, then observe"; the
  `worker_tasks` instruction may become unnecessary.
- Some Models may need answers that depend on observations a worker cannot see; those stay out of
  scope until a program key can express them, and the Realization refuses to program them.
