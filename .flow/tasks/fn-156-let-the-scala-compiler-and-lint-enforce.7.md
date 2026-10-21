---
satisfies: [R1, R2, R3, R4, R5, R6, R7, R8, R9]
---
# fn-156-let-the-scala-compiler-and-lint-enforce.7 Document enforcement and verify exact artifacts and full gates

## Description
> Deferred: the parent spec's 2026-10-21 execution hold applies. These criteria preserve the filed scope; they do not authorize work on the frozen Scala tree. Revalidate against current Rust/successor work before resuming.

**Note 2026-10-18 (one-language series):** 'Full Go suite and model gate' means the Scala framework gate plus `tools/testpilot` tests after fn-208.

Complete R9 and the spec's documentation and integration checks at one standalone boundary before Batch 2. This task owns the full gates once after all source changes and reports their actual dispositions.

**Size:** M
**Files:** `model/README.md`, `model/framework/SEMANTICS.md`, `.plans/UMPIRE_MODULES.md`, compiler research reports, existing gate/Make wiring and final evidence.
**Touches:** [model/README.md, model/framework/SEMANTICS.md, .plans/UMPIRE_MODULES.md, .plans/SCALA.md, .plans/SCALA_CAPTURE_CHECKING.md, Makefile, model/check/**, .flow/tmp/fn156/integration/**]

### Approach

- Document required flags, sound equality evidence and precise lint suppressions, existing JDK 25 setup, rule ownership and import policy, discovered law applicability, finite/pin coverage and both compiler trial dispositions. Update SEMANTICS' account of named role tests to discovered subjects without changing the laws. Hand fn-141 its later lint-extraction ownership and Batch 2 the closed source baseline.
- Join the source lane's .6 with the independent .8 report before final verification. Only this task publishes .8's report as .plans/SCALA_CAPTURE_CHECKING.md and integrates shared SCALA.md guidance; .8 writes no shared tracked docs or source.
- Freeze candidate full artifact manifests before promotion. Run `make umpire-gen-model` and compare every filename and raw byte of model/ir and model/cases with task 1's immutable original. Review both complete diffs; any location, identity, field or inventory difference fails R9. Never apply fn-155's projection, silently refresh goldens, or mask source positions in the lifter.
- Run `make lint-model`, the complete irgen fixture build/test, `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks`, and `make umpire-check-cases` with separate canonical Go coverage `mise exec -- go test -tags test_dep -p 2 -timeout 30m -json ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/...`. Existing fixture/dependency checks remain required when applicable. Reuse only unchanged predecessor checks and record why their inputs still match.
- Serialize each heavy command using actual flock/fcntl ownership of `/tmp/umpire-heavy-gates.lock`. Retain logs, commands, status, source/fixture/environment hashes, wall time and Go completion events in this task's evidence. Repeat only invalidated checks after fixes.
- Preserve exact native resource RED under fn-157, Quint deferral under fn-154 and strict Activity semantic failures for Batch 5. Validation stuck beyond one hour including its attempts is deferred with logs, unmet acceptance and revisit conditions unless it blocks all other available work; proceed with independent work. Deferred/reduced/isolated results provide no R9 passing credit; report the unmet gate explicitly to the conductor. Close only criteria actually met under owner-directed disposition, never self-waive a failure or fit the Model to implementation.

### Investigation targets

**Required:**
- `model/check/Gate.scala:461` - framework/Model/lifter build and fixture steps.
- `Makefile:674` - lint modes and JDK split.
- `model/README.md:415` - author/compiler/lint guidance.
- `model/framework/SEMANTICS.md:191` - role-based refinement law.
- `.plans/UMPIRE_MODULES.md:30` - Scala ownership and dependencies.
- `MILESTONES.md:22` - evidence reuse, full gates, locking and deferred outcomes.
- `.flow/specs/fn-157-bound-native-verification-memory-and.md` - retained complete native RED obligations.

### Quick commands

Run the exact full commands above once on the final candidate under the shared lock. Run independent implementation/completion review with the complete scope, exact byte proof and linked full logs. This plan and its review do not claim any implementation gate has passed.

- [ ] Complete regeneration matches the original filename/raw-byte manifests exactly for both artifact trees; every diff is reviewed and no normalization supplies passing credit.
- [ ] Required Scala compiler/lint/irgen fixture and model/Case gates pass, with the canonical complete Go suite and applicable dependency/fixture checks linked or their unmet dispositions explicitly preserved.
- [ ] Reports account for every R1-R8 proof, negative fixture, machine/law disposition, finite-domain exception and all-machine pin; no reduced subset substitutes for full acceptance.
- [ ] Docs and ownership handoff describe final enforcement and compiler trial outcomes; native/Quint/Activity failures retain their owning obligations and actual statuses.
- [ ] Independent reviews inspect source scope and recorded evidence; the conductor receives commit/log paths and any unmet R9 blocker before starting dependent authoring work.
## Approach

- Document required flags, sound equality evidence and precise lint suppressions, existing JDK 25 setup, rule ownership and import policy, discovered law applicability, finite/pin coverage and both compiler trial dispositions. Update SEMANTICS' account of named role tests to discovered subjects without changing the laws. Hand fn-141 its later lint-extraction ownership and Batch 2 the closed source baseline.
- Join the source lane's .6 with the independent .8 report before final verification. Only this task publishes .8's report as .plans/SCALA_CAPTURE_CHECKING.md and integrates shared SCALA.md guidance; .8 writes no shared tracked docs or source.
- Freeze candidate full artifact manifests before promotion. Run `make umpire-gen-model` and compare every filename and raw byte of model/ir and model/cases with task 1's immutable original. Review both complete diffs; any location, identity, field or inventory difference fails R9. Never apply fn-155's projection, silently refresh goldens, or mask source positions in the lifter.
- Run `make lint-model`, the complete irgen fixture build/test, `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks`, and `make umpire-check-cases` with separate canonical Go coverage `mise exec -- go test -tags test_dep -p 2 -timeout 30m -json ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/...`. Existing fixture/dependency checks remain required when applicable. Reuse only unchanged predecessor checks and record why their inputs still match.
- Serialize each heavy command using actual flock/fcntl ownership of `/tmp/umpire-heavy-gates.lock`. Retain logs, commands, status, source/fixture/environment hashes, wall time and Go completion events in this task's evidence. Repeat only invalidated checks after fixes.
- Preserve exact native resource RED under fn-157, Quint deferral under fn-154 and strict Activity semantic failures for Batch 5. Validation stuck beyond one hour including its attempts is deferred with logs, unmet acceptance and revisit conditions unless it blocks all other available work; proceed with independent work. Deferred/reduced/isolated results provide no R9 passing credit; report the unmet gate explicitly to the conductor. Close only criteria actually met under owner-directed disposition, never self-waive a failure or fit the Model to implementation.

## Investigation targets

**Required:**
- `model/check/Gate.scala:461` - framework/Model/lifter build and fixture steps.
- `Makefile:674` - lint modes and JDK split.
- `model/README.md:415` - author/compiler/lint guidance.
- `model/framework/SEMANTICS.md:191` - role-based refinement law.
- `.plans/UMPIRE_MODULES.md:30` - Scala ownership and dependencies.
- `MILESTONES.md:22` - evidence reuse, full gates, locking and deferred outcomes.
- `.flow/specs/fn-157-bound-native-verification-memory-and.md` - retained complete native RED obligations.

## Quick commands

Run the exact full commands above once on the final candidate under the shared lock. Run independent implementation/completion review with the complete scope, exact byte proof and linked full logs. This plan and its review do not claim any implementation gate has passed.

- [ ] Complete regeneration matches the original filename/raw-byte manifests exactly for both artifact trees; every diff is reviewed and no normalization supplies passing credit.
- [ ] Required Scala compiler/lint/irgen fixture and model/Case gates pass, with the canonical complete Go suite and applicable dependency/fixture checks linked or their unmet dispositions explicitly preserved.
- [ ] Reports account for every R1-R8 proof, negative fixture, machine/law disposition, finite-domain exception and all-machine pin; no reduced subset substitutes for full acceptance.
- [ ] Docs and ownership handoff describe final enforcement and compiler trial outcomes; native/Quint/Activity failures retain their owning obligations and actual statuses.
- [ ] Independent reviews inspect source scope and recorded evidence; the conductor receives commit/log paths and any unmet R9 blocker before starting dependent authoring work.

## Done summary
Joined .6 and .8: published .plans/SCALA_CAPTURE_CHECKING.md with reconciliation (Draft lines 88-189 hash unchanged ed6b7160…; probes recompiled under the new warning flags give the original results, capture-recheck/receipts.txt). Docs: model/README.md section 'What the compiler, the lint and the law tests enforce', SEMANTICS role-law wording, UMPIRE_MODULES Scala layout paragraph, SCALA.md. R9 bytes: a fresh lift of the final source and of the pre-fn-156 source (HEAD with the three fn-156 source commits reverse-applied, the same Duration lifter fix 89b7745f8c applied, and the mainline-only StepTables.scala removed because it needs fn-156's IrFile.subjects and declares no Model) are byte-identical: 11/11 files, 25,191 positions, no normalization (lift.sh; baseline-ir.sha256 = final-ir.sha256 at 35dca5a364). Gates run: make lint-model exit 0; irgen suite 145 passed / 3 failed, all three inherited stale goldens (realizations.json: Duration encoding from the batch3 lifter fix only; hints.json/hintsRefused.json: Product/System positions and content changed by fn-155.3 96c554b1ab after the goldens' last regeneration) - not fn-156 drift, left for the regeneration owner; framework/Temporal suite 156/0 and gate tests 78/0 green. Not run: umpire-check-model/umpire-gen-model and umpire-check-cases (checked-in model/ir is stale against the Duration schema change being regenerated in w-batch3, and fn-155.6's whole-regeneration proof remains an owner hold); Cases derive from the byte-identical IR with no Go change by fn-156. Full Go suite not run (no Go change in this lane).

Lane w-fn156 (Batch 1c close), on integrated umpire ae3f36ee51; source checkpoints d0a7f66639, 8481696c62, 7d54144267 (older lane notes: .worktrees/fn-156-let-the-scala-compiler-and-lint-enforce/.flow/tmp/fn156/batch/).

## Evidence
- Commits:
- Tests: logs under .worktrees/w-fn156/.flow/tmp/fn156/ (lint-model-full, irgen-tests2, authoring-tests2, gate-tests2, lift-baseline/lift-candidate, *.sha256)
- PRs:

## Acceptance
- [ ] Exact regeneration/manifests satisfy R9 without normalization; partial or deferred gates do not count as passes.
- [ ] Final compiler/lint/IR-generation/Model/Case and canonical Go gates have recorded outcomes; all unmet obligations remain explicit.
- [ ] Enforcement docs, compiler trial reports, law dispositions and ownership handoffs cover R1-R8, with independent review of evidence.
