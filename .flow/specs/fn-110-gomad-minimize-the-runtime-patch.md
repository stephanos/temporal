# Gomad: minimize the runtime patch without losing functionality

**Plan date:** 2026-09-30

## Goal & Context

Reduce the pinned Go 1.27.1 patch through both approved approaches. Move
Gomad-only implementation into the existing source overlay, then emit the
canonical patch with one context line. Preserve every supported behavior,
existing comment, boundary check, and qualification assertion.

The [patch-size investigation](../../docs/research/gomad/GOMAD_PATCH_SIZE.md)
measured a 32,275-byte, 998-line baseline. Its combined extraction candidate
measured 27,845 bytes and 849 lines with three context lines, or 20,352 bytes
and 581 lines with one context line. The combined byte reduction was 36.9%.
These are reference measurements, not a quota or completed qualification.
The context-only candidate reproduced identical source on macOS; extraction
prototypes have limited build and fixture evidence.

This spec owns patch representation and relocation of existing runtime/stdlib
implementation. The production-code cleanup in
[fn-108](fn-108-gomad-reduce-code-size-without-removing.md) has a separate
metric and excludes runtime/overlay redesign. Moving code into an overlay
reduces the upstream patch without claiming a net deletion of production code.
The [milestone constraints](../../MILESTONES.md#constraints)
continue to apply.

## Architecture & Data Models

### Overlay extraction

Keep integration hooks in upstream Go and give the existing overlay ownership
of the Gomad implementation behind them. Scope includes three scheduler
extractions from the investigation:

- Move `gomadSimulationTimeQuiescenceChanged` verbatim.
- Extract the quiescence and time-advance selection in `checkdead` behind a
  private interface returning whether to wake a timer or wait. Retain the
  upstream M/P timer-wake machinery and normal scheduler control flow.
- Extract the enabled idle-P syscall resumption in `exitsyscallNoP`. Retain
  arrival admission, locked-goroutine handling, P acquisition, and the calls
  that never return.

Move the crypto random-reader initialization, including its FIPS/internal
reader override, verbatim into an additive overlay in the same package.
Restore the pristine upstream crypto random source. Move the three syscall
linkname declarations into an additive Unix overlay with matching build
constraints. Retain the lazy environment reset and both syscall write hooks
at their present lifecycle points.

The helper interfaces preserve lock ownership, allocation constraints, return
semantics, and the order of random draws and observable decisions. Move
existing comments with their implementation. Reuse the existing overlay
rather than copying whole upstream scheduler or standard-library files.

### Canonical patch representation

Generate the final patch with one context line (`-U1`) through the governed
regeneration command. Retain canonical Git file headers, plain-text input,
path validation, dry-run application, and zero fuzz. Do not introduce a
context-size CLI option or compressed patch format.

For the same final source candidate, three-context-line and one-context-line
patches must materialize identical files. This comparison isolates the
representation change; it does not substitute for behavioral qualification
of the extracted source.

The release descriptor remains the owner of exact patch and overlay source
sets. Regenerate its derived consumers when paths change. Make the pinned
regeneration check follow the descriptor instead of the obsolete Go version
hardcoded in its current test.

## API Contracts

Keep public Go interfaces, CLI commands/flags/defaults, capability boundaries,
compatibility packs and adapter versions, schemas, transcript formats,
resource bounds, error classifications, and disabled/direct-seed/Runner
activation behavior unchanged. Extracted helpers are private implementation.

Raw patch and overlay bytes remain inputs to the toolchain build key. A new
patch representation or overlay produces a new build identity, and retained
artifacts keep their original identity. Existing artifacts remain replayable
with their original toolchain. Qualification of the final candidate records
and replays fresh artifacts within that candidate's identity. No schema
migration, identity override, or claim of identical binary text offsets across
builds is part of this change.

## Edge Cases & Constraints

- Preserve the complete quiescence response handling, arrival rechecks,
  transport syscall accounting, and `sched.lock` release/reacquisition.
  Normal helper returns retain the caller's expected lock state. Transport
  failure and deadlock retain their fatal behavior.
- Preserve saturated timer deadlines and the distinction between a timer at
  `maxWhen` and no timer. Preserve due-timer ordering before host-syscall
  arrivals, one arrival per idle window, and locked goroutine resumption.
- Retain activation before random initialization, one-P execution, host
  preemption suppression, host/target random-stream separation, run-queue
  choices, select caller-site capture, goroutine identities, and observations.
- Retain GC stabilization hooks, timer tie-breaking, synctest behavior,
  clock activation guards, traceback normalization, panic/exit/testing
  completion hooks, goroutine fields, and upstream size assertions.
- Crypto relocation retains initialization timing and both random-reader
  assignments. Syscall relocation retains exact symbols/signatures, Unix
  source selection, lazy environment copying, write guards, and instrumentation.
- Existing Linux and Darwin replay findings remain owned by fn-105 D12/D14.
  Record baseline dispositions and preserve any stricter expectations present
  at implementation time. New unexplained regressions fail this work; a
  missing host or gate is incomplete validation.
- Do not add allocations, new host reads, generic capability grants,
  dependencies, or resource-bound changes to obtain a smaller patch.

## Acceptance Criteria

- **R1:** Record the implementation baseline, pinned archive/patch digests,
  source-set inventory, and comparable byte/line counts before editing.
  Report patch bytes/lines, edited upstream files, added/deleted source lines,
  and overlay bytes/lines separately. Errors: unavailable or drifting baseline
  inputs prevent an equivalence or size claim.

- **R2:** The three scheduler implementations above live in the existing
  runtime overlay behind private interfaces, with upstream integration hooks
  and comments preserved. Regression evidence exercises timer advancement,
  arrival ordering, transport/quiescence races, locked goroutine resumption,
  and process-simulation clock/replay behavior. Errors: retry/external/deadlock
  responses and transport failures preserve current behavior and lock state;
  a changed draw order, allocation path, or host-timing decision is a regression.

- **R3:** Crypto initialization and the three syscall declarations use
  additive package overlays; the upstream crypto random source is pristine.
  Enabled entropy/transcript tests cover public random reads and key
  generation through the internal/FIPS override. Seeded environment, output,
  and disabled-mode tests preserve current results. Errors: unmodeled writes
  remain denied, and neither host entropy nor host environment is exposed by
  relocation; package lifecycle and build constraints remain intact.

- **R4:** The canonical regenerator emits `-U1`, and repeated regeneration
  from the same pinned final candidate yields identical patch bytes. On both
  supported hosts, final `-U3` and `-U1` candidates apply with zero fuzz and
  produce byte-identical files. The pinned regeneration check reads the
  descriptor and executes when its verified archive is available. Errors:
  malformed patches, unexpected paths, archive/version mismatches, fuzz-only
  application, and stale generated output remain rejected; a skip is not proof.

- **R5:** The descriptor's exact patch/overlay sets match the final inputs,
  derived consumers are current, and governed validation and archive-based
  overlay collision checks pass. Errors: undeclared additions, collisions,
  prohibited runtime edits, binary/generated input violations, and changed
  archive checksums remain failures without relaxing the allowlists' policy.

- **R6:** Every retained runtime, standard-library, activation, boundary,
  public-interface, and evidence contract above remains available. Existing
  comments and behavioral assertions remain, including the timer-presence
  result and goroutine size check. Errors: functionality removal, weakened
  negative tests, boundary widening, changed failure classification, or altered
  disabled behavior fails acceptance even if the patch becomes smaller.

- **R7:** The integrated toolchain passes the full Gomad gate on
  `darwin/arm64` and `linux/amd64`, plus affected entropy, process-simulation,
  Temporal integration, smoke, and qualification suites. Use the existing
  native entrypoints and `test_dep` conventions; require fresh same-seed runs
  and exact replay where the baseline contract requires them, including seeds
  11 and 17 for affected Temporal workloads. Preserve workload dispositions
  and record before/after outcomes. Errors: missing host execution, watchdogs,
  capacity failures, new replay divergence, or weakened expectations leave
  acceptance incomplete. Cross-compilation and small fixture checks alone
  do not satisfy this requirement.

- **R8:** Final measurements demonstrate both reductions independently.
  The final extracted source's `-U3` patch is smaller than the baseline `-U3`
  patch, and its canonical `-U1` patch is smaller than that final `-U3` patch.
  Publish the counts and qualification evidence, and update milestone status
  and maintainer regeneration guidance. Explain the new build identity and
  preserve original artifact bindings. Errors: hiding moved overlay code,
  deleting comments/features to hit 20,352 bytes, relabeling old artifacts,
  or reporting incomplete qualification as passing is unacceptable.

## Boundaries

- This scope relocates existing behavior and changes diff context. Runtime
  scheduling, GC, time, random, entropy, and replay semantics are not redesigned.
- Embedded goroutine state, compiler/linker hook consolidation, early syscall
  environment initialization, and compiler interception of initialization or
  syscall writes remain further candidates outside this spec.
- Full-file overlay replacement, patch compression, zero-context patches,
  new dependencies/platforms/capabilities, and source/test translation are
  excluded. Existing collision and prohibited-runtime policies stay intact.
- Production-code consolidation and downstream cell work retain their
  fn-108 and fn-107 owners. D12/D14 defect fixes retain their separate owners;
  this spec cannot close them through a refactor or disposition change.

## Decision Context

One context line gave the largest source-equivalent reduction in the
investigation while retaining an anchor at each hunk. Keeping Gomad-only
implementation in the overlay reduces the source carried in the upstream
patch and makes the quiescence implementation easier to inspect at one seam.
Three context lines remain useful as an intermediate measurement and source
comparison during qualification.

The 20,352-byte prototype is guidance because helper shape and necessary
regression coverage may change during implementation. The acceptance tests
require both reductions and preserved behavior rather than a byte quota.
Moving additional lifecycle work or changing goroutine layout would add
qualification questions beyond the measured core candidate.
