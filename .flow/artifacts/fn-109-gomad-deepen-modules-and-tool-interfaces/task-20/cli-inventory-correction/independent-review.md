# Task 20 CLI inventory correction independent source review

The nine added flag descriptions match their current Go registrations,
validation and consumers. The bounded documentation source progress is ready
with one minor citation correction. This review grants no formal implementation
receipt, native qualification, task completion or acceptance waiver.

## Review identity and scope

Reviewed on 2026-10-04 in `/Users/stephan/Workspace/skunkworks/gomad/temporal`,
branch `gomad`. Base and HEAD both equal
`5b8261599475f0cef21b6c7d23d272fc9439b26f`. I reviewed the uncommitted
`git diff BASE -- tools/gomad3/CLI.md .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/documentation-evidence.md`
and the new correction artifacts directly. The empty BASE..HEAD range does
not contain the correction.

The requesting-code-review skill supplies this independent review format.
I read AGENTS.md, the complete Gomad README and MILESTONES, the correction
admission, task 20, R9/R18/R19/R20, and the committed task-21 preservation
report. The Flow prose skill governs this report's prose.

Requested reviewer model is `gpt-6.1-sol` at high effort, under Codex routing.
Tier is session (`jev-unavailable(no_key)`). The writer and reviewer are from
the same family. Actual executed-model metadata is unavailable, so the
requested model is not asserted as an observed execution identity.

Reviewed document SHA-256 values, unchanged at the end of source inspection:

| File | SHA-256 |
| --- | --- |
| `tools/gomad3/CLI.md` | `8184291764fed265523cb9407a288987210a6270293265dce3a152899c81aa47` |
| `documentation-evidence.md` | `cbba93e5b2386fab23e440fb62448f680b1b6032cabd60c1d2dda83d48d6dde8` |
| Correction `source-admission.md` | `d822e4dd0bb94ebcd8929a5ff7f0c745d7dba692c07e88c8086f8b4eadbdb3d7` |

## Strengths

The CLI diff consists of exactly 18 added lines, forming nine substantive
paragraphs. All existing shell fences and the complete command index remain
byte-identical to the predecessor; SPEC.md is unchanged. The live registration
comparison and preserved source inventory corroborate that the correction
adds no public behavior.

The descriptions cover operational details beyond flag names. I verified the
following directly in source under `tools/gomad3`:

| Flag | Source behavior inspected |
| --- | --- |
| `--toolchain-root` | Nine command registrations; explicit/environment/manifest/adjacent resolution in `toolchain/installation.go`; clean absolute non-root validation, manifest refusal and existing replay compatibility checks. |
| `--terminate-grace` | Explore/plan shared parser and qualify registration default to two seconds; `runner/runner.go:1289` permits zero and requires the grace to fit both deadlines; the Unix supervisor reserves cleanup time and escalates process-group SIGTERM to SIGKILL. |
| `--env` | Shared explore/plan and qualify registrations; `parseEnvironment` validates ASCII names, reserved controls and loader prefixes, duplicates, NUL bytes and missing separators; seed environment construction adds UTC and recorded environment is restored on replay. |
| `--io-ro-mount` | `ParseMappings` resolves relative sources using the selected working directory, validates source directories and destination overlap; broker limits, immutable first captures, symlink/hard-link/special-file refusal and unstable capture detection; `gomadfs/fs.go` enforces EROFS for write-capable mounts; portable plans capture complete configured trees. |
| `--world-transition-limit` | Shared parser and qualify use 64 MiB; byteSize rejects zero, leading zeros and overflow; Runner requires a positive value without a smaller flag-specific upper cap; recording and execution composition bound encoded transitions separately from snapshots; plan/resume/replay retain the recorded bound. |
| `--observed` | Replay defaults to no copy; verification returns before execution; observed output copying creates the directory and replaces stdout/stderr using `os.WriteFile`; copy failure reaches CLI status 3. |
| `--max-bytes` | Parser default zero triggers parent stored bytes plus 1 MiB with saturation; final publication accounts for manifest and payload bytes; intermediate retained candidates use parent-derived limits; an unchanged minimization returns the parent; publication capacity failure reaches status 3. |
| `--min-free-bytes` | Qualify-set defaults to 2 GiB and uses the shared positive byte-size parser; Statfs uses Bavail/Bsize for the current user; equality passes; low space stops subsequent supported workloads and yields status 3; manifest-only check returns before execution. |
| `--prune-qualified-artifacts` | False by default; only qualified seeds with successful replay configured are selected; pruning checks every execution's attempted matching replay and retained paths; the set checkpoints projected evidence before campaign deletion and keeps qualification reports; pruning errors reach status 3. |

