---
satisfies: [R3, R5, R7, R8]
---
# fn-65-design-and-prototype-approachable.18 Complete Behavior and Query frontends and measure the authoring comparison

## Description
Implements R3, R5, R7, R8; use the parent spec and Nexus2 DESIGN.md for the approved semantics and prototype exceptions.

**Size:** M
**Files:** `model/Umpire/Behavior/Authoring.lean` (new), `model/Umpire/Query/Authoring.lean` (new), `model/Umpire/Target/Language.lean` (source reuse only), `model/Temporal/Feature/Nexus2/AuthoringTests.lean`, `model/Temporal/Feature/Nexus2/README.md`
**Touches:** [model/Umpire/Behavior/Authoring.lean, model/Umpire/Behavior/ImportTests.lean, model/Umpire/Query/Authoring.lean, model/Umpire/Query/Tests/Visibility.lean, model/Umpire/Target/Language.lean, model/Temporal/Feature/Nexus2/AuthoringTests.lean, model/Temporal/Feature/Nexus2/README.md]

### Approach
Compare focused property%/behavior%/query% forms in ordinary def declarations over the same checked constructors; semantic tests and conflicts are already established dependencies. Reuse source occurrence capture and diagnostic elaboration (Target/Language.lean:924,952) while producing kernel-checked checker-success evidence. Keep syntax outside FiniteMachine; use the narrow recorded AUT-07 prototype exception.

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

Reuse the preceding Property frontend/source/proof machinery. This slice adds only Behavior/Query surface adapters and their typed diagnostics, then performs the whole constructor/frontend comparison and selects the default example surface. Preserve the already-tested Property grammar/semantics.

Export reusable APIs through their existing owning facades and add the corresponding focused import checks when the public surface changes; keep new generic modules in the named owner. New tests must be imported into the named gate root immediately.

## Acceptance
- [ ] Actual compiled frontend specimens admit the same baseline, race, guarded conditions and cases as constructors with fixed semantic IDs/inputs. Compare checked meanings/fingerprints, selected traces and Query outcomes; separately explain legitimate source-location differences. No second evaluator or production migration is introduced.
- [ ] Diagnostic compilation tests freeze typed error kind, related IDs and exact source spans for malformed/duplicate IDs, wrong/missing references, missing capabilities, unknown state/result, unsupported syntax/operators, resulting/future guards, empty groups, contradictory scenarios, omitted/invalid units and Target mismatch. Retain parent/case/exception/clause provenance and publish no partial checked constant after error.
- [ ] Compiler-generated success evidence is kernel checked; audit transitive axioms for each trust-bearing Target/Property/Behavior/Query declaration and compare approved baselines. Test failed proof/admission paths; native diagnostics never become proof authority. If this route is impractical, retain tested successful-branch constructors and record measured cost instead of a silent native fallback.
- [ ] Measure compilation/admission time for baseline and 10x declarations and actual editor completion, hover, navigation and error recovery for both specimens. Name tool/environment and observed result; unavailable editor capabilities remain explicitly unmeasured, with no human-usability or final-grammar claim.
- [ ] Select one default Nexus2 surface only from actual semantic, diagnostic, trust and editor comparison evidence; preserve alternative test specimens and clearly label uncompiled illustrative syntax. README records the decision and remaining production AUT-07/AUT-08 reconciliation and human evaluation boundaries.

## Done summary
Implemented source-aware `behavior%` and `query%` frontends over the existing typed checkers, with public owner exports, open-term typed fallback, closed exact-span diagnostics, no partial checked values, and no native proof authority. Compiled constructor/frontend specimens now compare the complete baseline and guarded-race identities, semantic fields, fingerprints, selected traces, Query outcomes, source-only differences, contradictory-space status, and the full diagnostic/provenance matrix.

The kernel proof experiment retains explicit `checked` seams and checked `decide +kernel` failures for actual Behavior and Query inputs; the practical frontends return ordinary checker results whose axiom inventory matches the established `propext`, `Classical.choice`, and `Quot.sound` baseline. Equal-work 1x/10x admission guards and a matched Lean 4.33.1 LSP completion/hover/navigation/recovery matrix corrected the first review's measurement findings. Ordinary typed constructors remain the one Nexus2 default; compiled frontend alternatives remain isolated under the narrow AUT-07 prototype exception. Cold/repeated timings, real editor-client UI/latency, human readability/usability, final grammar approval, and production AUT-07/AUT-08 reconciliation remain explicitly unmeasured or future work.

Baseline: green via task17 handoff (Quick75 `/tmp/fn65-task17-review-fix-final-quick.log`; lint-model249 `/tmp/fn65-task17-review-fix-final-lint-model.log`). Final focused39, exact Quick75, and `make lint-model` passed. Go lint retained the inherited exit 2: all 1316 sorted diagnostic headers byte-match task17, SHA-256 `aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077`; no Go files changed in this task. User retained commit ownership, so HEAD remains `7774fdc7ac751ac959816c9829516ce54af57194` and changes remain staged.

Review receipt: `/tmp/impl-review-receipt-fn-65-design-and-prototype-approachable.18.json`; session `01a07446-8d2a-74c3-bf92-129e999c50a5`; reviewed owned tree `a106c4b7878c17300ec140dab3362c806abc1abc`. The first two findings were fixed; terminal verdict SHIP.

stage: impl-review - ran [2026-09-06T01:16:01Z..2026-09-06T01:26:18Z] (model: gpt-5.6-sol at medium)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: green via task17 handoff (Quick75 /tmp/fn65-task17-review-fix-final-quick.log; lint-model249 /tmp/fn65-task17-review-fix-final-lint-model.log), (cd model && mise exec -- lake build Temporal.Feature.Nexus2.AuthoringTests) (exit 0; /tmp/fn65-task18-review-fix-focused.log), (cd model && mise exec -- lake build Temporal.Feature.Nexus2.Tests Umpire.Property.Tests Umpire.Behavior.Tests Umpire.Query.Tests) (exit 0; /tmp/fn65-task18-review-fix-quick.log), make lint-model (exit 0; /tmp/fn65-task18-review-fix-lint-model.log), make lint-code GOLANGCI_LINT_FIX=false (inherited exit 2: 1316 sorted diagnostic headers exactly match task17, SHA-256 aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077, diff 0; /tmp/fn65-task18-final-lint-code.log), Lean 4.33.1 LSP matched constructor/frontend completion, hover, navigation, and recovery matrix (exit 0; /tmp/fn65-task18-lsp-matrix.log), Warm equal-work forced admission profiler: Property ctor .011164/.012127 vs frontend .011509/.012066; Behavior ctor .006103/.006932 vs frontend .006074/.006810; Query ctor .050297/.050475 vs frontend .049900/.050952 seconds at 1x/10x (/tmp/fn65-task18-admission-profile.log), official impl-review SHIP round 2; receipt /tmp/impl-review-receipt-fn-65-design-and-prototype-approachable.18.json; reviewed owned tree a106c4b7878c17300ec140dab3362c806abc1abc
- PRs: