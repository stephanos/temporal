---
satisfies: [R3, R8, R9]
---
# fn-78-typed-temporal-authoring-and-checked.4 Build the checked evidence-projection kernel

## Description
Define D3's generic checked command/event ownership and evidence-projection kernel. Provide one bounded, run-local, transactional admission interface that turns declared source evidence into pending, stutter, emitted semantic-step, or rejected results without importing Temporal or Nexus meaning into Umpire's generic owners.

**Size:** L
**Files:** `model/Umpire/Target/{Language,FiniteMachine}.lean`, `model/Umpire/Observation/{Declaration,Evaluation/**,Tests/**}.lean`, new focused projection modules under `model/Umpire/Observation/`
**Touches:** [model/Umpire/Target/Language.lean, model/Umpire/Target/FiniteMachine.lean, model/Umpire/Observation/**]

### Approach
- Consume task 2's explicit Target terminal seam and reuse the kernel's Action/Outcome representation while making controllable submissions and observed confirmed outcomes explicit at the checked boundary; coordinate with fn-75's semantic facade if it has landed.
- Define closed projection declarations for scope keys, stable source-event identities, causal references, semantic outputs, evidence-field policy, and configurable buffer/key/support/work ceilings.
- Stage deduplication, causal closure, transition validation, emitted steps, and exact transitive support before committing an append. A rejection commits none of that append's releases.
- Keep accepted events immutable and preserve prior semantic state, diagnostics, and proved violations across later rejected inputs.
- Admit only declared/authorized evidence fields and redaction dispositions; raw runtime evidence remains outside Property and generic semantic APIs.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Target/FiniteMachine.lean:420-496` — checked Target seam
- `model/Umpire/Observation/Declaration.lean` — current evidence declarations and bounds
- `model/Umpire/Observation/Evaluation/Admission.lean:516-577` — checked whole-trace admission
- `model/Umpire/Observation/Evaluation/Structure.lean` — causal ordering/closure analysis
- `model/Umpire/Observation/Tests.lean` — focused test root

### Key context
- Equal Target states alone are insufficient for merging projection or monitor progress.
- Missing parents are pending; irrelevant or duplicate admitted evidence stutters; known cycles, identity conflicts, unsupported relevant evidence, and invalid semantic transitions reject.
## Acceptance
- [ ] The checked boundary distinguishes authorized command submission from confirmed Target-owned semantic events without replacing the Target's Action/Outcome authority.
- [ ] Projection exposes a deterministic bounded `admit`/`close` surface with explicit pending, stutter, one-or-more emission, and typed rejection results.
- [ ] Each emitted step retains exact direct and transitive supporting evidence plus declared scope identities; source order and causal references determine order, never cross-source wall time.
- [ ] Rejected appends emit no new steps and leave accepted events, buffers, semantic state, prior diagnostics, and prior violations unchanged.
- [ ] Tests cover duplicates, irrelevant evidence, missing parents, multiple released steps, cycles, conflicts, unsupported evidence, invalid transitions, wrong scope/operation, immutable accepted events, and buffer/key/support/work exhaustion.
- [ ] Fresh projector instances isolate repeated/concurrent Runs, and tenfold evidence/overlapping-key loads either succeed within declared ceilings or fail closed deterministically.
- [ ] Raw evidence cannot enter Property evaluation or bypass checked projection; compile-failure and axiom-audit baselines pass.
## Done summary
Implemented a closed, Target-bound, run-local evidence projector under Umpire.Observation.Projection. Authorized submissions stutter; confirmations emit proof-carrying Target steps. Admission stages causal/source-order release, per-operation state, support, and ceilings atomically; an error returns neither replacement Run state nor new emissions. Existing Property verdicts and diagnostics remain consumer-owned and are never revised by projection errors.

Status: completed after conductor review; changes remain uncommitted for the user.

Owned files:
- model/Umpire/Observation/Projection/Declaration.lean
- model/Umpire/Observation/Projection.lean
- model/Umpire/Observation/Tests/Projection.lean
- model/Umpire/Observation/Tests/ProjectionBoundary.lean
- model/Umpire/Observation.lean (facade import)
- model/Umpire/Observation/Tests.lean (test imports)

The public seam is check/start/admit/close, with immutable accepted/steps/pending/state/work/isClosed views. Declaration data binds version, scope field names, operation field, sources, evidence kind/field dispositions, command/confirmation ownership, initial setup/state, Target fingerprint, and independent limits into canonical provenance. Source identities contain scope + source + ordinal; each source event separately records supporting Run Event sequences. Duplicate equality includes those immutable provenance sequences; changed provenance under the same scoped source identity rejects rather than revising prior support. Each emission retains direct/transitive source identities and exact Run sequence unions. Source order and causal references drive release; wall time is absent. Target terminal declarations govern closure.

Tests cover submissions, duplicate/irrelevant stutter, missing parents, source gaps, simultaneous buffered releases, multiple outputs from one source event, exact direct/transitive support, cycles, identity and operation conflicts, cross-run/scope/source rejection, unsupported evidence, missing submission, invalid transitions, incomparable cross-source confirmation, redaction/retention/rejection policy, immutable accepted events/provenance, transactional rejection after prior steps, late support exhaustion, exact work ceilings including duplicates, all configured ceilings, 10 vs 100 events/overlapping operation keys, fresh/repeated run isolation, explicit terminal/nonterminal/incomplete/closed behavior, canonical order invariance, version rejection, compile-failure seals, and guarded axiom baselines. Two behavioral regressions were observed red before fixes: incomparable source ordering and uncharged duplicate work.

Baseline: green, LEAN_NUM_THREADS=2 make umpire-build-model, 483 jobs (/tmp/fn78-task4-baseline-build.log).

Shared DefinitionId validation now checks projection/scope/operation/source/kind/field IDs. Its six-case regression was observed red before the fix (/tmp/fn78-task4-id-red.log). Final full build and one-thread model lint retry passed.
- Focused Observation and Target suite passed after the ID guard, 73 jobs (/tmp/fn78-task4-final-focused.log).
- Last projection/boundary suite passed, 46 jobs (/tmp/fn78-task4-last-focused.log).
- Final full model build passed after all edits, 487 jobs (/tmp/fn78-task4-final-build-after-id.log); exit 0.
- make lint-code GOLANGCI_LINT_FIX=false: inherited failure, 1,284 existing issues, exit 2 (/tmp/fn78-task4-lint-code.log); matches task .2's baseline. No Go files changed. The recipe's later go-vet step did not run.
- LEAN_NUM_THREADS=2 make lint-model: the OS killed the final builtin-lint process with signal 9 after its 351-module rebuild; make reported Error 137, wrapper exit 2 (/tmp/fn78-task4-lint-model.log). No Lean diagnostics were emitted. The one-thread retry passed after the final build.
- LEAN_NUM_THREADS=1 make lint-model: passed, exit 0, including all 351 builtin-lint build jobs and the aggregate lint pass (/tmp/fn78-task4-lint-model-single-thread.log).
- git diff --check passed after all source edits.

Trust: check/admit/close use the same [propext, Classical.choice, Quot.sound] transitive baseline as validateEvidenceBackedTrace and evaluateProperty; Step.semantic is axiom-free. Tests guard those exact inventories. No new custom/compiler-trust axiom, toolchain, or dependency enters the load-bearing path.

Limits: work is an explicitly documented conservative finite graph-comparison reservation, not elapsed CPU time. Finite input/retention ceilings remain independent. Version one supports retain/redact/reject fields and explicitly rejects hash dispositions; hashing remains with the existing Observation owner. Raw SDK history, Profile/Driver authorization, semantic Property/monitor evaluation, Nexus mapping, and portable lowering stay with their existing/downstream owners. Run Event provenance must be supplied by the authorized adapter; no cryptographic source authentication is added. No Temporal/Nexus meaning, protocol changes, fixture promotions, shared Flow lifecycle mutations, commits, or worktrees were introduced.

stage: impl-review - passed(gpt-6-astra medium; mixed-edge finding fixed; SHIP)

Base commit: 75482e5cd36e77a405d5e27c865d405cc0918fd4
HEAD: 75482e5cd36e77a405d5e27c865d405cc0918fd4
Commits: none; user-owned uncommitted implementation.

Conductor-review follow-up (P2 mixed-edge ordering):
- The same-operation check now traverses ordering reachability over both causal references and source-order edges. It uses a bounded visited-array queue and shares its edge predicate with cycle validation. No evidence-support or provenance accumulation changed.
- Added direct and buffered/reordered A0 -> A1 -> B0 -> B1 regressions. Both emit the same Target states while B1 retains only its own direct/transitive causal identity and Run sequence 201; B0 and A1 are ordering predecessors, not extra evidence support.
- Regression observed red before the fix (/tmp/fn78-task4-review-red.log), then the complete focused Observation/Target suite passed 73 jobs (/tmp/fn78-task4-review-focused.log).
- Follow-up files: model/Umpire/Observation/Projection.lean; model/Umpire/Observation/Tests/Projection.lean.
- Post-fix full build passed 487 jobs; one-thread lint completed its 351-module aggregate run without diagnostics. Logs: /tmp/fn78-task4-review-build.log and /tmp/fn78-task4-review-lint-model.log. Reviewer confirmed SHIP; its R9 execution limitation is covered by these actual-workspace logs.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: green - LEAN_NUM_THREADS=2 make umpire-build-model (483 jobs; /tmp/fn78-task4-baseline-build.log), cd model && mise exec -- lake build Umpire.Observation.Tests Umpire.Target.Tests.FiniteMachine Umpire.Target.Tests.FiniteTable Umpire.Target.Tests.Compatibility (passed, 73 jobs; /tmp/fn78-task4-final-focused.log), LEAN_NUM_THREADS=2 make umpire-build-model (passed after all edits, 487 jobs; /tmp/fn78-task4-final-build-after-id.log), LEAN_NUM_THREADS=2 make lint-model (OS killed builtin-lint process with signal 9 after 351-module rebuild; make Error 137, exit 2; /tmp/fn78-task4-lint-model.log), LEAN_NUM_THREADS=1 make lint-model (passed, exit 0; /tmp/fn78-task4-lint-model-single-thread.log), make lint-code GOLANGCI_LINT_FIX=false (inherited failure, 1284 unchanged Go lint issues, exit 2; later go-vet recipe step not reached; /tmp/fn78-task4-lint-code.log), git diff --check (passed), ProjectionBoundary guarded compile-failure and axiom audits (passed as part of model build), Projection regression red-to-green: incomparable source order and duplicate work charging (/tmp/fn78-task4-behavior-red.log), Projection DefinitionId regression red-to-green (/tmp/fn78-task4-id-red.log), Post-review focused Observation/Target suite passed 73 jobs (/tmp/fn78-task4-review-focused.log), Post-review full model build passed 487 jobs (/tmp/fn78-task4-review-build.log), Post-review LEAN_NUM_THREADS=1 make lint-model completed aggregate lint without diagnostics (/tmp/fn78-task4-review-lint-model.log), gpt-6-astra medium implementation re-review: SHIP, mixed-edge finding fixed
- PRs: