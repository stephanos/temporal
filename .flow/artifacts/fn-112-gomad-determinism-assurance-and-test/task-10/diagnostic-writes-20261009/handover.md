# Task 10 diagnostic-write source progress

The four soak diagnostics now check write errors while preserving their literal
bytes, operands, order and primary statuses 2/3. Task
`fn-112-gomad-determinism-assurance-and-test.10` remains `in_progress`. Root owns
review, lifecycle and commits. Required aggregate source lint remains RED.

Tier: session (jev-unavailable(no_key)); requested implementer gpt-6.1-sol at
high; actual execution metadata was not exposed.

The workspace is `/Users/stephan/Workspace/skunkworks/gomad/temporal`, branch
`gomad`, base/HEAD `4de2ba7892570a865c27a178b681e31f28fca79b`. The commit range
is empty. The two admitted source files and this packet are uncommitted. No
worker staging, commit, review verdict, Flow mutation, publication or native
execution occurred. All started command handles are terminal; the shared
Go/build/lint/generator/cache lane is released.

## Controls and preservation

`TestSoakDiagnosticWritesPreserveStatus` runs through public `run`, reusing
the existing test-only counting writer and real read-only-file EBADF writer.
It covers positional usage, invalid seed, pre-schema invalid input, post-schema
report publication failure, failed stdout summary with healthy/failed stderr,
and healthy summary output. Corrected controls pass before production edits and
inside the final full package run. They preserve exact attempted calls and
bytes, zero unexpected stdout writes, unchanged classifications, published
ledger/report correspondence and summary bytes. Report obstruction remains a
directory and no summary is published when report publication fails.

The valid tiny manifests and `--budget=1ns --batches=1` exhaust the real budget
before any batch can launch the absent Gomad executable. Reports assert zero
executions, toolchains and cohorts, with one infrastructure failure. This is
portable adapter evidence and supplies no native qualification or soak bound.

The first fixture run failed on two incorrect expected literals. Its immutable
receipt remains `preservation-controls-before-fix.json`. Pinned stock Go's
`src/os/file_unix.go:26-44` returns EEXIST in its own file-over-directory
preflight, before the filesystem rename; soak's outcome literal is
`infrastructure`. Correcting those independent fixture expectations did not
change production. The meaningful defect control is actual configured errcheck
RED63, followed by exactly four removed diagnostics and zero introduced ones.

The seal proves the four ordered diagnostic calls and every original invalid
input test body unchanged. It verifies 1,098 other tracked product inputs
unchanged from admission, including declared generator inputs, pins, generated
outputs, runtime sources, manifest and documentation. Original first-baseline
and fixed-identity obligations retain their meaning and remain open wherever
the prior packets did not prove them. The two protected `.turbo` files match
root's before hashes and the Git index is empty.

## Current gates

| Receipt | Actual result |
| --- | --- |
| baseline-command-package | Exit0; 47 top-level tests, 195 tests/subtests, no failures/skips |
| preservation-controls-before-fix-publication | Exit0 before production edits; 24 tests/subtests, no failures/skips |
| final-command-soak-set | Exit0; 107 top-level tests, 318 tests/subtests (cmd214, soak22, set82), no failures/skips |
| affected-vet / standalone-errortype | Exit0 / exit0 |
| architecture-source-sets | Exit0; nine tests/subtests including full host-package vet for darwin/arm64, linux/amd64 and developmental linux/arm64 |
| baseline-validate / final-validate / format-check | Exit0 / exit0 / exit0; check-only generated validation and formatting |
| baseline-configured-lint / final-configured-lint | Exit1 / exit1; unfiltered RED63 to RED59, exactly the four soak diagnostics removed |
| make-fast-admission | Exit0; 55 host packages, revision-filtered zero issues; errortype reached |
| make-gomad-original-base | Exit2; 55 host packages, configured original-base RED204 versus retained208; integrated errortype unreached |
| final-seal | Exit0; source/tool/stream hashes, untouched inputs, original test bodies and exact residual inventory checked |

All observations bind frozen source, pinned stock Go1.27.1, lint2.13.0,
errortype, explicit cache/module/proxy settings, timestamps, elapsed time,
numeric exits and separate stdout/stderr hashes. The private temp root is
`/tmp/fn11210-diagnostic.dzr2T3sO`, overlayfs `e6312165da52bad5`. The old FUSE
Git-fixture failure remains in task51's packet. Root admitted this one
changed-temp full ordinary baseline; its success proves no causal explanation
of that old failure.

The 59 residual adapter findings are compatibility_pack24, diagnostic5,
main27 and upgrade3, all outside this admission. No filtering, waiver,
suppression, test weakening or scope expansion was added. The original-base
`951c5516e9e7b3066e7e069adda9565cfd68844c` gate remains RED204. Fresh standalone
errortype does not replace the integrated stage that never ran.

`evidence.json` indexes 16 command receipts; `final-seal.json` records the seal
that produced it. The prior immutable source-acceptance packet is referenced
with hashes, including documentation, guide-check and retained R6/R7/R11
evidence. Native full test-host, runtime execution, scheduled soak and measured
bounds stay deferred to fn-149/fn-128. No partial portable pass is labeled a
native gate. Root may review and commit verified progress; formal task review
and acceptance await the required source-gate resolution.

Defect route:
- prior fixes: original bae373d147 retained; root confirms no competing worker/fix, six predecessors Done and tracker inactive; local memory searches found no soak matches.
- diagnosis: actual errcheck reports four unchecked writes; public healthy/EBADF controls confirm preserved terminal classification and publication behavior.
- introduced by: historical unchecked calls in bae373d147; bisection skipped because no known-green analyzer revision is supplied.
- base: actual configured RED63; corrected preservation controls green before production edits. Head: RED59 with exactly those four findings removed; full affected tests green.
- live: no live-app surface; public command adapter exercised in ordinary tests.

stage: impl-review - skipped(policy: root owns review/lifecycle; required aggregate source lint remains red)
