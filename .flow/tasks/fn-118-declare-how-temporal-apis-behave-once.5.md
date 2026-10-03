---
satisfies: [R4, R6, R8]
---
# fn-118-declare-how-temporal-apis-behave-once.5 Migrate realizations to derived waits and close fn-118 with Contract-freeze evidence

## Description
Delete the hand-written waits from both realizations, regenerate Cases, prove every Contract unchanged with the Program differences listed, and run the closing gates.

**Size:** M
**Files:** `model/temporal/standaloneactivity/Realization.scala` (`awaitStatus` and its callers), `model/temporal/nexuscaller/Realization.scala` (the two 250 ms polls and the two `timeoutMs = 5000` commands; line numbers in task 1's inventory predate fn-112.9 and fn-114.3, so re-locate them), `model/ir/**`, `model/cases/**`, lowering migration goldens' allowed-Program-delta list, `model/README.md`, `model/SEMANTICS.md`, `.plans/API_BEHAVIOR_HINTS.md` (after-numbers), `.flow/tmp/fn118-5/**`.
**Touches:** [model/temporal/standaloneactivity/Realization.scala, model/temporal/nexuscaller/Realization.scala, model/temporal/realize/**, model/ir/**, model/cases/**, tools/umpire/lower/testdata/**, tools/umpire/model/testdata/**, model/README.md, model/SEMANTICS.md, .plans/API_BEHAVIOR_HINTS.md, .flow/tmp/fn118-5/**]

### Approach
- Replace each explicit poll/interval/timeout covered by a hint with the plain read + condition; a wait no hint covers keeps its explicit form and is listed with its reason (R4). Keep task 4's final rule for explicit polls (accepted only with a recorded reason).
- Regenerate (`make umpire-gen-model`); compare against the baseline goldens: every Contract byte-identical, every Program difference listed (R6). A changed Contract stops the task.
- Recount polls and total declared wait budget with task 1's command; list candidate hints not adopted with the Case that would need each (R8).
- Run model gate, lint-model, Umpire and Testpilot Go tests, lint-code-fast and the live Case tests once (`make umpire-check-live-tests`).

### Investigation targets
**Required:**
- `.plans/API_BEHAVIOR_HINTS.md`
- both realization files above
- `tools/umpire/lower/migration_golden_test.go`

### Quick commands
```bash
make umpire-gen-model && git diff --stat model/ir model/cases
make umpire-check-model && make umpire-check-live-tests
```

### Execution constraints
- Contracts frozen; only Program waiting may change.
## Acceptance
- [ ] No realization contains a literal interval or timeout or an explicit poll a hint covers; remaining explicit waits are listed with reasons.
- [ ] Every existing Contract is unchanged per the baseline goldens; Program differences are listed.
- [ ] Done summary gives polls and wait budget before/after and the non-adopted candidates with the Case that would need each.
- [ ] Model gate, lint-model, Go tests, lint-code-fast and live Case tests pass; README/SEMANTICS document hints.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
