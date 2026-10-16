# Gomad: sync with upstream Temporal main

## Conversation Evidence

> user: "let's add a task to rebase on latest main from github.com/temporalio/temporal"

## Goal & Context

Bring the `gomad` branch up to date with `github.com/temporalio/temporal` `main` so Gomad runs against current Temporal server code. [paraphrase]

On 2026-10-16 `gomad` is 102 commits behind upstream `main` (`dcd360f177`, 2026-10-15) and 6,002 commits ahead of their merge base `951c5516e9` (2026-09-10); upstream changed 832 files since that base.

## Architecture & Data Models

Integrate upstream with a merge commit of `upstream/main` into `gomad`, not a literal history rewrite. Conductor decision (2026-10-16): replaying 6,002 commits is impractical, and rewriting them would orphan the commit SHAs that retained Flow evidence, qualification reports and artifact provenance bind. The merge keeps every existing SHA valid and leaves a single reviewable conflict-resolution commit. Conflicts are expected only where Gomad touched server code (`common`, `service`, `temporal`, `tests/testcore`) and in `go.mod`/`go.sum`.

## Acceptance Criteria

- **R1:** `gomad` contains upstream `main` at a recorded upstream SHA; conflict resolutions keep Gomad's bounded server seams (build tags, injection seams) with the default production build unchanged. Errors: a resolution that drops a Gomad seam or changes default production behavior is a finding.
- **R2:** Dependency changes from upstream are reconciled with Gomad compatibility packs and version pins: `make -C tools/gomad validate`, pin-impact and pack checks pass, and any pack invalidated by a dependency bump is re-reviewed through the existing discover/review/generate/check flow rather than widened.
- **R3:** The default host gates (build, vet, lint, `test-wasm`, archcheck) and the WASM qualification ladder (runtime qualification, Temporal onebox/workflow/shard, multi-instance cluster, SQLite and the retained capacity/resource budgets) pass on the merged candidate; rotated guest/module/model identities are recorded. Errors: a Temporal test that now needs a Gomad-specific overlay is a blocker, not a fix.
- **R4:** One handover records the upstream SHA, conflict list and resolutions, commands, exits and identity rotations; MILESTONES and README reference the new upstream base.

## Boundaries

- No push, PR or CI action; the merge stays local until the owner authorizes publishing.
- Native backend execution stays deferred under fn-128/fn-149.
- No source translation or test rewriting to absorb upstream changes.

## Decision Context

Ordered after fn-182 (which edits guest provider selection in server code) and fn-184, and before fn-187 and fn-185, so the final host-side cost profile measures the synced candidate.
