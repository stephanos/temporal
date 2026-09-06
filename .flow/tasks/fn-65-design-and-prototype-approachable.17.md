---
satisfies: [R3, R5, R7, R8]
---
# fn-65-design-and-prototype-approachable.17 Prototype the source-aware checked Property frontend

## Description
Implements R3, R5, R7, R8; use the parent spec and Nexus2 DESIGN.md for the approved semantics and prototype exceptions.

**Size:** M
**Files:** `model/Umpire/Property/Authoring.lean`, `model/Umpire/Property/ImportTests.lean`, `model/Temporal/Feature/Nexus2/AuthoringTests.lean`, `model/Temporal/Feature/Nexus2/README.md`
**Touches:** [model/Umpire/Property/Authoring.lean, model/Umpire/Property/ImportTests.lean, model/Temporal/Feature/Nexus2/AuthoringTests.lean, model/Temporal/Feature/Nexus2/README.md]

### Approach
Compare focused property% forms in ordinary def declarations over the same checked constructors; semantic tests and conflicts are already established dependencies. Reuse source occurrence capture and diagnostic elaboration (Target/Language.lean:924,952) while producing kernel-checked checker-success evidence. Keep syntax outside FiniteMachine; use the narrow recorded AUT-07 prototype exception.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Target/Language.lean` — source capture, elaboration and native default
- `model/Umpire/Property/Check.lean` — typed failures/checked extraction
- `model/Umpire/Behavior/Language.lean` — checker result
- `model/Umpire/Query/Language.lean` — Query checker/limits
- `model/Temporal/Feature/Nexus2/DESIGN.md` — comparison decisions and evaluation tasks
- `.plans/LEAN_GUIDELINES.md` — trust/editor/checked example requirements

### Quick commands
```bash
(cd model && mise exec -- lake build Temporal.Feature.Nexus2.Tests Umpire.Property.Tests Umpire.Behavior.Tests Umpire.Query.Tests)
make lint-model
make lint-code GOLANGCI_LINT_FIX=false
```

Baseline only existing roots before creation; after implementation include the new roots named below. Run focused commands during iteration, and the parent final gates at prototype completion. Use the Makefile LEAN_LAKE platform wrapper if direct Lake invocation cannot find the macOS SDK. Preserve comments and existing unrelated changes. No commits unless the user requests them.

Post-edit focused gate (mandatory after creating the root): `(cd model && mise exec -- lake build Temporal.Feature.Nexus2.AuthoringTests)`.

Scope this slice to property% and its Target/source capture reuse. Behavior and Query syntax stay constructor-only until the next task; do not add their frontend files here. Required negative semantics/coverage/conflicts are already dependencies.

Export reusable APIs through their existing owning facades and add the corresponding focused import checks when the public surface changes; keep new generic modules in the named owner. New tests must be imported into the named gate root immediately.

## Acceptance
- [ ] Compile and admit property% specimens for baseline/race, same-step cases and guarded bounded responses over the same constructors and fixed IDs; compare checked meaning/fingerprints to constructor fixtures. Syntax supplies no separate evaluator or implicit precedence.
- [ ] Source-diagnostic compilation tests freeze typed error kind, related IDs and exact parent/case/exception/clause spans for malformed/duplicate IDs, missing/wrong-kind references, missing capabilities, unsupported operators/syntax, future/result guards, empty groups and invalid/missing units. No partial checked constant escapes.
- [ ] Synthesize kernel-checked checker-success evidence using ordinary reduction or reusable proved reflection, never the existing native checkedTarget default. Audit exported checked Property/Target dependency inventories and a failing admission/proof path. If impractical, report measured costs and preserve successful-branch constructors rather than silently change trust.
- [ ] Measure Property constructor/frontend compilation and admission for baseline/10x declarations and record available completion/hover/navigation/error-recovery observations, with tool/environment and explicitly unmeasured items. No human usability or final grammar approval claim.
- [ ] Add AuthoringTests as a runnable root immediately and keep a single default example plus comparison specimens. Behavior/Query frontend and cross-language final selection belong to the next task.

## Done summary
Implemented the narrow source-aware `property%` frontend over the existing Property checker, with baseline and guarded race equivalence, exact nested source diagnostics, public facade checks, trust-boundary evidence, and constructor/frontend measurements. The frontend returns the checker's `Except` result because `rfl`, `decide`, `decide +kernel`, targeted unfolding, a minimal closed check, and a 6.67-second owner-local kernel probe did not synthesize checker-success evidence; `PropertySpec.checked` preserves the explicit kernel-proof constructor route without weakening trust.

Source diagnostics now freeze complete typed errors, related IDs, roles, and exact parent/case/exception/clause spans for all required negative cases, including a multi-related-ID nested case and the exact missing-unit compiler error. Diagnostic JSON uses `Lean.Json.compress`, with newline, tab, and carriage-return parsing coverage. Single warm Lean 4.33.1 profiler observations were constructor 1x 0.000593s, constructor 10x 0.002720s, frontend 1x 0.013429s, and frontend 10x 0.119365s; compiler recovery was observed, while completion, hover, navigation, interactive latency, cold-cache variance, and human/final-grammar usability were not measured.

No commit was created; user retains commit ownership and HEAD remains `7774fdc7ac751ac959816c9829516ce54af57194`. Review receipt: `/tmp/impl-review-receipt-fn-65-design-and-prototype-approachable.17.json`; reviewed owned tree: `7c1a6d158e3d95f5d886726f198e4604087382ac`.

stage: impl-review - ran [2026-09-06T00:19:48Z..2026-09-06T00:25:05Z] | codex:gpt-5.6-sol:medium | session 01a07415-c1eb-79b2-b6b4-8d77721bafcd | NEEDS_WORK then SHIP
stage: plan-sync - skipped(config: planSync.enabled=false)
stage: tracker-sync - skipped(config: bridge inactive)
## Evidence
- Commits:
- Tests: baseline: green via task16 handoff (Quick roots 75, model lint 249, Go inherited set 1316), (cd model && mise exec -- lake build Temporal.Feature.Nexus2.AuthoringTests Umpire.Property.ImportTests) (40 jobs, green; /tmp/fn65-task17-review-fix-focused.log), (cd model && mise exec -- lake build Temporal.Feature.Nexus2.Tests Umpire.Property.Tests Umpire.Behavior.Tests Umpire.Query.Tests) (75 jobs, green; /tmp/fn65-task17-review-fix-final-quick.log), make lint-model (249 jobs, green; /tmp/fn65-task17-review-fix-final-lint-model.log), make lint-code GOLANGCI_LINT_FIX=false (inherited exit 2: 1316 sorted diagnostic headers exactly match task16, SHA-256 aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077, diff 0; /tmp/fn65-task17-final-lint-code.log), Lean profiler single warm run: constructor 1x 0.000593s, constructor 10x 0.002720s, frontend 1x 0.013429s, frontend 10x 0.119365s (/tmp/fn65-task17-profile3.log), kernel proof probes: rfl, decide, decide +kernel, targeted unfolding, minimal closed context, and owner-local decide +kernel failed without resource-limit diagnostics; owner-local build 6.674667s (/tmp/fn65-task17-owner-proof-probe.log), impl-review codex:gpt-5.6-sol:medium session 01a07415-c1eb-79b2-b6b4-8d77721bafcd round 2 SHIP; reviewed owned tree 7c1a6d158e3d95f5d886726f198e4604087382ac
- PRs: