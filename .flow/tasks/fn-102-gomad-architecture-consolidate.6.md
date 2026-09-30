---
satisfies: [R6]
---
# fn-102-gomad-architecture-consolidate.6 Reconcile architecture documentation and qualify the integrated refactor

## Description
R6. Reconcile SPEC PLATFORM.SUPPORT, ARCHITECTURE verification scope/research claims, README classic-GC claims and historical SIM-0/SIM-1 glossary wording. Preserve stable IDs and distinguish supported platform/capability from workload-qualified and exact replay. Consult latest fn-99/fn-100/fn-101 evidence at execution time; do not freeze today's milestone state into future docs. Explain new private policy owners and public executor migration. Run full Gomad gates on both qualified hosts/CI and root development lint, recording actual evidence; do not mark blocked platform checks passed. Consume task 3's 10/100-job deterministic bound evidence and inspect the integrated data flow for new seed-count-proportional state or full-payload copies. No quantitative performance baseline or throughput/RSS claim is part of this refactor. Reuse existing CI, no new service. Preserve existing comments. No new dependencies. Run focused commands from tools/gomad3 with GOWORK=off and -tags test_dep; use the patched toolchain for runtime-consumer tests.

**Size:** M

**Touches:** [tools/gomad3/README.md, tools/gomad3/ARCHITECTURE.md, tools/gomad3/SPEC.md, tools/gomad3/GLOSSARY.md, tools/gomad3/Makefile, .github/workflows/*gomad*] — WIDER: existing gate wiring only if required; no unrelated CI changes.

**Files:** `tools/gomad3/{README.md,ARCHITECTURE.md,SPEC.md,GLOSSARY.md}`; existing verification/CI wiring only if required for new checks.

### Quick commands

`make -C tools/gomad3 validate`
`make -C tools/gomad3 test`
`make -C tools/gomad3 core-qualification`
`make lint-code-fast`
Run the same Gomad gates on darwin/arm64 and linux/amd64; use existing CI for the other host.

## Acceptance
- [ ] R6 docs match current implementation and dated qualification evidence, with stable SPEC IDs.
- [ ] Public Go source-level executor migration is documented without claiming universal API compatibility.
- [ ] Focused and integrated validation/generation/lint have retained outcomes on both platforms; blockers remain explicit.
- [ ] Task 3's 10/100-job control-bound evidence and integrated data-flow review verify unchanged hard bounds; no unmeasured performance claims appear in docs.

## Done summary
NOT IMPLEMENTED. Moved to fn-105-gomad-follow-ups-deferred-scope.5 (D5) on 2026-09-29 as a scope cut; the task text above remains the implementation brief.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: