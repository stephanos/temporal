# Independent diagnostic-write source progress review

## Strengths

The admitted change removes four unchecked diagnostic writes without changing
the command's existing behavior. `soak.go:37`, `:49`, `:59` and `:71` preserve
all four original fmt call literals, operands and call order. Failed usage and
seed reporting still return 2. A failed operation with an empty report schema
still returns 2; an initialized report or failed stdout summary still returns
3. Reporting failure changes neither operation precedence nor completed
publication. The diff introduces no production helper, fallback, injection
seam, framework, dependency, flag, pin or runtime change.

`soak_test.go:37` adds controls through public `run` for healthy stderr and a
real read-only-file EBADF writer. The tests check exact attempted diagnostic
bytes and call counts, absence of unexpected stdout writes, both report-schema
classifications, report-publication obstruction, failed stdout summary and
healthy summary output. They check the retained ledger/report correspondence
and published summary even when stdout reporting fails. The tiny valid
manifests at `soak_test.go:172` use the real 1ns budget. The budget check at
`qualification/soak/soak.go:379` prevents execution before the absent executable
can launch. One infrastructure failure with zero repetitions, toolchains and
cohorts establishes adapter coverage only.

The existing invalid-input test body is byte-preserved. Independent read-only
comparison also verified every other 1,098 tracked product inputs against the
admission base, including generator inputs and outputs, runtime sources,
manifests, configuration and documentation. The task record adds the bounded
admission and two Touches paths without changing acceptance or dependency
edges. Flow runtime state remains `in_progress`; the tracked JSON's `todo`
value is superseded by the separate runtime state reported by flowctl.

The evidence preserves the initial fixture expectation errors. EISDIR and
`infrastructure_failure` were incorrect expectations; pinned Unix Go returns
EEXIST for file-over-directory rename and the soak outcome is `infrastructure`.
These fixture failures supply no behavioral defect claim. The meaningful
defect control is the actual configured analyzer failure, RED63 before the
production edit and RED59 afterward. Raw outputs show exactly the original
soak findings at lines 37, 47, 55 and 62 removed, with zero introduced findings.

Independent reconstruction binds the ordinary admission baseline to aggregate
`5efeac884fbdb101b466760688d6837c7da15c1810985a76efaf47e3dfe8324b`.
Replacing only the final production file with its admission-base bytes yields
`2ed11ca9783eb3132836a112d7b42f382414030a7cc4d98bee777b3b011febf8`,
the passing pre-production publication-controls receipt. Thus the final test
controls passed against the original production source before this fix. The
final product aggregate is
`9a3b419eb5c42d484147ca4d8e447b3a3670500dad0a2ed69f5356f442caab00`.

I independently verified 19 completed packet command records, their 38 raw
stdout/stderr hashes, five tool hashes, preserved prior-packet hashes and
protected user-file hashes. Raw JSON test events agree with receipt events.
Worker evidence retains 16 gates; the final seal and two append-only root
receipts account for the other three records. Numeric exits, source stability,
timestamps and explicit cache/module/proxy/temp inputs are consistent.

| Retained check | Result and limit |
| --- | --- |
| Ordinary command baseline | Exit 0; 195 tests/subtests, no failures or skips |
| Final command/soak/set coverage | Exit 0; 318 tests/subtests, no failures or skips |
| Root independent controls | Exit 0; 24 tests/subtests, no failures or skips, same final aggregate |
| Architecture/public/purity/static boundaries | Exit 0; nine tests/subtests, including Darwin arm64 and Linux amd64 host-package vet plus developmental Linux arm64 |
| Affected vet, standalone errortype, generated validation, formatting | Exit 0 for each |
| Configured affected lint | Exit 1; 63 to 59 diagnostics, exactly four removed, zero introduced |
| Admission-base make lint-code-fast | Exit 0; 55 host packages, revision-filtered zero findings; errortype reached |
| Original-base make lint-code-gomad3 | Exit 2; 204 diagnostics against `951c5516e9e7b3066e7e069adda9565cfd68844c`, versus retained 208; integrated errortype unreached |

## Issues

### Critical

None found in the admitted source progress.

### Important

No introduced correctness, preservation or evidence-integrity defect found.
Required source gates remain RED. The 59 residual adapter findings belong to
`compatibility_pack.go` (24), `diagnostic.go` (5), `main.go` (27) and
`upgrade.go` (3), outside this admission. The original-base aggregate remains
RED204. These retained blockers prevent acceptance and merge readiness even
though this repair introduces zero findings.

### Minor

None found that requires a change to this candidate.

## Recommendations

Root may commit this verified progress under the existing bounded admission
and keep task 10 acceptance open. Do not turn the revision-filtered zero into
a full lint pass or substitute standalone errortype for the integrated stage
that never ran. Preserve the historical guide-check RED and FUSE fixture
failure. The changed private overlayfs ordinary-package baseline supplies no
causal explanation for the old Git-fixture failure. Honor the owner's stop
after the current commit; this review requests no additional task planning.

Keep fn-128 and fn-149 deferred and unverified. The portable budget/publication
controls, static checks and package tests establish no native test-host pass,
scheduled soak result or measured Gomad bound.

## Assessment

SOURCE_PROGRESS_PASS

Ready to merge: No. Required source gates remain RED.

This is an independent source progress review of the uncommitted tracked diff
and new packet at base/HEAD
`4de2ba7892570a865c27a178b681e31f28fca79b` on `gomad`. It is not formal
configured Codex implementation review, formal Flow acceptance, native
qualification or goal/task completion. Scope follows the diagnostic admission,
current task and parent contracts, AGENTS.md, README and MILESTONES delivery
order. Root owns lifecycle, staging, commits and all acceptance decisions.

The requested project reviewer is `gpt-6.1-sol` at high effort, from the same
model family as the writer. Actual runtime model telemetry was not exposed;
this report makes no claim about actual execution metadata. I ran no Go,
test, lint, build, cache or generator command and made no lifecycle, index,
history or external mutation. The sole reviewer write is this report.

Audit input SHA-256 values:

| Input | SHA-256 |
| --- | --- |
| admission.md | `b7c7975cacf1d752b07e43b1fc865e5c4ef52cd16e2fa8c2a99162d1b03cc67f` |
| handover.md | `e4072d284d49252915c01e8fbab19931c4fef16aa260d0ae1dfdab5e022b0245` |
| evidence.json | `45a99efe3c26b9170fddd717ccfcabdadf9d4dbddb7d43245a2f6a871b07f490` |
| final-seal.json | `84c289278700eac87ac6359e33550313c81adef58ec8977f884c16670e23d314` |
| soak.go | `3a32736f267a8d0949dfc58d48656c51d0f63429c5e70575bd9b95cc5f8d8315` |
| soak_test.go | `f396db0b606d024246fc83cc74d3ee65a065a60054e8522b62409d2496c1c81d` |
| root-independent-controls.json | `fcf4253d9770cd33feae3cdb9a4281e2b094c073396ca500c4f0c272fa13c80a` |
| root-packet-audit.json | `3371d43457d859f0c0c6e15e76e6f44461015bcd8c5c04b17d3ad11a1478215c` |
