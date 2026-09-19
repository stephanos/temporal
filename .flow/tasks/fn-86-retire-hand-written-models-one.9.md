---
satisfies: [R8]
---
# fn-86-retire-hand-written-models-one.9 Rule drafts, one-authoring-path documents and the full gate

## Description
Close the spec (R8): draft under GOV-02 the AUT-08 amendment removing the expert alternative, the AUT-07a amendment naming the commands as the only authoring path for feature Models, and the MOD rule for task .8 enforced under MOD-11; make the architecture documents, `model/README.md`, `model/AUTHORING.md` and the order document describe one authoring path; run the full gate. Single finalization task.

**Size:** M
**Files:** `.plans/UMPIRE4_SPEC.md` (AUT-08 lines about the expert alternative; AUT-07a; a new MOD-16 with the `authoring-path-isolation` label; MOD-11's list; each marked `drafted by fn-86; awaiting GOV-02 approval`), `model/README.md` (:36 the nonexistent `Success.Producer`; :94-101 the Lifecycle walkthrough; :141-199 typed authoring; :204-211 typed-Nexus Known Gaps; :221-231 the coverage record and Race links), `model/ARCHITECTURE.md` (:46-49, :103-114, :147-150, :254-256), `model/Umpire/ARCHITECTURE.md` (:49-50, :103-105, :246-274), `model/AUTHORING.md` (a short "one path" paragraph; a delta, not a rewrite), `tools/umpire/CONTEXT.md` (:41-44 "Derived rule" names the field relation), `tests/testcore/testpilot/README.md` (:19-43), `.plans/UMPIRE4_ORDER.md` (fn-86 entry; the fn-79 and fn-33 rows reflect the recorded behavior; gate baselines), `model/HANDWRITTEN_INVENTORY.md` (final state: every row resolved)
**Touches:** [.plans/UMPIRE4_SPEC.md, .plans/UMPIRE4_ORDER.md, model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md, model/AUTHORING.md, tools/umpire/CONTEXT.md, tests/testcore/testpilot/README.md, model/HANDWRITTEN_INVENTORY.md]

### Approach
- Use the docs-gap list in the planning record as the checklist; every `path:line` is either rewritten or recorded as still true.
- MOD-15: every backticked dotted name cited by the new rule text must resolve; `go test ./tools/umpire/vocabulary/...`.
- Gate order: `go clean -cache`, `make umpire-check-regression`, `make lint-model` (LEAN_NUM_THREADS=1), `make lint-code GOLANGCI_LINT_FIX=false`; record the numbers in the order document's baseline table; treat a `lint-code` count below 161 as a truncated run.

### Investigation targets
**Required:**
- `.plans/UMPIRE4_SPEC.md:172-173,284-301` — MOD-11, AUT-07a, AUT-08 (the sentence at :298-299)
- `.plans/UMPIRE4_ORDER.md:166-197,207-216,238,249-255`
- the docs-gap table for fn-86 in the planning record

**Optional:**
- `tools/umpire/vocabulary/spec_names_test.go:26-39` — the MOD-15 gate

### Key context
- Boundaries: no edits to historical `.plans` documents other than the order document and the drafted rules.

## Acceptance
- [ ] AUT-08 no longer offers direct `Umpire.Machine` construction to feature authors; AUT-07a names the commands as the only authoring path; the new MOD rule exists and MOD-11 lists it; all three are drafts under GOV-02; MOD-15 gate green
- [ ] no document describes a hand-written authoring path for feature Models; the docs-gap list is fully worked; the inventory's every row is resolved
- [ ] `make umpire-check-regression` exit 0; `make lint-model` at or below its baseline; `make lint-code` at 161 after `go clean -cache`; baselines recorded in the order document


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
