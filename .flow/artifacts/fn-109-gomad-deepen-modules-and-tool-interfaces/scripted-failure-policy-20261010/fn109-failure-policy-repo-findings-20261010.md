# Failure-policy test attachment findings

The two existing failure-policy tests can use the existing scripted preparation helper at their final configuration points. This source inspection supports a two-assignment proposal. Root owns admission, implementation sequencing and acceptance. No Go, build, lint, vet, generator or target-process command ran during this inspection.

Inspected HEAD is `70e1cbb6293a4b8c6d230db446e105ec7eb2cba2`. The working tree already has Flow and milestone edits and untracked evidence; this report leaves them intact. The three relevant Runner files have no working-tree diff. The research route supplied by root is requested `gpt-6-astra/high`, tier `session`, unavailable `jev-unavailable(no_key)`. Actual model telemetry is unverified. No judge retry or route change occurred. Artifact prose follows `/home/agent/.codex/docs/flow-next/prose.md`.

## Exact source proposal

Add `configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)` immediately before each existing `exploreWith` call in `tools/gomad3/runner/runner_test.go`.

| Test and current lines | Final configuration point | Preserved behavior |
| --- | --- | --- |
| `TestRunFirstFailureCancelsActiveTargetsWithoutPublishingThem`, 544-565 | After `testConfig` at 547, before call at 548 | Seeds `1-10`, `PolicyFirst`, parallel 3; attempted 3, failures 1, cancelled 2, distinct failures 1, `StopFirstFailure`, one artifact and two `.partial` entries. |
| `TestRunBudgetCountsDistinctSignatures`, 567-585 | After `config.FailureBudget = 2` at 577, before call at 578 | Seeds `1-10`, `PolicyBudget`, parallel 1; outputs `same` until seed 4 emits `different`, all exit 1; attempted/failures 4, distinct failures 2 and `StopFailureBudget`. |

`testConfig` at `runner_test.go:2504` returns executor-only dependencies and retains execution timeout 1 second, overall timeout 10 seconds and termination grace 100 milliseconds. Attaching earlier or changing this shared helper would broaden the proposal.

## Preparation and bootstrap boundaries

- `runner/preparation_dependencies.go:17` calls real `preparation.Prepare` when `prepare` is absent. Its bootstrap method at line 24 calls real `profile.BootstrapFrame` when `bootstrap` is absent.
- `internal/preparation/preparation.go:54` selects the real deterministic-I/O validation operation. Even a custom preparer reaches `services.validate` at line 102 after its `Prepare` call. The retained frontier records unsupported-profile failures before these two executors run; this inspection did not reproduce those failures.
- `runner/preparation_fixture_test.go:17` requires explicit nonnil preparer/executor, checks the exact preparer and preparation root, rejects adapter replacements, calls that preparer, checks target kind/source/argv/adapters and invokes real `prepared.Verify`. It returns the existing synthetic marker from line 41. No new helper or production seam is needed.
- `runner/runner_local.go:250` passes the final config's preparer, target and environment into preparation. `runner/runner.go:681` obtains bootstrap bytes, copies them into `execution.Spec.IO.Config` at line 698 and calls the selected executor at line 712.
- `firstFailureExecutor.Run` at `runner_test.go:2484` reads the seed from the environment, waits until all three executors enter, lets seed 1 fail, then waits for cancellation in the other two. It returns cancelled/signal/killed results without parsing bootstrap bytes. Preserve that rendezvous and cancellation sequence exactly.
- `fakeExecutor.Run` at `runner_test.go:2332` records the request and selects its callback by environment seed. It does not parse bootstrap bytes or start a child. The budget test's callback only creates `processResult` values.

The proposed marker therefore stays inside scripted execution. Actual campaign orchestration, completion handling, failure signatures, artifact publication and partial-directory accounting remain exercised. Any newly reached decoder/process boundary or behavioral failure requires root disposition with the raw result retained.

## Preservation checks

The implementation must reconstruct the complete original `runner_test.go` byte-for-byte after removing exactly the two inserted statements within the named functions. Do not globally remove matching lines, because earlier admitted attachments already use the same statement. Require exactly two added statements, zero removed or changed original lines, and no other product-file change. Preserve all helpers, comments, assertions, payloads, error text, policies, timeouts and existing attachments.

Current SHA256 values provide the inspected-source seal.

| Path under `tools/gomad3/runner/` | SHA256 |
| --- | --- |
| `runner_test.go` | `d251cc9c5f32b95821fcede257df76ad2fe147c4e915128812a2e1ef275a5e96` |
| `preparation_fixture_test.go` | `c43c4fb18ad07b9b9bbb6efba9dc5a6194a99d86ae2405cd0dc4b6088943402e` |
| `preparation_dependencies.go` | `4f9e93b79fc75e984a34e6fa7591bf4ff077db5dd1e96330697483b9af434e56` |

