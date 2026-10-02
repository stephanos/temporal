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

One command takes a candidate `go.mod` and `go.sum` (or a Go version) and
reports every pin it invalidates: adapters by module and version, packs by
rule and source-set digest, interception fingerprints, and clock-inventory
references. The report is path-free canonical JSON with a human rendering, and
exits nonzero when any pin is invalidated. It reads the same descriptors the
build reads, so it cannot disagree with the build's fail-closed checks.

### Adapter regeneration

One governed command re-derives an adapter's rewrite and digest anchors for a
new exact module version. It applies each rewrite by its existing structural
anchor, fails when an anchor no longer matches, and prints the changed
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
  `go.mod` or Go version, and a fixture bump of one adapted and one packed
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

- Patch and overlay reduction, feature removal, version-range pins, automatic
  approval, and release bundles are excluded.
- Automating the runtime patch rebase across Go releases is excluded; the
  existing `patch-regenerate` command and upgrade dossier keep that scope.

## Decision Context

The assessment ranked adapter anchors as the second-largest recurring cost
after the Go rebase, and the only one with no repair command. A report that
runs before a bump lets the owner batch repairs, where today the first signal
is a failed build.
