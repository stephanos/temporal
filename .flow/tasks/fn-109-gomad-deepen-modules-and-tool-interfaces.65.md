---
satisfies: [R5, R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.65 Restore explicit scripted Runner preparation and bootstrap coverage

## Description
Restore the first six explicit scripted seed/World campaign calls through two private per-invocation dependency functions, leaving production defaults and real orchestration behavior unchanged. Follow [the bounded admission](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-65/admission.md); it names the exact six calls and retained assertions.

**Touches:** [tools/gomad3/runner/preparation_dependencies.go, tools/gomad3/runner/runner.go, tools/gomad3/runner/campaign_options.go, tools/gomad3/runner/runner_local.go, tools/gomad3/runner/preparation_fixture_test.go, tools/gomad3/runner/preparation_dependencies_test.go, tools/gomad3/runner/runner_test.go, tools/gomad3/runner/executor_injection_characterization_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-65/**]

Only preparation and bootstrap operations are injectable; nil calls the original operation lazily. Existing executor and full private dependency propagation survive resumed request reconstruction. Each injected operation is rejected for an isolated campaign even when executor is nil, with the existing validation order/error. No public request or wire-format changes, global hooks, executor-type inference, blanket fixture migration, storage work or changes to existing assertions/helpers. Native qualification and ordinary portable assertions remain distinct. This correction supports open fn-109.63/fn-112.10 acceptance; it does not admit dependent simulation/storage work.

Quick commands use the pinned documented environment and `test_dep`: unchanged target-mutation test for RED; all six tests and additive dependency/default/isolated controls for GREEN; existing preparation-error/local-phase controls and public profile/package-architecture boundaries; affected vet/errortype; generated validation; formatting; configured unfiltered affected-package lint; repository `make lint-code-fast`. One frozen ordinary Runner run and original-base lint comparison follow at the root's batch gate. Required red gates remain accurately reported with acceptance open, not suppressed or replaced by scoped checks.

## Acceptance
- [ ] The unchanged target-mutation test first fails at the documented preparation assertion on the actual candidate, then reaches the real file mutation and expected prepared_target_integrity error. All six explicitly admitted tests pass with every existing assertion unchanged.
- [ ] The two private operations preserve lazy production defaults, original arguments/errors and resource order. Per-call fixture construction retains actual copied target data and controller/journal/World/filesystem/artifact behavior; forwarded bootstrap arguments and marker bytes are checked with concurrency-safe observations.
- [ ] Public/default and executor-only calls retain real preparation refusal; preparation-only substitution still meets the real bootstrap guard. Isolated execution rejects prepare-only, bootstrap-only and combined substitutions with nil executor before invoking them or starting a coordinator, preserving existing error and validation ordering.
- [ ] No existing helper/assertion, public API, qualified host admission, global hook, unrelated test, runtime, storage, adapter, toolchain, replay/plan/guidance/minimize owner or CLI contract is changed. Existing preparation failures/cancellation and local phase controls retain their behavior; resumed reconstruction retains the full private dependency carrier without claiming portable resume qualification.
- [ ] Focused source tests, package architecture, public profile guards, generated validation, affected vet/errortype and formatting pass on the frozen candidate. Configured affected lint, repository fast lint, the frozen ordinary Runner observation and original-base lint comparison are retained honestly; unproved/red source acceptance remains open and no new unadmitted original outcome or lint finding is introduced.
- [ ] Fresh independent review accepts the current integrated change with its actual coverage limits. Formal completion follows all retained task-owned gates; native obligations remain with fn-128/fn-149 and no native/CI/PR/push authority is inferred.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
