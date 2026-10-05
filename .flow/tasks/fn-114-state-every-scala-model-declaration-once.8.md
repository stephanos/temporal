---
satisfies: [R1, R8, R9, R11]
---
# fn-114-state-every-scala-model-declaration-once.8 Close fn-114 with literal and line counts and full gates

## Description
Count, classify and gate once at the end, and update the module map for the new root ownership.

**Size:** S
**Files:** `.flow/tmp/fn114-8/**` evidence; small literal fixes in Models; `.plans/UMPIRE_MODULES.md` ("Root lists keep their present meaning until fn-114"), `model/README.md`.
**Touches:** [model/temporal/features/nexuscaller/**, model/temporal/shared/worker/**, model/README.md, .plans/UMPIRE_MODULES.md, .flow/tmp/fn114-8/**]

### Approach
- Re-run task 1's literal and line counting command; classify each remaining literal into fn-112 R18's three kinds; list any other literal with its line and reason (R11). Remove literals that turn out to be avoidable.
- Run the full model gate, `make lint-model`, the Umpire Go tests and `make lint-code-fast` once, with `-json` timing per the MILESTONES verification instructions; record commands, results and log paths.
- Report line counts of Models and the gate program before (task 1) and after (R9), and the lift-stage times from task 1 (R8).

### Quick commands
```bash
make umpire-check-model && make lint-model && make lint-code-fast
go test -count=1 -json -tags test_dep ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... > .flow/tmp/fn114-8/go-test.json
```

### Execution constraints
- fn-120.2 (refuse unnamed branches), fn-118's behavior phase and fn-119 wait for this spec to close; say so in the done summary so the conductor can release them.
## Acceptance
- [ ] Every remaining Model string literal is classified into the three allowed kinds or listed with line and reason.
- [ ] Model and gate-program line counts before/after are in the done summary, with the lift-stage times.
- [ ] Model gate, lint-model, Umpire Go tests and lint-code-fast pass in full; R1 goldens pass.
- [ ] Module map and README describe Scala-owned roots.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
