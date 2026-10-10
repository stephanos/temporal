---
satisfies: [R5, R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.76 Restore scripted first-failure and distinct-budget coverage

## Description
Restore the two existing failure-policy tests' original cancellation, artifact and budget assertions using the existing private `scriptedPreparationDependencies`. Retained ordinary evidence stops at deterministic-I/O preparation refusal before either scripted executor reaches those assertions.

**Size:** S
**Files:** tools/gomad3/runner/runner_test.go and task-specific evidence owner .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/scripted-failure-policy-20261010/**
**Touches:** [tools/gomad3/runner/runner_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/scripted-failure-policy-20261010/**]

The product change is two added statements with eight acceptance obligations. Attach current R5/R18/R19. Formal `depends_on` is `[]`.

### Scope

In `tools/gomad3/runner/runner_test.go`, add exactly `configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)` in `TestRunFirstFailureCancelsActiveTargetsWithoutPublishingThem` after final `testConfig`, and in `TestRunBudgetCountsDistinctSignatures` after `config.FailureBudget = 2`, immediately before each existing `exploreWith` call. Preserve every original byte after selective removal of those two insertions within those exact function boundaries.

Preserve assertions, comments, payloads, three-executor rendezvous, duplicate/distinct ordering, timeouts, caches, errors, defaults, helpers and all earlier admitted attachments. Keep real target verification, orchestration, journal and artifact behavior. Existing default, preparation-negative, isolated-injection and public-profile controls stay unchanged. These executors ignore synthetic bootstrap; reaching a real decoder/process requires root disposition.

### Source admission and consumers

Root must first reconcile task75's independently accepted SOURCE checkpoint, explicitly admit this new owner, record the released shared Go/compiler/build/lint/vet/generator lane and freeze the actual immediate baseline/candidate. Lane release supplies no execution authority. Formal task75 Done is unnecessary. Fn155 retains priority and its unavailable linux/arm64 first-native gate still holds `.2/.8` despite committed correction `682dd4fa01`. No supported-native qualification, CI, PR or push action is admitted.

Root adds this owner alongside task75 to `.63` prerequisites; `.21` consumes it through `.63`. Fn112.10 consumes a dated prose trace while retaining its existing predecessors. Preserve the task75 trace and other parent ownership.

### Bounded investigation targets

1. The two named function bodies in `runner_test.go`.
2. Existing `runner/preparation_fixture_test.go` helper.
3. Existing `runner/preparation_dependencies.go` defaults.
4. Existing `runner/preparation_dependencies_test.go` controls.
5. Existing `runner/executor_injection_characterization_test.go` isolation controls.
6. Existing `deterministicio/profile_portable_test.go` public-profile controls.
7. Task75 reconciliation/receipts and the committed 87-parent frontier, limited to source admission and exact consumed-domain reuse.

Use the [repo findings](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/scripted-failure-policy-20261010/fn109-failure-policy-repo-findings-20261010.md) for command/tool shapes and retained receipt paths. Wider failure groups, production behavior, helpers, recorded formats and native owners remain separately owned. Preserve original research facts/bytes; retain fresh source-run proofs separately under this task's evidence owner. Retain newly reached failures with their original diagnostics. Historical RED50 and other unproved source requirements remain open; focused source progress grants no formal SHIP/Done.

## Acceptance
- [ ] Root records task75's independently accepted SOURCE checkpoint/reconciliation, explicit new-owner admission, fn155 priority disposition and shared Go/compiler/build/lint/vet/generator lane release. Bind the immediate before/after source identities, actual tool/environment inputs, timestamps, numeric exits and external bounds. New-owner formal dependencies stay empty; task75 Done is unnecessary.
- [ ] Exactly two statements attach the final preparer/executor at the named boundaries. A function-aware verifier removes only the new statement from each exact selected function, proves each removal occurred once, and compares the complete reconstructed `runner_test.go` byte-for-byte with the frozen immediate baseline. Require zero original deletions/changes and no other product-file change; preserve earlier matching assignments, assertions, comments, helpers, payloads, errors, defaults, caches and timeouts.
- [ ] Retain the actual pre-edit preparation refusals and post-edit focused raw JSON from the command below. Preserve cancellation rendezvous, 3 attempts, 1 failure, 2 cancellations, 1 distinct failure, first-failure stop, 1 artifact and 2 cancelled partials. Preserve budget duplicate ordering through seed 4, 4 attempts/failures, 2 distinct failures and budget stop. No assertion or fixture change may manufacture those results.

Run from `tools/gomad3` with admitted pinned ordinary Go1.27.1. The third command is the exact root-admitted control selection; preserve it without adding tests.

```sh
go test -tags test_dep -count=1 -json -run '^(TestRunFirstFailureCancelsActiveTargetsWithoutPublishingThem|TestRunBudgetCountsDistinctSignatures)$' ./runner
go test -tags test_dep -count=1 -json ./runner
go test -tags test_dep -count=1 -json -run '^(TestPreparationDependencies(ForwardRealFixtureInputs|OperationErrorsRemainUnchanged|FailuresStopAtOriginalStages|KeepRealDefaultsAndBootstrapGuard)|TestInjectionCharacterization(IsolatedExploreRejectsEverySubstitution|IsolatedPreparationDependencies)|TestPortableProfilePublicGuardsRemainFirst)$' ./runner ./deterministicio
```

- [ ] Fresh complete ordinary before/after logs and raw JSON support a union comparison of every actual top-level/subtest name, terminal outcome and diagnostic, including run-only, newly reachable and package terminals. Enumerate selected changes; require all unselected normalized outcomes/diagnostics unchanged, retaining raw text and justified normalization rules. Missing/truncated terminals, timeouts, toolchain refusal or unexpected downstream behavior remain open with root disposition. Historical counts and subtraction supply no acceptance.
- [ ] The unchanged controls above prove preparation failure stages/error identity, real defaults/bootstrap guard, isolated injection and public-profile refusal behavior. Retain current-candidate receipts or reconcile each exact complete consumed domain against an accepted receipt. Preserve qualified-host skips and unsupported-host expectations. Synthetic bootstrap never reaches a real decoder/process.
- [ ] Retain formatting/diff checks, affected host vet and errortype, unfiltered Runner lint, one actual canonical `make lint-code-fast` invocation with `GOLANGCI_LINT_BASE_REV=951c5516e9e7b3066e7e069adda9565cfd68844c`, and required nested aggregate baseline/final `make lint-code-gomad3` receipts. Source BASE identifies the frozen immediate source start and is separate from that original lint comparison base, as task75's lint-base clarification records. Bind admitted tool versions/configurations; use `SHELL=/bin/sh`, `ALL_TEST_TAGS=test_dep` and `GOLANGCI_LINT_FIX=false`. Preserve full diagnostic blocks with no suppression, exclusion, auto-fix or rule/config change. An unreached errortype gate remains unproved.
- [ ] Retain architecture/public-surface checks with `-tags test_dep -count=1 -json`, both supported source sets (`CGO_ENABLED=0 GOOS=darwin GOARCH=arm64` and `CGO_ENABLED=0 GOOS=linux GOARCH=amd64` vet), and `make -C tools/gomad3 validate SHELL=/bin/sh`, using the exact command shapes in the repo findings. Receipt reuse proves complete consumed-input and tool/environment equality for each gate, including all 132 materialized inputs and 113 top-level validation tests where consumed; filenames or unchanged selected-file hashes alone are insufficient. Static/source evidence grants no native execution claim.
- [ ] A fresh independent reviewer assesses the frozen integrated source, whole-file proof, complete ordinary union and all current/reconciled source receipts. Retain reviewer identity/verdict and root reconciliation; workers cannot self-certify. Ordinary RED130, collateral RED22, Runner RED6 and aggregate RED50 are historical candidate observations, not prescribed new results; their unresolved requirements, other failures and unproved errortype remain open in their owners/consumers. Preserve original research facts/bytes separately from new source-run proofs, task74's unresolved deadline/killed137 and crash-helper evidence. This correction grants no formal SHIP/Done until its owned source requirements pass. Fn128/fn149 remain deferred and unverified, fn155 keeps its first-native gate, and fn112.10 receives only a dated prose trace.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
