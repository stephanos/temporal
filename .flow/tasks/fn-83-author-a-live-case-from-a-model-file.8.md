---
satisfies: [R5, R8]
---
# fn-83-author-a-live-case-from-a-model-file.8 Authoring walkthrough, drift test, documents, roadmap, and full gate

## Description
Write `model/AUTHORING.md` from the R6 Model file's marked regions with a Go drift test (R5), sweep every document the docs-gap scan named, add the `case` concept entry and one drafted AUT-09 amendment to the spec under GOV-02, reconcile the roadmap, and run the full gate (R8). Single finalization task.

**Size:** M
**Files:** `model/AUTHORING.md` (new), `model/Temporal/Feature/Nexus/Success/Nexus.md` (deleted), `model/Temporal/Feature/Nexus/Success/DESIGN.md` (new, the proposed forms moved verbatim), `model/Temporal/Feature/Nexus/Success/Integration.md` (reduced), `tools/umpire/authoring/drift_test.go` (new), `model/README.md`, `model/ARCHITECTURE.md`, `model/Umpire/ARCHITECTURE.md`, `tools/umpire/CONTEXT.md`, `tools/umpire/CLEANUP_INVENTORY.md`, `common/testing/testpilot/README.md`, `.plans/UMPIRE4_SPEC.md`, `.plans/UMPIRE4_ORDER.md`
**Touches:** [model/AUTHORING.md, model/Temporal/Feature/Nexus/Success/*.md, tools/umpire/authoring/**, model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md, tools/umpire/CONTEXT.md, tools/umpire/CLEANUP_INVENTORY.md, common/testing/testpilot/README.md, .plans/UMPIRE4_SPEC.md, .plans/UMPIRE4_ORDER.md]

### Approach
- Tutorial: empty file to green test in order: `enum` (or `inductive`) declarations, `model`, `property`, `scenario`, `limits`, `query`, `case` (framework keys spelled `word:`), `lake build`, `make umpire-gen-case-runtime-conformance`, the Go test, `make umpire-check-regression`; then `umpire-run` against a dev server; then a section listing every located diagnostic from .3 and .5 with its fix; then the fault lines with the outage Model as the example.
- Drift test: parse `AUTHORING.md` fenced `lean` blocks tagged with a marker name, read the R6 Model file's `-- authoring: <name>` regions, `require.Equal` each pair; a missing or duplicate marker fails naming it. Generate-and-diff is the nearest prior art in the regression-views generator; this is extract-and-compare.
- Documents: apply the docs-gap scan findings: README Case production and regression paragraphs, both ARCHITECTURE files (new `Umpire.Case.Producer` row, Case production ownership, runtime handoff), CONTEXT.md glossary entries for Realization, Template, Hook, Evidence mapping and the Fault entry's authored spelling, CLEANUP_INVENTORY's renderer row marked as later accounting, the Testpilot README fault paragraph and the provisioning sentence.
- Spec: add a `case` block concept to Core concepts and draft one amendment under a new ID for AUT-09 (evidence resolution against generated history names), marked pending human approval; do not approve it. The fn-82 spec-name resolution test must pass.
- Roadmap: move fn-83 into the completed list with the measured numbers (files per Case, lines of the translated Model, fixtures runnable from the CLI); record the three pending rules and the amendment as the items needing a human.

### Investigation targets
**Required:**
- `model/Temporal/Feature/Nexus3/Nexus.md:31-63, 194-205, 250-270` — what to keep, what moves to DESIGN.md
- `model/README.md:28-64, 295-348` and `model/ARCHITECTURE.md:110-148, 238-246` — stale Producer and renderer statements
- `tools/umpire/cmd/umpire-gen-regression-views/render_test.go:28-92` — markdown generate-and-diff prior art
- `.plans/UMPIRE4_SPEC.md:28-69, 248, 418-437` — concepts and the three drafted rules
- `.plans/UMPIRE4_ORDER.md` — the fn-82 entry shape to follow for fn-83's completed entry

### Acceptance
- [ ] `model/AUTHORING.md` exists, the old tutorial is deleted, DESIGN.md holds the proposed forms verbatim
- [ ] `go test -tags test_dep ./tools/umpire/...` passes including the drift test; a deliberately edited block fails it naming the marker
- [ ] Every document the docs-gap scan named is updated; no document names the deleted Producer or outage files
- [ ] `UMPIRE4_SPEC.md` has the `case` concept and the drafted amendment marked pending; the spec-name resolution test passes
- [ ] `make umpire-check-regression`, `make lint-model`, and `make lint-code` (touched files) pass; `UMPIRE4_ORDER.md` records fn-83 as completed with the measurements

## Acceptance
- [ ] TBD

## Done summary
Blocked:
Blocked 2026-09-10; superseded by fn-85 ("Model side effects as typed interfaces and run query sets").

The per-Case `case` block (one Query, one hand-picked whole-Program template, per-Case evidence lines) is replaced by:

- **Side effects in the Model.** Entities with structured state, and interfaces with a kind (`call`, `command`, `reply`, `observation`), a party, input classes with representatives, and result classes. Request fields that decide the outcome are Model behavior, not binding detail.
- **A Temporal Realization** that binds interfaces, result classes, observations, setup parameters and parties to RPCs, workflow commands, handler replies, history events and dynamic config. The Producer assembles Program and Contract from the witness; whole-Program templates and the `case` command are removed.
- **Query sets per purpose.** A set binds each party to test or environment; a functional set compiles to one Case per Query; canary and exploratory sets are admitted for fn-70/fn-29 and fn-33.

fn-85's final task closes this task as superseded and names where its concern went. Design record: `model/Temporal/Feature/Nexus/DESIGN.md`.
## Evidence
- Commits:
- Tests:
- PRs:
