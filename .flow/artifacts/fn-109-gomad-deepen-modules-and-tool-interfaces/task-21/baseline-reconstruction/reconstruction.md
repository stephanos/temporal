# fn-109 historical baseline reconstruction

The verified nested-module baseline is `6782b55f49a0317b230e827ea2a63a37d116d502`
plus the dirty fn-108.2–.6 changes recorded by fn-109's first task. It is available
at `/tmp/fn109-baseline-reconstruction.lDSSw8Gx/tools/gomad3`. This is input
preparation only. Task 21 has not been started or claimed, its dependencies 19/20
remain unresolved, and no acceptance or qualification claim follows from this
reconstruction. No commit was created (`commits: []`).

The source manifest SHA-256 is
`d78601b3176195f8cc06860f5499e757a2d04333b9211f0ed13a92976b017845`.
The input manifest SHA-256 is
`5b1904a15a5e11dd93926465d97c5467ab2a008f48b20610d9b0ffcfa4e8d0e6`.
These identify the newly reconstructed nested-module files and retained inputs,
respectively; they are not a claimed historical full-repository snapshot digest.

## Inputs and exact verification

`reconstruct.py` read the base through `git archive` and `git ls-tree`, with no
checkout, worktree, index operation or shared source edit. All 663 extracted
files matched their full base Git blob identities. It mechanically applied only
the 24 tracked modifications in fn-108's `final-working-tree.diff`. Every hunk
matched its exact old line range with zero search, offset or fuzz; every old and
new Git blob hash matched the diff's retained index prefix. That input's full
SHA-256 is `47afb28785b4680a405c198bb07a334307ebbb4c11e3e5fbef9ae3a114513855`,
consistent with the abbreviated hash in fn-108 `final.md`.

The seven additions came from exact final new-file blocks:

| Input | Selected paths beneath tools/gomad3 |
| --- | --- |
| task5.diff | runner/completion.go, runner/completion_test.go, runner/completion_characterization_test.go |
| task6.diff | runner/retention.go, runner/retention_test.go, runner/retention_characterization_test.go |
| task4-working-tree.diff | upgrade/upgrade_unix_test.go |

The overlapping tracked-file hunks in those task diffs were excluded.
`evidence.json` records selected block starting lines, hunk counts, excluded
paths, all input hashes, every reconstructed changed-file hash, base hashes and
command results. The task-6 review says its stored diff includes the final test
helper renames, and that final block was used.

The complete reconstructed set is exactly the 663 base files plus those seven
additions: 670 paths. Its 24 modified paths and seven new paths exactly match
`final-protected-paths.txt`. All 670 paths and individual physical line counts
match `final-size-files.txt`; the other 639 files remain byte-identical to the
base. `source.sha256` contains SHA-256 for every reconstructed file.

The fn-109 task-3 and task-6 preimage manifests and all 69 available corresponding
preimage files were checked against their retained SHA-256 values. Of those,
24 equal the reconstructed baseline. The remaining 45 are recorded as later
states or later additions and were not substituted. For example, task-6's
retention.go changes the baseline `CampaignSpec` parameter to `campaignRequest`;
completion.go and completion_test.go still match exactly. These crosschecks
support provenance where equal and do not redefine the baseline where unequal.

## Execution and bounds

The successful reconstruction command was
`python3 .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-21/baseline-reconstruction/reconstruct.py`.
It exited 0; internal execution time was 1.123527 seconds, from
2026-10-04T04:30:34.979860Z to 2026-10-04T04:30:36.103387Z. Its subprocesses
(`mktemp -d`, `git archive`, `git ls-tree`) each exited 0 and have UTC timestamps
and individual durations in `evidence.json`. An initial attempt exited 1 before
extraction because the archive guard rejected the legitimate top-level `tools`
directory; the guard was corrected. Its unused scratch directory remains
`/tmp/fn109-baseline-reconstruction.ZSYMJ26M`; nothing was deleted.

Only the artifacts in this directory were written in the shared repository.
All reconstructed source remains in the isolated scratch directory. No Go
package loading, tests, builds, generation, memory/copy measurement, native
qualification, Flow mutation or Git mutation ran. No bridge or additional
agent was used. Requested routing was gpt-6.1-sol/high; actual execution metadata
is unknown, with the previously judged session fallback unavailable(no_key).

Final `sha256sum -c` checks of both manifests exited 0 with pipe failure checking
enabled, reading inputs from the repository root and source from the scratch
root. Their exact commands and tool-reported durations are in `evidence.json`.

## Provenance limits

The retained historical gate fingerprint `a875ec2570434ad6` lacks a retained
fingerprint-generation command, so this reconstruction does not claim to
reproduce it. Most new-file blocks lack a retained historical full-file hash;
their exact bytes are recovered from the selected complete addition hunks,
checked against hunk lengths and final physical-line inventory, with matching
later preimages used only as supplementary evidence. `upgrade_unix_test.go`
also matches its retained Git blob hash prefix. The reconstruction establishes
the specified nested-module input from retained artifacts, not a clean checkout
or the planning revision `d4d800fb47`, and does not reconstruct other modules or
the entire dirty repository.
