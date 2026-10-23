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

**The worker's actions are the evidence.** There are no worker logs. Every worker action reaches
the server as an RPC (completion with its result, failure with its details, heartbeats) or as the
timeout its silence causes, so the server's history of the activity (Describe and the activity's
events) shows what the worker did. The Runner feeds that history to the Model in order, as
Observation events. The Model judges it against the program it issued: a worker that completes
where the program said fail, or heartbeats with other details, is an **Unexplained** observation.
The worker and its SDK are part of the system under test, so a deviation is a finding about the
system, never a refusal.

**Refusals stay for Umpire's own faults.** The issued program must equal what the Model would
answer in each state it admits; that is checked deterministically when the program is issued and
on replay, not by observing the system, and a mismatch is a Realization defect. A task the program
does not cover ends the Run Refused as unprogrammed. Driver faults are unchanged.

**One path for all workers.** The in-process Go worker follows programs too, so the live Answer
path and the Shim disappear. Callbacks remain in the Run as recorded events, so replay and
reduction are unchanged in kind.

## Recommendations on the open questions

1. Evidence source: the server's history only; worker actions are observed through it (owner, 2026-10-23).
2. Program scope: one program per proposal; batch runs compose them by activity id.
3. The in-process Go worker switches to programs (one path), retiring live Answer for workers.

## Consequences

- R40 is rewritten: a worker in any language follows a Model-issued program and the Run records
  what happened. The Shim (fn-228.5) is superseded; its partial start was deleted at the owner's request.
- The Temporal Driver's worker gate (fn-228.4) becomes "deliver the program, then observe"; the
  `worker_tasks` instruction may become unnecessary.
- Some Models may need answers that depend on observations a worker cannot see; those stay out of
  scope until a program key can express them, and the Realization refuses to program them.
