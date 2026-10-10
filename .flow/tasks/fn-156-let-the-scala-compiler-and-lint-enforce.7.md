---
satisfies: [R1, R2, R3, R4, R5, R6, R7, R8, R9]
---
# fn-156-let-the-scala-compiler-and-lint-enforce.7 Document enforcement and verify exact artifacts and full gates

## Description

Complete R9 and the spec's documentation and integration checks at one standalone boundary before Batch 2. This task owns the full gates once after all source changes and reports their actual dispositions.

**Size:** M
**Files:** `model/README.md`, `model/SEMANTICS.md`, `.plans/UMPIRE_MODULES.md`, compiler research reports, existing gate/Make wiring and final evidence.
**Touches:** [model/README.md, model/SEMANTICS.md, .plans/UMPIRE_MODULES.md, .plans/SCALA.md, .plans/SCALA_CAPTURE_CHECKING.md, Makefile, model/check/**, .flow/tmp/fn156/integration/**]

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
- `model/SEMANTICS.md:191` - role-based refinement law.
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
TBD

## Evidence
- Commits:
- Tests:
- PRs:

## Acceptance
- [ ] TBD
