---
satisfies: [R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.61 Observe real competing-build lock contention instead of sleeping

## Description
Bounded R18/R19 test synchronization correction supporting fn-112.10 source acceptance, admitted at cd883200c6791527acf263b151e7cf54d7ccd1e8. Follow MILESTONES.md; root owns lifecycle, gates, review and commits. Native fn149/fn128 stay deferred and unverified.

**Touches:** [tools/gomad3/toolchain/build_test.go]

Replace only TestBuildSerializesConcurrentSameKey's25ms sleep with an observed synchronization condition from the actual second builder's context.Done call after acquireBuildLock receives hostfs.ErrContended. A test-local context wrapper may notify once when Done is queried, delegate all context semantics and preserve cancellation. Keep first builder blocked until the second has actually observed lock contention. Do not add a production seam, polling delay, scheduler-yield loop, third build, platform guard, policy exception or generic wait unrelated to lock acquisition. Preserve two results, equal build keys, one actual build and at least one Waited result. Bound start/contention/completion waits and ensure cancellation/release/drain on failure rather than leaking builders. Preserve every other existing assertion, fixture, comment and test byte.

Retain independently fixed expectations before implementation and a matched unchanged baseline. Genuine causal sensitivity must demonstrate that removing/bypassing the contention observation or waiting guarantee is detected; a green timing race is not proof. Review the real lock loop and context invocation before implementing. This is a test-only change; production load/watchdog mechanisms remain unchanged. Retain the existing forbidden-sleep diagnostic as RED and require its exact disappearance with no introduced lint. Scope maintenance/preservation proof to this one fixture and any test-local synchronization declarations added to the same file. Full original-base lint and parent acceptance remain owned and open wherever unproved.

**Quick:** pinned stock Go1.27.1 tests -tags test_dep -count=1: TestBuildSerializesConcurrentSameKey and all TestBuild* controls in ./toolchain on BASE and final; focused sensitivity controls with actual outcomes; affected configured toolchain lint, vet/gofmt, check-only make lint-code-fast FIX=false and original-base integrated lint against951c5516e9e7b3066e7e069adda9565cfd68844c. All Go/build/test/lint/vet/generator commands wait for explicit root serialized-lane grant. Bind source/tool/raw hashes, exits, elapsed time and any inherited red/skipped/unexecuted requirements. Evidence paths .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-61/ are lifecycle exemptions from product Touches. No worker Flow completion/review verdict/commit until root grant, policy widening, native revival, merge/push/PR/CI or new dependencies.

## Acceptance
- [ ] The fixture observes real second-build lock contention before releasing the first, preserving exactly two results, one build, equal keys and Waited assertion.
- [ ] All waits are bounded and failure cleanup cancels, releases and drains safely; context wrapper delegates semantics and reports contention once. No production change or polling/sleep/yield substitute.
- [ ] Matched BASE/final TestBuild* and independently fixed synchronization/sensitivity controls retain actual outcomes; all non-admitted test bytes are reconstructed unchanged.
- [ ] Actual affected lint removes exactly the forbidden sleep, introduces zero findings, and original-base residual diagnostic blocks remain equivalent after line mapping; vet/format/check-only canonical fast lint verified.
- [ ] Independent review and separate integrated checkpoint retain source-bound receipts and parent/native limits; Done only after all requirements still owned here pass, without inferring parent source acceptance.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
