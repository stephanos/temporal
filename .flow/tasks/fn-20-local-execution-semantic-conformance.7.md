---
satisfies: [R3, R4, R7, R8]
---

# fn-20-local-execution-semantic-conformance.7 Run the bounded live proof and synchronize current-model documentation

## Description
Complete R3/R4/R7/R8 with one fn-19-to-fn-20 caller-closure proof, public usage docs, architecture handoff, and honest roadmap status.

**Size:** M
**Files:** `tools/umpire/runevaluation/live_test.go`, `tools/umpire/runevaluation/README.md`, `model/README.md`, `model/ARCHITECTURE.md`, `.plans/UMPIRE4_COMPONENTS.md`
**Touches:** [tools/umpire/runevaluation/live_test.go, tools/umpire/runevaluation/README.md, model/README.md, model/ARCHITECTURE.md, .plans/UMPIRE4_COMPONENTS.md]

### Approach
- Produce one real closed four-member caller-closure set through fn-19, check it through the actual sibling pair, publish/reopen the six-member set, and independently assert cleanup/source closure plus the API/history-backed Property result and diagnostic obligations.
- Run the full corruption/ambiguity suite before the green live control and require exact deterministic rerun destination.
- Document the two offline command inputs, six-member output, checker-pair build, statuses, Limits, three independent result dimensions, evidence dispositions, fail-closed cases, and copy-paste root command.
- Update C7, Milestone B, and pilot step 5 only after the live and mutation evidence genuinely passes; retain remote/Observation Evaluation and replay/promotion gaps.
- Run all focused suites plus the stable regression gate; add no CI workflow or model-local Make target.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-19-bounded-local-temporal-execution-and.7.md` — live four-member producer/harness
- `model/README.md:93-98,149-158` — current model/runtime handoff text
- `model/ARCHITECTURE.md:90-103,147-159` — Feature/System/Tool ownership and lifecycle
- `.plans/UMPIRE4_COMPONENTS.md:330-363,613-640` — C7 and Milestone B status
- `.plans/UMPIRE4_COMPONENTS.md:700-715` — pilot sequencing/status

## Acceptance
- [ ] One bounded real local run yields a reopened six-member set whose Result proves the intended API/history Property and explicitly binds cleanup/source closure.
- [ ] Corruption/ambiguity controls run first and every repeated valid run/check produces the expected stable semantic and publication identities.
- [ ] Documentation gives copy-paste usage and explains why operational success is not semantic satisfaction.
- [ ] Roadmap claims only the one local scenario and keeps replay/promotion, formal checking, and non-local/release Observation Evaluation open.
- [ ] Focused Lean/Go, direct/root command, and `make umpire-check-regression` checks pass without prohibited dependencies or new CI.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
