---
satisfies: [R2, R3, R4]
---

# fn-20-local-execution-semantic-conformance.2 Build the exact Temporal Nexus Implementation Link evaluator

## Description
### Umpire4 reconciliation (normative)

The Temporal checker composes `Temporal.System.Nexus.Observation` with `Temporal.System.Nexus.ImplementationLink`; it must never map raw evidence directly to Feature facts. Unknown, unsupported, unaccepted, or failed Implementation Link remains distinct from a Property result.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Implement the fixed Lean side of the private checker bridge and the sole live-evidence adapter (R2/R3/R4/R5), using Task `.1` as the semantic entry point.

**Size:** M
**Files:** `model/Temporal/Tool/RunEvaluation/Protocol.lean`, `model/Temporal/Tool/RunEvaluation.lean`, `model/Temporal/Tool/RunEvaluationTests.lean`, `model/Temporal/Feature/Nexus/Observation.lean`, `model/lakefile.toml`
**Touches:** [model/Temporal/Tool/RunEvaluation/Protocol.lean, model/Temporal/Tool/RunEvaluation.lean, model/Temporal/Tool/RunEvaluationTests.lean, model/Temporal/Feature/Nexus/Observation.lean, model/lakefile.toml]

### Approach
- Register the closed checker identity/version/digest and exactly one caller-closure declaration closure; resolve every request identity against compiled checked values.
- Decode only the private direct Generated View with the four exact non-path admitted artifact-binding tuples, separate Run/RawEvidence Known Gaps, exact canonical request shape, and Limits; never read a file, manifest, artifact member, environment option, or arbitrary extension.
- Freeze the fn-19 source schema/version/digest table after that dependency lands and translate its four source kinds into fn-4's typed EvidenceBundle while preserving order, causality, gaps, closure, correlations, and dispositions.
- Call Task `.1`; then compose its Observation Evaluation/verdict Generated View with the exact compiled ExperimentSpec plan and checked program/mapping/query/Property values to compute fn-18's `evaluationOutcomeChecksum` in the Lean authority.
- Emit mapping/Observation Evaluation-only `observationKnownGaps` and the canonical exact-value union `resultKnownGaps` from request Run Known Gaps, RawEvidence Known Gaps, and semantic Known Gaps; keep unknown/conflict/unsupported distinct from protocol failure.
- Register `temporal-run-evaluation-checker` and prove stdin/stdout/stderr bytes, exit behavior, request/response N/N+1 Limits, and deterministic repeated checking.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-4-umpire-observation-and-semantic-verdicts.4.md` — Temporal-owned mapping/profile seam
- `model/Temporal/Feature/Nexus/CallerClosure.lean:441-471` — exact checked Property subject
- `model/Temporal/Feature/Nexus/CallerClosureTests.lean` — current scenario fixture/testing pattern
- `.flow/tasks/fn-19-bounded-local-temporal-execution-and.7.md` — exact four-source producer
- `.flow/tasks/fn-18-versioned-umpire-artifact-boundary.6.md` — plan-sensitive outcome identity and Known Gap projections
- `model/lakefile.toml:1-20` — executable registration pattern
## Acceptance
- [ ] The checker accepts only the exact compiled caller-closure experiment/program/mapping/query/Property/source closure and exact echoed artifact bindings; every drift rejects deterministically.
- [ ] Four-source mapping preserves source-local and causal facts; incomplete/ambiguous/conflicting/unsupported/disposition cases receive the correct fn-4 outcome without guessed order.
- [ ] Accepted-outcome identity includes the compiled ExperimentSpec plan plus every fn-18 stable semantic input and excludes only the specified transport/run fields.
- [ ] Semantic Known Gaps and the canonical Result Known Gap union follow the parent contract byte-for-byte and preserve upstream auditability through bound artifacts.
- [ ] The executable performs no filesystem, network, environment-authority, artifact admission/publication, or Temporal runtime operation.
- [ ] Canonical protocol, 32-MiB N/N+1, and no-stderr deterministic success tests pass.
## Done summary
Implemented the v2 local run-evaluation boundary end to end: bounded canonical protocol input, lossless multi-source/source-local evidence with checked Observation semantics, exact Implementation Link/Feature projection, Lean-owned accepted checksum, and DrivePlan-bound Result checksums. The review redesign also closes every actual source fact/field/type/disposition/digest, preserves known-gap uncertainty, admits valid fn-19 non-success prefixes, and projects every non-accepted status to a fn-18-valid empty incomplete Result.

Added the narrow Darwin prerequisite in commit 480138d20 by replacing unsupported `syscall` descriptor operations with existing `x/sys/unix` equivalents. Baseline was red before task edits because task-owned checker/Go/Make targets did not yet exist and `make umpire-check-regression` failed to compile `syscall.Openat`; after the prerequisite, its focused package and full regression passed with physical Darwin temporary paths. The default Darwin `/var` alias still makes the unmodified temp environment fail artifact containment, so final regression evidence records both that inherited environment result and the green physical-path invocation.

The bounded review loop resumed the same Codex session and fixed all six findings; the final receipt reports SHIP with zero introduced/pre-existing findings and no unaddressed R-IDs. Memory capture was required and attempted after NEEDS_WORK to SHIP, but skipped non-blockingly because the enabled repository memory store is not initialized.

stage: impl-review - ran [2026-08-30T00:09:32Z..2026-08-30T01:11:54Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 480138d20038410177f14a1ef5f38c571d1afea9, 5d195041a20dffa94da168958db3a9e188c0001d, dd1634eeda76759acb7ede6f03d65965aab4404f, d01423f15250827bab1987cbe38e1ccb1af1dcbc
- Tests: baseline: red (task-owned temporal-run-evaluation-checker, Go adapter/CLI packages, and Make target absent pre-edit; make umpire-check-regression failed pre-edit because syscall.Openat is unavailable on Darwin), RED_EXPECTED: go test -count=1 ./tools/common/artifactio (undefined syscall.Openat on Darwin), TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-tmp go test -count=1 ./tools/common/artifactio, RED_EXPECTED: cd model && mise exec -- lake build Temporal.Tool.RunEvaluationTests temporal-run-evaluation-checker (smallest Protocol + RunEvaluation behavior tests), cd model && mise exec -- lake build Umpire.Observation.Tests.EvidenceLink Umpire.Artifact.Tests.Result Umpire.Artifact.Tests.Set Temporal.Tool.RunEvaluationTests temporal-run-evaluation-checker, go test -count=1 ./tools/umpire/runevaluation/..., go test -count=1 ./tools/umpire/cmd/umpire-local-run-evaluation/..., make umpire-check-local-run-evaluation SET=tools/umpire/temporal/nexus/testdata/caller-closure-run-set OUTPUT_ROOT=/tmp/umpire-local-results, temporal-run-evaluation-checker stdin boundary: 33554432 bytes => non-canonical, 33554433 bytes => oversized, no stdout, inherited environment: make umpire-check-regression (Lean build green, then Darwin /var symlink containment failure), TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-tmp PATH=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/bin:$PATH make umpire-check-regression, cd model && mise exec -- lake lint, impl-review codex session 01a04ffb-4503-73a3-ae68-b7372def0208: SHIP
- PRs:
