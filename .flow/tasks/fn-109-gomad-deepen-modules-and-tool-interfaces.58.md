---
satisfies: [R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.58 Check inspection and invalid compiler-fixture cleanup

## Description
Own exactly two retained unchecked production cleanup results, deferred opened.Close in runner/inspect.go and the immediate invalid-package workspace cleanup in internal/gomadtool/conformance/driver.go. Healthy cleanup preserves original result tuples, primary errors/bytes, call order and ownership. Only genuine cleanup failures acquire direct or primary-first joined errors.

**Touches:** [tools/gomad3/runner/inspect.go, tools/gomad3/runner/inspect_cleanup_test.go, tools/gomad3/internal/gomadtool/conformance/driver.go, tools/gomad3/internal/gomadtool/conformance/driver_cleanup_test.go]

### Exact correction boundary

Inspect may use named result/error and an inline deferred closure at the existing registration. Evaluate the same receiver there, keep artifact/project-manifest/target-sharing/optional-choice order and cleanup after all existing work. Nil Close preserves the complete original result/error. Sole cleanup error returns direct with zero Inspection; primary plus genuine cleanup failure returns primary-first joined error with zero Inspection. Downstream command statuses and diagnostic-write behavior remain unchanged.

The conformance invalid-package branch captures cleanup before constructing the exact original error negative compiler fixture package is invalid: %s. Nil cleanup retains that original error object and nil/nil/error tuple. Genuine cleanup error joins primary-first. One cleanup attempt at the current point, with no ownership transfer, broader defer, retry or reordering. conformancePassed stays false with no cases executed and command status1 unchanged.

Add controls in the declared new files without editing runner/inspect_test.go or any old tests. Inspection controls cover canonical no-choice success, Choices:true returning zero plus exact artifact has no choice trace after opening, missing/conflicting earlier-error precedence and target sharing. Conformance uses a test-owned root/compiler manifest/descriptor/interception report with a structurally accepted but invalid nonempty package prefix to reach rejection. Require nil fixture/cleanup returns, exact primary bytes and no workspace. Valid-prefix controls retain fixture order/argv and caller-owned cleanup; runWith requests only GOROOT and performs no compiler execution.

Pinned qualified Unix os.Root.Close returns nil, so genuine artifact Close failure is unavailable through these hosts. Immediate workspace RemoveAll has no deterministic existing failure hook. Retain analyzer RED/source preservation proof and honest genuine cleanup/multiple-error runtime gaps. Ordinary fixture preparation does not prove native compiler execution. Do not introduce a fault seam, descriptor theft or races.


### Execution and retained acceptance

Root admits this exact correction to unblock fn-112.10's required integrated source lint. Task21 consumes its evidence. Depend only on Done task52; progress-only correction owners are verification consumers rather than prerequisite-completion cycles. Root owns selection, integration, review, lifecycle and target commits. A fresh worker owns implementation/tests/evidence in an isolated worktree after planning records are committed. No shared checkout writer or concurrent Go/build/lint/generator gate is admitted. Root schedules the shared gate lane; keep each checked candidate frozen. Handovers under the task-unique artifact directory are lifecycle output, excluded from product Touches.

Read AGENTS.md, tools/gomad3/README.md, MILESTONES.md, the parent/task and retained task55 reconciled proof/review plus task56's reviewed handover when available. Re-anchor at the committed worktree base; keep actual original-base RED and exact source hashes as defect evidence. Use apply_patch and established libraries, pinned stock Go1.27.1, all Go tests -tags test_dep -count=1, the established cache/file-proxy environment and private temporary directories. Run focused baseline/final controls, ordinary affected packages, affected vet/standalone errortype, relevant architecture/public/purity/private-injection checks, both supported static source sets, fresh check-only validation, format, actual unfiltered affected configured lint, actual FIX=false make lint-code-fast against the worktree admission base, and original-base make --trace lint-code-gomad3 against951c5516e9e7b3066e7e069adda9565cfd68844c. Reuse valid exact-source receipts. Retain raw source/tool/command/exit/elapsed bindings and exact removed/introduced/residual findings; fast-filtered green does not replace red unfiltered lint or integrated errortype.

Parallel scope independence grants no acceptance waiver. Root independently verifies and reviews the integrated target and runs focused integrated controls before completion. A red required source gate licenses only a reviewed source-progress commit, keeps formal acceptance in_progress, and supplies no formal SHIP or full-spec acceptance. Preserve original first-baseline/preservation/R18/R19 requirements and explicit native fn149/fn128 deferrals. No native-host spoofing, new fault hook/seam, framework/library, policy/pin/API change, weakened assertion, unrelated comment edit, PR, push or CI authority. Disclose every unreachable genuine cleanup-failure branch precisely.

### Combined source verification (2026-10-09)

Tasks57-59 are separately reviewed and integrated through a7657365281dccbdcac27ffd9b2c34e54a36d3ea. Root frozen combined fingerprint4287cc794ab0a4e55f2262ebd600052bf6b7a641655c7ba7ac2b6383477c5575 binds11 terminal receipts. Actual original-base lint80 to60 removes18 errcheck and2 exhaustive findings, zero introduced, with every residual full block preserved after line mapping. Required lint remains RED60, affected configured lint RED24, integrated errortype unreached. Focused observations12pass/88fail/0skip retain inherited failures. Static/boundary/private/validate/format/vet/standalone-errortype pass. Full ordinary Runner was attempted once but explicitly diagnostic-aborted at its unchanged unbounded executor.started receive;274pass/226fail/6skip are partial observations, not complete coverage. This source-owned hang and later unexecuted tests remain acceptance gaps. See [combined source-progress](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/combined-57-59/source-progress.md) and hang-diagnostic.md. Fresh combined source-progress review accepted the bounded integration with no introduced findings; see the combined independent-review artifact. Tasks remain in_progress; no formal SHIP/Done or native qualification. Retain all first-baseline/fixed-identity/R18/R19/source obligations and deferred fn149/fn128.
## Acceptance
- [ ] Exactly two production cleanup findings are removed with zero introduced findings. Successful cleanup preserves original tuple/error identity/bytes, evaluation/order/lifetime and cleanup ownership; only genuine cleanup errors change results.
- [ ] Additive baseline/final ordinary inspection and compiler-fixture preparation controls pin literal output/errors, earlier precedence, no workspace debris and successful caller-owned cleanup without rewriting existing tests or claiming native compiler execution.
- [ ] Frozen focused/ordinary/static/validation/format/vet/errortype/actual configured-fast-original lint receipts bind the actual candidate and preserve every required red source gate and cleanup reachability gap.
- [ ] Root verifies the integrated target, obtains fresh independent review and commits this task separately; Done requires all still-owned source acceptance.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
