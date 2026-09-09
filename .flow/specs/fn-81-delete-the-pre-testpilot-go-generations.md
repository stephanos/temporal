## Goal & Context
<!-- scope: business -->

Status: implementation specification for a mechanical deletion sweep of the pre-Testpilot Go
generations. [UMPIRE4_SPEC](../../.plans/UMPIRE4_SPEC.md) remains normative; nothing here changes
modeled behavior.

The repository still carries every earlier generation of this project as live, compiled, and
partly CI-wired code. Nothing in the current Umpire, Testpilot, or Temporal model imports it. The
2026-09-08 assessment measured it:

| Tree | Go lines | Still wired into |
| --- | --- | --- |
| gomad, gomad1, gomad2, gomad3, gomad3sim, gomad3integration | 209,000 | gomad3 workflow, Makefile targets, three nested go.mod files |
| umpire1, umpire2, umpire3 | 92,700 | umpire3 and umpire-model-verification workflows, 90 Makefile lines, a second Lake project under umpire3 |
| agentworkflow, fairsim, planindex | 14,000 | Makefile |
| legacy tests under tests | 6,200 | the live-test gate that pins eleven expected failures by name |
| cmd/umpire-genmodels | — | imports umpire2 |

Live Umpire plus Testpilot is about 42,000 lines. Two costs are paid today. The gomad3 workflow
triggers on any change to go.mod or the Makefile, so it runs on unrelated pull requests. The
working disk is at 97 percent and umpire3's Lake build alone holds 677 MB.

The Lean previous generations (Nexus v1, Nexus2, Umpire Artifact, Space, Exploration) are out of
scope. They are imported by live modules or reserved by open specs (fn-22, fn-33, fn-79, fn-80)
and need a roadmap decision, not a sweep.

## Architecture & Data Models
<!-- scope: technical -->

### Deletion set

- Go trees: `tools/gomad`, `tools/gomad1`, `tools/gomad2`, `tools/gomad3`, `tools/gomad3sim`,
  `tools/gomad3integration`, `tools/umpire1`, `tools/umpire2`, `tools/umpire3`,
  `tools/agentworkflow`, `tools/fairsim`, `tools/planindex`, and `cmd/umpire-genmodels`, including
  their nested go.mod and go.sum files and any checked-in toolchains.
- Tests: every file under `tests` whose name begins with `umpire2_` or `umpire3_`, and the
  `tests/probe` and `tests/gomadfunctional` packages.
- CI: the `umpire3`, `gomad3`, and `umpire-model-verification` workflows.
- Makefile: every variable, target, and `.PHONY` entry whose only purpose is a deleted tree,
  including the `ALL_SRC` and `ALL_SCRIPTS` prune clauses for the gomad3 toolchain.
- The `umpire-check-live-tests` expected-failure list. The gate becomes a plain pass gate over
  the retained live tests.

### Retention set

- `.plans` history documents for GOMAD, UMPIRE2, and UMPIRE3 stay as historical record.
- `tools/umpire`, `common/testing/testpilot`, `tests/testcore/testpilot`, `tools/common`, and
  every other tool directory not named above stay untouched.
- The `tools/umpire/regression` CI workflow test keeps asserting the retained commands and must be
  updated to the new expected-failure state rather than deleted.

### Evidence rule

A removal decision needs repository-wide consumer evidence, following the ledger format
fn-66 established in `tools/umpire/CLEANUP_INVENTORY.md`. Absence of a Go import alone is not
evidence. Each deleted root records its consumers found by searching Go imports, Makefile,
workflows, shell scripts, proto files, Lean sources, generated manifests, and documentation, and
the disposition of each consumer.

## Edge Cases & Constraints
<!-- scope: technical -->

- A retained file that references a deleted path only in prose (README, plan doc, comment) is
  edited to say the path was removed, never left dangling.
- `go.mod` and `go.sum` at the root may lose dependencies only used by deleted trees; `go mod
  tidy` output is reviewed and recorded, and a dependency also used by live code stays.
- The root `go build ./...` and `go vet` scope grows to nothing new; the deletion must not expose a
  previously excluded package to the default build.
- The live-test gate must not silently pass because a test was deleted rather than fixed. The
  retained live tests are enumerated by name in the gate.
- Deletion happens in one commit per tree family so a revert is scoped.

## Quick commands

```bash
go build -tags test_dep ./...
CGO_ENABLED=0 go test -tags test_dep ./tools/... ./common/testing/testpilot/... ./tests/testcore/testpilot/...
make lint-code
make umpire-check-regression
git grep -n -E 'umpire[123]|gomad|agentworkflow|fairsim|planindex' -- . ':!.plans' ':!.flow'
```

## Acceptance Criteria
<!-- scope: both -->

- **R1:** A consumer-evidence ledger under `tools/umpire` classifies every deletion-set root with
  its repository-wide consumers and their disposition before any deletion lands. Errors: a root
  with an unclassified live consumer blocks deletion of that root.
- **R2:** The Go trees, nested modules, checked-in toolchains, and `cmd/umpire-genmodels` are
  deleted; the root module builds and vets; `go mod tidy` removes only dependencies with no
  retained importer. Errors: a build or vet failure blocks; a tidy diff that drops a dependency
  used by retained code blocks.
- **R3:** The legacy tests, the three workflows, and every Makefile variable, target, and prune
  clause serving a deleted tree are removed; no retained file references a deleted path except as
  a historical note. Errors: `git grep` over retained sources for the deleted names returns only
  `.plans` and `.flow` hits and explicitly annotated historical notes.
- **R4:** `umpire-check-live-tests` enumerates the retained live tests by name and requires them
  all to pass; the pinned expected-failure list is gone; the `tools/umpire/regression` workflow
  test asserts the new gate. Errors: a retained live test that fails blocks the gate.
- **R5:** `make lint-code`, `make umpire-check-regression`, and the Go test set in Quick commands
  pass; the task receipt records deleted line count and freed disk. Errors: a failed gate blocks
  completion.

## Boundaries
<!-- scope: business -->

- No Lean deletions. Nexus v1, Nexus2, Umpire Artifact, Space, and Exploration stay.
- No `.plans` deletions.
- No new CI workflow or dead-code detection gate.
- No refactoring of retained code; deleted-path references are edited, not restructured.
- No change to `tools/umpire`, `tools/common`, or Testpilot packages beyond reference edits.

## Decision Context
<!-- scope: both -->

Deleting rather than archiving to a branch was chosen because git history already preserves every
tree, and an archive branch invites the same CI wiring to return. Retaining the `.plans` history
documents keeps the reasoning that led to Umpire 4 readable without keeping code that competes
with it. Excluding the Lean generations keeps this spec mechanical; their removal depends on
fn-79 and fn-80 outcomes and belongs in a later consolidation spec. The fn-66 ledger format is
reused because it already survived a plan review and a completion review.
