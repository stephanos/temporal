# Gomad: reduce version-pin maintenance

**Plan date:** 2026-10-01

## Goal & Context

Cut the manual work a dependency or Go version bump causes, without loosening
any exact pin. The [2026-10-01 quality assessment](../../MILESTONES.md#maintenance-cost)
counted the pins: a 1010-line runtime patch with a 57-file overlay, 131
interception fingerprints, 15 dependency adapters anchored by 129 SHA-256
literals, and 12 compatibility packs covering 19 module versions. Upstream
`go.mod` changed in 73 commits over six months, including 4 `go` directive
bumps. Each pin fails closed on a bump, which is intended. The cost is that
adapter repair has no command, pack repair takes four commands, and nothing
reports ahead of time which pins a given bump breaks.

Counts above come from the assessment's source reading and are re-measured as
the baseline before work starts.

### Relationship to existing work

- fn-110 owns reducing the runtime patch. This spec does not change the patch
  or overlay content.
- fn-107 owns the downstream cell, including the six adapters for modules the
  server does not import.
- fn-112 owns determinism assurance and the test suite shape.
- [COMPAT-8](../../.plans/GOMAD_NEXT.md#compat-8-dependency-and-go-upgrade-impact-reports)
  is the roadmap item this spec draws from. Rollback bundles and release
  attestations stay on the roadmap.

## Architecture & Data Models

### Pin impact report

One command takes a candidate `go.mod` and `go.sum` and
reports every pin it invalidates: adapters by module and version, packs by
rule and source-set digest, interception fingerprints, and clock-inventory
references. The report is path-free canonical JSON with a human rendering, and
exits nonzero when any pin is invalidated. It reads the same descriptors the
build reads, so it cannot disagree with the build's fail-closed checks.

### Adapter regeneration

One governed command re-derives an adapter's rewrite and digest anchors for a
new exact module version. It applies each rewrite by its existing structural
exact-occurrence anchor, fails when an anchor no longer matches exactly once, and prints the changed
upstream source for review. It writes the new anchors only with an explicit
approval digest, the same control compatibility packs use. A changed upstream
file never produces a silently shifted rewrite.

### Pack refresh

One command runs `discover`, `review`, and `generate` for every request a bump
invalidates and stops at the review approval. Stale variants that no qualified
module version selects are removed, with the evidence that nothing selects
them.

## API Contracts

Pins, pack and adapter identities, fail-closed behavior, approval digests,
target identity, and replay compatibility are unchanged. The new commands are
`gomadtool` subcommands; the `gomad` CLI grammar is unchanged. Regenerated
adapters and packs carry new identities, and retained artifacts keep theirs.

## Edge Cases & Constraints

- The [milestone constraints](../../MILESTONES.md#constraints) apply. No pin is
  widened to a version range, and no generic capability is granted.
- Regeneration never approves on its own. Review of changed upstream source by
  a person stays in the flow.
- The impact report must name a pin it cannot evaluate as unknown, never as
  unaffected.
- Module downloads use the exported proxy settings and fail as infrastructure
  errors when unavailable.
- Both platforms' packs and adapters are covered; a host that cannot evaluate
  the other platform's pins says so in the report.

## Acceptance Criteria

- **R1:** A re-measured baseline records every pin class, its count, and the
  commands and manual steps a bump of each currently needs. Errors: a count
  that differs from the assessment is corrected in the milestones.

- **R2:** The pin impact report lists every invalidated pin for a candidate
  `go.mod`, and a fixture bump of one adapted and one packed
  module shows the expected entries. Errors: a pin the build later rejects
  that the report called unaffected fails this criterion.

- **R3:** Adapter regeneration re-derives anchors for a new exact version
  behind an approval digest, and a negative fixture with a moved anchor fails
  without writing. One real adapter is regenerated across a version bump with
  the command and qualifies.

- **R4:** Pack refresh runs the authoring steps for every invalidated request
  up to approval in one command. Unselected stale pack variants are removed
  with retained evidence.

- **R5:** The README and upgrade guide describe the bump procedure with the new
  commands, and the measured manual steps per bump are reported against the
  R1 baseline.

- **R6:** `make -C tools/gomad3 validate` and `test`, compatibility-pack
  qualification, and the core set pass on both platforms. Errors: missing
  linux/amd64 evidence leaves acceptance incomplete.

## Boundaries

- A Go-version candidate as report input is excluded; the upgrade dossier
  covers Go bumps.
- Patch and overlay reduction, feature removal, version-range pins, automatic
  approval, and release bundles are excluded.
- Automating the runtime patch rebase across Go releases is excluded; the
  existing `patch-regenerate` command and upgrade dossier keep that scope.

## Decision Context

The assessment ranked adapter anchors as the second-largest recurring cost
after the Go rebase, and the only one with no repair command. A report that
runs before a bump lets the owner batch repairs, where today the first signal
is a failed build.

## Planning decisions (2026-10-01)

Task breakdown settled the points below. Each is a default the owner can
change before the task that uses it starts.

- **R2 source of truth.** Adapter anchors stay Go constants. The report reads
  them by importing the adapter registry, so no data migration and no adapter
  identity change.
- **R2 input** is one candidate `go.mod` with its `go.sum`, by default the
  repository root module. Exit status follows the existing convention: 0 for
  no invalidated pin, 1 for at least one, 2 for invalid input, 3 for
  infrastructure failure. An unknown pin counts as invalidated.
- **R2 side effects.** Module resolution runs outside the target module with a
  private module cache, because `go mod download` inside a target module
  rewrites its `go.sum`.
- **R3 anchors** are the existing exact-occurrence byte anchors. An anchor
  that matches zero or more than one time fails. The approval digest covers
  the changed upstream source and the proposed new anchors, comes from a dry
  run, and is passed back on the command line. The transaction covers the
  generated outputs: the full set is staged and verified in a scratch copy,
  then published under an exclusive lock after revalidating the checkout, with
  a marker that lets the next run complete or roll back an interrupted
  publication.
- **R3 real adapter** is the first adapted module the root `go.mod` has moved
  past when the task starts. If none has moved, the fixture bump stands in and
  the criterion says so.
- **R4 inputs.** Refresh runs on a checkout where the bump is applied. Each
  request is discovered in its own target module through one checked-in
  request-to-directory table shared with the Makefile qualification list.
- **R4 approval** is per request. A request is done when its stored approval
  matches the review digest of freshly discovered evidence, so an approval of
  older evidence never counts and partial progress survives a rerun. Variants are removed only on a host that can evaluate their
  platform; the others are reported and left.
- **R5 manual step** means one command invocation or one hand edit of a
  checked-in file.

## Open Questions

1. fn-109 tasks 10 and 11 change the adapter registry and source-inventory
   owner, and fn-105 task 8 adds adapters. Should R3 wait for them, or land
   first and be rebased?

## Quick commands

```bash
make -C tools/gomad3 validate
go -C tools/gomad3 test -tags test_dep ./cmd/gomadtool ./deterministicio ./internal/compatibilitypack/...
```

## Early proof point

Task fn-113-gomad-reduce-version-pin-maintenance.1 validates the core approach:
the impact report names exactly the pins the build rejects for a fixture bump.
If the report and the build disagree, re-evaluate reading anchors through the
registry before tasks 2 and 3.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1 | Re-measured pin baseline | .1 | — |
| R2 | Pin impact report | .1 | — |
| R3 | Adapter regeneration behind approval | .2 | — |
| R4 | Pack refresh and stale variants | .3 | — |
| R5 | Bump procedure documented and measured | .4 | — |
| R6 | Gates pass on both platforms | .4 | — |