The evidence document separates the prior formal review's recorded source
from the new correction and keeps API/identity reconciliation and native
qualification open. The historical preservation audit remains an immutable
finding against its original source.

## Findings

### Critical

None.

### Important

None.

### Minor

1. Incorrect World flag registration citation. `documentation-evidence.md:34`
   cites `qualify.go:61,67`, and correction `check.py:71` repeats line 67.
   `cmd/gomad/internal/cli/qualify.go:67` registers `--choice-bytes`;
   `--world-transition-limit` is registered at line 65, with its default at
   line 61. The CLI paragraph is correct, and the checker independently
   validates the actual registration inventory, so this does not change
   documented behavior. Change the evidence citation to `61,65` and the
   checker source row to `65`, retaining the original captures and this review.

## Verification and checker limits

I ran the correction `check.py` in normal read-only mode, with output summarized
through Python. It exited 0 and all 19 check rows passed. Its 978-file source
inventory found only CLI.md changed, preserving the other 977 entries; its
2,396 historical artifact hashes matched. Admission identity, tracked diff,
HEAD, links, fences, shell examples, full command index and SPEC preservation
passed. The index remained empty, and only the two admitted tracked documents
were modified.

The checker requires descriptive paragraphs and syntax/default/effect terms,
and compares live flag registrations to the committed inventory. It does not
prove semantic truth merely by finding terms or printing source rows. The
consumer reads above supply that independent semantic review. The incorrect
line-67 evidence row illustrates this limit.

I inspected the retained baseline document check, which records all nine
missing descriptions and passing preservation checks. An independent read
of `git show BASE:tools/gomad3/CLI.md` reproduced zero flag-bearing paragraphs
for each of the nine names without editing the current tree.

I verified both final Quick receipts against their retained log SHA-256 values
and recomputed the current 978-entry source digest. Both logs match, and both
receipts' pre/post digests equal current
`8c39e4e71a8fbe168d0e73d6242c04f2667a38d2decb314e2a3f921d1128af24`.
The focused log records both named tests passing, with receipt exit 0 and
0.330 seconds elapsed. The validation receipt records exit 0 and 2.422 seconds
elapsed. I reused these source-matching results without rerunning their gates.

Commands I ran include `pwd`, scoped `rg`/`rg --files`, `cat`, `sed`, `nl`,
`git status --short`, `git diff BASE -- <the two documents>`, scoped
`git diff --check BASE -- <the two documents>` (exit 0), diff stat/numstat/name
reads, `git diff --cached --name-only`, `git rev-parse HEAD`, `git show` through
an inline read-only Python check, `shasum -a 256`, `uname -sm`, and
`command -v go`/`ls -ld` toolchain probes. Inline Python also compared receipt
and source hashes and resolved evidence source citations. Two early citation
probes failed because abbreviated filenames needed their preceding directory;
the corrected directory-aware probe exited 0. A stock-toolchain source-path
probe and an initial filesystem search included absent paths; later source
reads resolved the mount behavior. These were diagnostic path errors, not
product-test failures. No snapshot mode or run_checks.py execution occurred.

## Recommendations

Return the minor citation correction to the documentation writer, retain these
original observations, and review the bounded correction against new hashes.
Before root changes the task lifecycle, date the `in_progress` and pending
review statements in `documentation-evidence.md:12-14,156-160` explicitly to
this correction source freeze. They describe the reviewed tree correctly now,
but root's planned blocked progress checkpoint will otherwise make their
undated present tense stale. This is a handoff recommendation, not a claim
that the current recorded state is false.

## Assessment and qualification limits

Ready for a bounded SOURCE PROGRESS commit with the minor citation correction.
No Critical or Important source findings remain. Executable source is
unchanged, and MILESTONES verification instruction 3 supports document/diff
checks and reuse of source-matching focused evidence.

The observed host is Linux aarch64; the patched `.toolchain/bin/go` is absent.
The retained stock-Go checks are developmental. Both native gates, R18 public
API and independent identity reconciliation, R19, task 19 formal review,
inherited D5 and fn-105.5 closure remain open. The reported existing root lint
failure was not rerun. Formal review remains conductor-deferred, with no
backend substitution. The old three-draw formal verdict applies only to its
recorded earlier source. This fresh source review is not a formal SHIP receipt.

I changed only this new report through apply_patch. No production, original
artifact, index, HEAD, branch, lifecycle, Flow or review-ledger mutation occurred.
All reviewer commands have exited; no running command is handed back.