Bind each eventual command to the admitted frozen candidate, actual tools/environment, timestamps, numeric exit and complete actual JSON terminals. Compare final ordinary test names, outcomes and diagnostics with the immediate frozen baseline. A test-count subtraction from the task75 report cannot establish preservation or acceptance.

## Proposed commands for root admission

These commands have not run. Use pinned ordinary Go1.27.1 and the root-admitted environment, external bounds and serial lane. The following commands run from `tools/gomad3`.

```sh
go test -tags test_dep -count=1 -json -run '^(TestRunFirstFailureCancelsActiveTargetsWithoutPublishingThem|TestRunBudgetCountsDistinctSignatures)$' ./runner
go test -tags test_dep -count=1 -json -run '^(TestPreparationDependencies(ForwardRealFixtureInputs|OperationErrorsRemainUnchanged|FailuresStopAtOriginalStages|KeepRealDefaultsAndBootstrapGuard)|TestInjectionCharacterization(IsolatedExploreRejectsEverySubstitution|IsolatedPreparationDependencies)|TestPortableProfilePublicGuardsRemainFirst)$' ./runner ./deterministicio
go test -tags test_dep -count=1 -json ./runner
go vet -tags test_dep ./runner
go vet -tags test_dep -vettool=<root-admitted-errortype> -style-check=false ./runner
<root-admitted-golangci-lint> run --config ../../.github/.golangci.yml --build-tags test_dep --timeout 10m --fix=false ./runner
go test -tags test_dep -count=1 -json -run '^Test(PackageArchitecture|PureModulesHaveNoHostEffects|PublicPackagesDoNotExportTypeAliases|PublicPackagesDoNotExportForwardingAliases|RunnerRequestsCompileInExternalModule|RunnerExecutionInjectionIsPrivate|ArchitectureEffectFixtures|ArchitecturePublicSignatureFixtures|ExactModuleEdges)$' .
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go vet -tags test_dep ./runner
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go vet -tags test_dep ./runner
```

The angle-bracket lint-tool entries denote identities root must bind before execution. The existing controls are in `preparation_dependencies_test.go:31,135,165,205`, `executor_injection_characterization_test.go:151,168`, and `deterministicio/profile_portable_test.go:162`. Preserve their unsupported-host guard expectations and qualified-host skip conditions. The focused test and ordinary Runner commands need before/after evidence; controls need current-candidate evidence or exact consumed-domain reconciliation under root's admission.

From repository root, require `gofmt -l tools/gomad3/runner/runner_test.go` and `git diff --check`, plus the admitted forms of `make lint-code-fast` and `make lint-code-gomad3` with `SHELL=/bin/sh`, `ALL_TEST_TAGS=test_dep`, `GOLANGCI_LINT_FIX=false` and pinned lint tools. The task75 retained lint comparison uses `GOLANGCI_LINT_BASE_REV=951c5516e9e7b3066e7e069adda9565cfd68844c`; root must distinguish that original lint baseline from the new source-start revision. Retain full original diagnostic blocks and actual integrated errortype reachability. `make -C tools/gomad3 validate SHELL=/bin/sh` supplies generated validation; reuse requires complete consumed-domain equality, including the previously retained 132 materialized inputs and 113 top-level tests. No generated-input change follows from these two assignments.

The retained task75 command receipts live under `.worktrees/fn-109-75-six-attachments-candidate/.flow/tmp/fn10975-evidence/`, especially `after-controls-binding.json`, `host-vet-binding.json`, `errortype-binding.json`, `runner-lint-binding.json`, `fast-lint-binding.json`, `after-aggregate-lint-binding.json`, `architecture-binding.json`, both `vet-*-binding.json` files and `validate-binding.json`. Those receipts establish prior command shapes, not a pass for this proposal.

## Source-start risks and documentation

`MILESTONES.md` gives fn155 priority and root control of shared Go/build/lint/vet/generator work. The retained frontier explicitly proposes a new bounded owner and admission after that lane releases. Existing fn109.65-.75 attachments do not authorize these sites. Root must bind the actual source start and predecessor reconciliation before implementation or execution. Existing dirty Flow/milestone edits must survive.

The task75 frontier retains ordinary RED130, collateral RED22, Runner RED6, aggregate RED50 and unreached integrated errortype. Task74's publication deadline and the divergence crash-helper exit retain unresolved causes. Passing these two tests cannot close those independent requirements or the acceptance consumers fn109.63/.21 and fn112.10. Static darwin/arm64 and linux/amd64 checks do not supply native execution; native fn128/fn149 deferrals remain unchanged.

No README, SPEC, ARCHITECTURE, CLI, interface inventory or product documentation update is needed. The proposal restores existing test coverage through an existing private test helper without changing a product contract. Root should record the selected scope, preservation proof, actual results and independent review in the bounded owner/evidence after admission. This scout writes only this findings file.
