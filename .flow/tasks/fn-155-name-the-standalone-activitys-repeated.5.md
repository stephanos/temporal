---
satisfies: [R6]
---
# fn-155-name-the-standalone-activitys-repeated.5 Single-definition Realization evidence builders

## Description
Implements the local items of spec section F in the Realization file. Each of these gets a single definition: the run-event attempt record, the Describe read, status and run-state conditions, the worker activation, and the run-scoped Describe base. The per-policy `deadlines` generator is deferred (spec Boundaries).

**Size:** M
**Files:** `model/temporal/features/activity/standalone/system/Realization.scala`
**Touches:** [model/temporal/features/activity/standalone/system/Realization.scala]

### Approach
- **Attempt record.** `attemptRecord(script, number, carrier, extraGuard*)` replaces `heartbeatDelivered`, `heartbeatRetryDelivered` and `resetDelivered`. Its guard must stay byte-identical in lowered Cases.
- **Describe read.** Rename `externalRead` to say it reads Describe. Reuse it for `heartbeatReceipt`, `heartbeatCompleted` and `heartbeatExpired`. Move `activityRunFieldForInfo` above its first use.
- **Conditions and activation.** `statusIs`/`runStateIs` conditions, an `activation` val, and a run-scoped Describe base carrying `runId := executionRun`.
- **Lift check.** Each helper must lift. The lifter follows helper defs with bound parameters (`model/irgen/Realizations.scala:86-144`). Check each one with a scratch lift.

### Investigation targets
**Required:**
- Pre-split `system/Realization.scala:93-243`, `:658-751`, `:919-961`
- `model/irgen/Realizations.scala:86-144`, `:812-848`

### Key context
- **Trigger identity.** Joint conflict scopes need realized trigger identity (memory: joint-conflict-scopes-require-realized). Keep evidence ids and `confirms` unchanged.

### Acceptance
- [ ] `make umpire-check-cases` shows lowered Cases unchanged apart from mapped identities
- [ ] Each listed builder has one definition
- [ ] Focused realization and lowering tests pass

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
