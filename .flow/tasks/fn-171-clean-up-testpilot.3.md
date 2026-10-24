---
satisfies: [R3]
---
# fn-171-clean-up-testpilot.3 Tidy: current docs, package docs, test-only exports, generated-case loaders to lower, canary preflight import

## Description
> Deferred: the parent spec's 2026-10-21 execution hold applies. These criteria preserve the filed scope; they do not authorize work on the frozen Scala tree. Revalidate against current Rust/successor work before resuming.

**Recut 2026-10-18 (one-language series):** Drop the loader move to `tools/umpire/lower` (deleted by fn-208); generated-case loaders go to `tools/testpilot`. The docs part stays.

From the Testpilot research (2026-10-12).
1. Docs: `temporal/README.md:8,63` and `tests/testcore/testpilot/README.md:32` still say Case 1.0, but 4.0 is the only admitted format. Move the root README's CLI exit-code section (about lines 525-593) to `tools/umpire/README.md`, and consider splitting its protocol-extension checklist (about lines 270-495) into its own doc. Add package docs to `replay`, `temporal/worker`, `temporal/control`, `temporal/internal/{delivery,activation}`, `duration` and `tests/testcore/testpilot`. Fix `.plans/UMPIRE_MODULES.md:65`, which lists `casefile.ErrNoncanonical`/`Indent`; the code has `FormatError`/`CheckVersion`/`JSONVersion`.
2. Move `tests/testcore/testpilot`'s test-only exports (`NexusCallerProfile`, `WorkflowStartProfile`, `NexusPairProfile`, `LoadModelCase`, `ResolveSettings`) to `_test.go`.
3. Move `GeneratedCases`/`LoadGeneratedCase`/`GeneratedCaseName` (which wrap `lower.GeneratedCase`) into `tools/umpire/lower`, and make `tools/canary/preflight/harness_test.go` import them from there, not from `tests/testcore/testpilot` (contradicting `UMPIRE_MODULES.md:73`).
4. Optional, size only: the oversized files `execution/scheduler.go` (1,316 lines), `execution/dataflow.go`, `delivery/ledger.go`, `assessment.go` and `execution/prepare.go`.
## Acceptance
- [ ] Current docs name the actually admitted artifact/version and existing exported interfaces; historical Case guidance is not restored after fn-228.
- [ ] Surviving packages have the described package docs; test-only exports remain test-only with all consumers compiling.
- [ ] Generated-loader ownership and canary imports follow the surviving successor tooling, with no test-package production dependency; retired loaders are not recreated.
- [ ] Scope remains behavior-preserving; optional file splitting is not required for completion.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: