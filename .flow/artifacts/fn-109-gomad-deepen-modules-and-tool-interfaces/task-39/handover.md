# Task 39 source progress

Target file helpers now report all nine formerly unchecked cleanup returns at
their original lifetimes. Nil cleanup preserves the primary error and its unwrap
shape; sole cleanup failures return directly and simultaneous failures join
primary first. Hash-reader failure clears digest/size, and GOPATH removal failure
clears the adapter digest. These newly surfaced failures are a bounded behavior
change, so this handover makes no universal byte-equivalence claim.

Status remains `in_progress` under SOURCE_PROGRESS_ONLY. Root owns independent
source review, Git/index/commits, Flow lifecycle and admission of another writer.
No worker review, bridge, commit or lifecycle mutation ran. No executed-model
metadata was exposed.

Tier: session (jev-unavailable(no_key)); explicit AGENTS Sol pin retained.
stage: impl-review - skipped(policy: conductor-deferred - root owns the gate)

The deferred hash/input Close and adapter RemoveAll remain at function return.
Copy releases output before input, including both early output-error branches.
Write checks Close immediately after its original Chmod/Write/Sync failures.
The explicit `close prepared exec target: %w` branch and successful final Close
remain intact. No branch removes a partial destination. The mutation fixture
reports a cleanup failure with nonfatal `t.Error(closeErr)` before its unchanged
primary `t.Fatal(err)` assertion. All original comments and other assertions stay
intact.

## Verification

`baseline: red` records actual unfiltered pinned target lint before source edits.
[BASE lint](baseline-lint.log) reports exactly nine errcheck findings at the
admitted sites. [Final lint](final-lint.log) exits 0 with `0 issues.`; no new
diagnostic, suppression, configuration or policy change was introduced.

Four real-file controls were added before production edits. They pin the literal
17-byte `#!/bin/sh\nexit 0\n` executable, digest
`sha256:306c6ca7407560340797866e077e053627ad409277d1b9da58106fce4cf717cb`,
source mode 0751, copied mode 0700, private-write modes 0400/0700 and empty/nonempty
bytes. Missing, symlink, nonexecutable and directory sources preserve validation
precedence over an existing destination. Collision/missing-parent cases preserve
exact error text, direct/single-wrapper concrete error shape, cause and existing
source/destination bytes. These are preservation controls, not cleanup-fault
reproductions.

The first [BASE characterization](baseline-controls.log) failed because the new
directory-copy expectation incorrectly required a direct error. Actual BASE
returned its existing `*fmt.wrapError`; the control was corrected to require that
wrapper and its direct plain cause. This was a test-authoring error, not the
production defect. [Corrected BASE controls](baseline-controls-corrected.log) and
[final controls](final-controls.log) both pass all 14 selected top-level tests,
including four cleanup tests and their twelve subcases. Task 38's existing
canonical/projection/digest controls are reused without alteration.

| Final command | Exit | Observation |
| --- | --- | --- |
| Focused target Quick command with `-v -count=1 -tags test_dep` | 0 | 14 top-level tests pass, 0.044 package seconds |
| Unfiltered pinned golangci-lint v2.13.0 on `./target` | 0 | Zero issues, all nine mapped findings removed |
| Adapter prepared source-set pin test | 0 | SKIP at adapter_regenerate_test.go:41; `.toolchain/bin/go` absent, pin proof remains unavailable |
| `TestPackageArchitecture` | 0 | One executed test passes, 0.741 package seconds |
| Pinned errortype vet on `./target` | 0 | Empty diagnostics |
| `gofmt -l` on the four Touches, empty-output check | 0 | Empty output |
| `git diff --check` | 0 | Empty output |

Each named log has a corresponding `.receipt.txt` retaining the exact command,
working directory, offline stock-Go environment, tool/config hashes, start/end
UTC, whole-second elapsed duration, terminal exit, log hash and equal before/after
source-manifest digests. Every source check ran on frozen inputs. BASE lint's
manifest matches [task 38's final manifest](../task-38/final-appendf-controls.before.sha256)
at `a5eb12d3228b6fe5b829532d784229a2c8bdcfe7df61d53f81cba5646dbec7af`.
Corrected BASE controls add only the new test and bind
`306ce57f417e2ef3e63dc81ee1cf2dc34a2215738437bbeacee321a7ef1f7018`.
All final checks bind the single [final source manifest](final-source-manifest.log)
at `f49d379db52a343ea61d5a0824a463dd66f0ff5f014ea774a8bc08099be52520`.
[Protected-input comparison](final-protected-inputs.log) verifies 1,036 unchanged
entries, exactly three admitted existing files changed and the one new cleanup
test. The manifest includes nested module files, runtime/pin/profile/generator
inputs and lint configuration; it does not establish whole-repository identity.

Similar code search selected extension of the three existing target file helpers
and conditional cleanup composition from `artifact/open.go:271`. Makefile
VERSION_INPUTS, BOUNDARY_INPUTS, COMPATIBILITY_INPUTS and protocol generation's
explicit livecap input list were inspected. None of the four Touches is a
generator/schema/template/overlay input, and all those inputs remain unchanged;
generator validation was not applicable to these cleanup-only edits.

## Acceptance still open

Real first-Close, simultaneous primary/cleanup, post-open Chmod/Write/Sync and
RemoveAll faults remain unexecuted. Separate real-file calls demonstrate concrete
error/unwrap shapes, not cross-call primary pointer identity. Genuine fault
composition, derived-result clearing and once-only cleanup ownership still need
independent inspection and eventual lawful execution evidence. No filesystem
framework, private injection seam, descriptor theft, second-Close proof or
source-text behavior assertion was added. The adapter-pin test's exact unavailable
input is retained in [its log](final-adapter.log).

Root's fresh independent source review and source-progress commit remain pending.
Every original R18/R19, task21/predecessor, matched first-baseline identity,
full/default/functional/affected-consumer/formal/native Darwin requirement remains
open wherever unproved. No unchanged unsupported-runtime/full/native gate was
retried. Developmental stock Go1.27.1 linux/arm64 results establish only the listed
source controls. Transferred native Linux execution remains under fn128 and is
nonblocking. Fn113 retains adapter regeneration approval, pins, publication and
qualification ownership.

Defect route:
- prior fixes: root serialized this owner after independently reviewed task38 commit; relevant target/cache bug memories read. PR/tracker/other-branch checks not done under the bounded conductor-owned dispatch.
- diagnosis: actual BASE unfiltered lint confirms nine unchecked cleanup returns; final actual lint confirms exactly those findings removed with zero introduced diagnostics. Genuine cleanup-fault behavior remains an explicit proof gap.
- introduced by: not bisected; no known lint-green target revision supplied, and the dispatch forbids worktrees/history changes.
- base: lint exit 1 at 5c5c025adeabacd9f3217de2b9ab8f7171b0c33c; corrected real-file controls pass on unchanged production. Head: uncommitted frozen final source lint and controls exit 0, bound by the final source manifest.
- live: no live app surface; these are host file helpers. No fault-test reproduction commit was made because root owns Git and no lawful cleanup-fault seam exists.
