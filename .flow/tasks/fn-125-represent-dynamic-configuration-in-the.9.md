---
satisfies: [R10]
---
# fn-125-represent-dynamic-configuration-in-the.9 Declare bound assumptions on server durations and check them at preparation

## Description
Implements R10 (spec Part D). A wait bound or kit deadline that rests on a server duration declares an `atMost` / `atLeast` assumption beside fn-118's bound; lowering adds it to the Case's required settings with its relation and origin; preparation checks the relation against the Profile; the functional harness runs the key at its declared value. The Model holds no duration.

**Cross-spec entry gate:** fn-118.4 done (bounds derived in the lowering). Coordinate with fn-124.3, which moves timeouts and activity attempts into the realization: whichever lands second rebases, and the assumption sits on the declaration fn-124.3 leaves. Not concurrent with fn-124.8. Depends on task 6.

**Size:** M
**Files:** `model/temporal/realize/**` (`CauseKind.timer.boundedBy(...).assuming(timerMaxTimeShift atMost 1.second)`, the Nexus retry poll on `retryPolicy.initialInterval`); `ir.proto` (`CauseBound.assumes`) and generated Go; `model/lifter/**`; `tools/umpire/lower/**`; `common/testing/testpilot/internal/execution/prepare.go` (relations); `common/testing/testpilot/temporal/profile.go`; `tests/testpilot_generated_test.go` (set a relation's key to its declared value); `model/cases/**`.
**Touches:** [model/temporal/realize/**, model/lifter/**, model/ir/**, model/cases/**, proto/internal/temporal/server/api/umpire/v1/**, api/umpire/v1/**, tools/umpire/lower/**, tools/umpire/model/**, common/testing/testpilot/**, tests/testpilot_generated_test.go]

### Approach
- Declare `history.timerProcessorMaxTimeShift atMost 1s` under the timer slack and the Nexus `retryPolicy.initialInterval` relation under the attempt poll. Walk `.plans/DYNAMIC_CONFIG.md` section 4: declare each relation a Case rests on, or list it with the Case that would need it.
- Preparation: `EQUAL` stays as today; `AT_MOST` / `AT_LEAST` parse the Profile's value with the key's codec and compare. Owner Q3 (decided 2026-10-04): fail closed. A Profile that does not state a required key's value, local or remote/canary, is `PreparationUnavailable`; no registered default is trusted. An unmet or unstated relation names the key, the relation, the value and the origin's position.
- The functional harness sets an inequality's key to its declared value, so one declared value configures the cluster and stands behind the bound.
- The Profile's scale factor multiplies wait bounds only; a test shows it changes no required value, request field or server timer.
- Record the Case-byte delta (assumption entries in required settings).

### Investigation targets
**Required:**
- `.plans/DYNAMIC_CONFIG.md` section 4; `model/temporal/realize/Kit.scala:73-79,163`
- fn-118.4's bound derivation in `tools/umpire/lower`
- `common/testing/testpilot/internal/execution/prepare.go:90-120`; `contract/profile.go` (`BoundScale`)
**Optional:**
- `.flow/tasks/fn-124-shrink-and-simplify-the-umpire-go.3.md`

### Quick commands
```bash
make protoc && make umpire-gen-model && git diff --stat model/ir model/cases
make umpire-check-cases
go test -count=1 -tags test_dep -p 2 ./tools/umpire/... ./common/testing/testpilot/...
```

### Execution constraints
- No Model duration; timers stay actions. Case bytes change only by the recorded assumption entries.

## Acceptance
- [ ] The timer slack and the Nexus attempt poll declare their server-duration assumptions; each relation in `DYNAMIC_CONFIG.md` section 4 is declared or listed with the Case that would need it.
- [ ] Preparation checks `atMost`/`atLeast` against the Profile; an unmet or unstated relation (any Profile, fail closed per Q3) is `PreparationUnavailable` naming key, relation, value and origin position.
- [ ] The functional harness runs each inequality's key at its declared value; the scale factor changes no required value, request field or server timer.
- [ ] The Case-byte delta is recorded; tooling and Testpilot tests, `make umpire-check-cases` and `make lint-code-fast` pass.


## Done summary
Blocked:
Blocked: deferred by the owner on 2026-10-05 together with the whole of fn-125 (dynamic configuration in the Models). Task 1 (the HSM/CHASM switch fixes) is done and merged; revive the spec to continue.
## Evidence
- Commits:
- Tests:
- PRs:
