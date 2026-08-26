---
satisfies: [R2]
---
# fn-27-hermetic-ci-execution-and-qualification.2 Bind and execute the disposable CI runtime profile

## Description
Implement R2 by composing a distinct CI RuntimeConfiguration with the existing runtime engine and loopback lifecycle.

**Size:** M
**Files:** `model/Temporal/System/Execution/CIProfile.lean`, `model/Temporal/System/Execution/CIProfileTests.lean`, `model/Temporal/Feature/Nexus/Execution.lean`, `model/Temporal/Feature/Nexus/ExecutionTests.lean`, `tools/umpire/runtime/**`, `tools/umpire/temporal/local/**`, `tools/umpire/temporal/nexus/testdata/caller-closure-ci-input-set/**`
**Touches:** [model/Temporal/System/Execution/CIProfile.lean, model/Temporal/System/Execution/CIProfileTests.lean, model/Temporal/Feature/Nexus/Execution.lean, model/Temporal/Feature/Nexus/ExecutionTests.lean, tools/umpire/runtime/**, tools/umpire/temporal/local/**, tools/umpire/temporal/nexus/testdata/caller-closure-ci-input-set/**]

### Approach

- Add the checked `temporal.runtime-profile.ci-hermetic` instance with the existing five budgets, four evidence sources, capability closure, seed zero, and attempt one.
- Produce an exact two-member CI fixture that reuses the caller-closure ExperimentSpec artifact bytes and changes only the checked RuntimeConfiguration/profile binding.
- Generalize runtime preflight behind a closed profile registry or exact authority constructor; do not expose a profile, endpoint, namespace, credential, timeout, or executable selector.
- Reuse the same context-aware LiteServer and participant lifecycle; enforce loopback-only execution authority, no filesystem/publication responsibility, and exactly-once isolation/cleanup.
- Re-run the independent phase/status/capacity oracle for CI and prove local request/configuration behavior unchanged.

### Investigation targets

**Required** (read before coding):
- `.flow/specs/fn-19-bounded-local-temporal-execution-and.md` — runtime API, phase table, limits, and authority boundary
- `.flow/tasks/fn-19-bounded-local-temporal-execution-and.1.md` — portable profile composition
- `.flow/tasks/fn-19-bounded-local-temporal-execution-and.3.md` — independent engine oracle
- `.flow/tasks/fn-19-bounded-local-temporal-execution-and.4.md` — LiteServer lifecycle adapter
- `.flow/tasks/fn-19-bounded-local-temporal-execution-and.5.md` — exact caller-closure configuration/fixture
- `temporaltest/server.go` — lifecycle seam to reuse

### Acceptance

- [ ] The CI fixture contains the byte-identical ExperimentSpec plus exactly one distinct admitted CI RuntimeConfiguration.
- [ ] Invalid profile/program/budget/capability/authority/input variants fail before server startup.
- [ ] Every phase outcome preserves fn-19 precedence, evidence bounds, fresh isolation/cleanup contexts, and zero surviving handles.
- [ ] Local runtime fixtures, APIs, and command behavior remain unchanged.

## Acceptance
- [ ] R2 exact configuration, runtime admission, lifecycle, and evidence closure are complete.
- [ ] Fake-oracle and bounded live loopback tests pass with no filesystem/publication or external authority.
- [ ] Existing lifecycle comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
