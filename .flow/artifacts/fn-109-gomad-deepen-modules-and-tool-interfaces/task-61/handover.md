# Task61 source handover

The competing-build fixture now releases its first builder after the second observes real lock contention. Its test-local context delegates semantics and notifies once; five-second operation/cleanup bounds cancel, release and drain builders. All original two-result, equal-key, one-build and Waited assertions remain unchanged.

Task fn-109-gomad-deepen-modules-and-tool-interfaces.61 remains in_progress. Root owns review, integration, commits and completion; no local commit or review verdict was issued.
Tier: IMPLEMENTER gpt-6.1-sol/high; judge unavailable(no_key), explicit project routing retained; telemetry unobserved.
stage: impl-review - skipped(policy: host-deferred - conductor owns the gate)

Base and HEAD remain 7b75ae312a6641c6b00afbb63f44ae9a9dd0c2b2. The only product change is tools/gomad3/toolchain/build_test.go, SHA256 ba64e85358a6993e669a5a1925ab0cee8ece625fb43df987b9bdeca437b2cd15. Source fingerprints change from5c5867fb702a0d3dbeeff6c374e9a061df02e7d4aec78ca7fa312084e1f12471 to0965d047c4803221aa245dc3e9b34d612c5f213c5b816e57961cde02cf0ca9db; only that fixture differs among1,064 bound inputs.

[Source preservation](source-preservation.json) and [verify_source.py](verify_source.py) confirm all other test bytes, original assertions/fixture inputs and production build/lock/hostfs sources match BASE. The observer follows the existing simulationProgressContext pattern. In this fixture, only acquireBuildLock's real ErrContended retry path queries the second context's Done. fakeRunner's shared sync.Once remains unchanged; builds=1 alone supplies no universal duplicate-invocation proof.

[Supplementary expectations](expectations.md), [verification.json](verification.json), [verify_evidence.py](verify_evidence.py) and [evidence.json](evidence.json) retain commands, source/tool/raw/auxiliary hashes, exits and elapsed times. All16 receipts are terminal, stable and not timed out; both verifiers actually return0. Historical run_gate.py remains at its original b9b9f127b6c94cfeec460d93a41601c765eb4aae8c75316a9afb98fc610886c6 hash. Distinct run_final_gate.py additionally binds overlay inputs.

The governing task requirements committed at7b75 and original BASE outcome assertions establish independent pre-edit expectations. Baseline receipts do not hash expectations.md, so independent retained evidence does not establish that document's complete pre-edit contents or supplementary control-design chronology. That provenance requirement remains open; verified source ordering, preservation and actual mutant outcomes do not close it.

| Gate | BASE exit / seconds | Final exit / seconds |
| --- | --- | --- |
| TestBuild* | 0 / 10.887 | 0 / 3.967 |
| Configured toolchain lint | 1 / 5.314 | 0 / 3.186 |
| Toolchain vet | 0 / 1.731 | 0 / 0.418 |
| Check-only gofmt | 0 / 0.009 | 0 / 0.005 |
| Toolchain errortype | not run | 0 / 1.674 |
| Canonical fast, base7b75, FIX=false | not run | 0 / 8.110 |
| Original-base nested lint, base951c, FIX=false | 2 / 17.703 | 2 / 2.542 |

Matched TestBuild* runs each pass11 top-level tests and17 subtests with zero fail/skip observations. Context controls pass0/0.929s, covering concurrent/repeated Done notification, underlying channel identity, background/deadline/value forwarding and cancellation identity.

[Actual v2 controls](control-inputs-v2.json) reject muted notification with the bounded contention timeout (1/6.734s), waited=false with the original Waited assertion (1/1.396s), and bypassed waiting with a forced sequential second builder through that assertion (1/1.344s). Every mutant logs that all launched builders drained; none reaches process/cleanup timeout. The sequential control fixes its schedule rather than relying on a race. [prepare_controls.py](prepare_controls.py) retains exact transformations and scratch inputs under the assigned tmp-61 directory. Superseded unexecuted draft descriptors were removed; executed raw receipts remain intact.

Baseline is behaviorally green and lint-red. Actual original-base lint drops60 to59 solely by removing build_test.go:97's forbidden sleep. Zero findings are introduced; every residual path/line/column/message/statement/caret block is byte-identical. No residual lies on the changed file, so line mapping is identity. Integrated errortype remains unreached; the affected standalone check passes. Fast lint0 remains scoped/diff-filtered.

Root still owes integrated checks/checkpoint and lifecycle decisions. Fresh bounded source-progress review found no introduced source issue and identified the supplementary chronology gap above. RED59 retains seven errcheck, eight forbidigo and44 staticcheck findings. Formal SHIP, Done, supplementary pre-edit provenance, parent first-baseline/fixed-identity/R18/R19 and full source acceptance remain open wherever unproved. Native fn149/fn128 remain deferred/unverified; these ordinary fixtures supply no native pass, soak bound, PR, push, merge or CI authority.

Defect route:
- prior fixes: root's admitted research and scoped Git history identify the original fixture; external PR/tracker discovery remains unchecked under this bounded dispatch.
- diagnosis: actual BASE analyzer confirms the sleep RED; actual final mutants reject missing notification and missing waiting.
- introduced by: bisect skipped because no known-good revision was supplied.
- base: TestBuild*0, affected lint1/one sleep and integrated lint2/RED60; head: frozen source above, TestBuild*0, affected lint0 and integrated lint2/RED59.
- live: no live application surface; production sources remain unchanged.
