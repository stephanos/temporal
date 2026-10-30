Review complete. The source hunks are small and consistent with the task; the bulk of the diff is deletions plus evidence artifacts.

**What I verified**

- All six package trees are gone; `tools/umpire` now holds only `cmd/umpire` and `temporaldriver`. A repo-wide search for the removed import paths, the backend Make targets, `UMPIRE_BACKENDS_OUT` and `quint.sh` finds only the Testpilot forbidden-import fixture and the recorded historical/archived exemptions.
- Deleted packages imported only testify and protobuf packages that the rest of the module still uses, so `go.mod` needs no tidy.
- The DLQ change keeps pointers end to end. `printTable` dereferences pointers and the JSON encoder emits the same fields, so sorted table/JSON output is unchanged. The `TestDLQCommand_V2` control exercises the list path.
- The `lint-code-fast` loop fix is the minimal correct one: under `set -e` without `pipefail`, the old `[ -d ] && printf` tail made the while loop, and thus the `targets=$(...)` assignment, fail on a deleted final directory. The `if` form returns 0 and still lets `lint-code` failures propagate (negative control recorded exit 2).
- The renewed gaps golden is internally consistent: 81 `Case:` lines match `cases=81 confirmed=81 mismatch=0`. Both header readers (`snapshot.Payload` in Go and `snapshot_payload`/`insta::Snapshot::from_file` in Rust) only require a `source:` line, so the new header parses.
- CI selector, README, AGENTS.md and model/README.md agree with what remains on disk.

## Findings

- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `common/testing/testpilot/duration/duration.go:1`
- **R-IDs**: []
- **Problem**: The only importer of this package was the deleted `tools/umpire/realization`. No Go file imports it any more, so it is now a dead package that still builds and runs in the retained test selector. The ownership row in `.plans/UMPIRE_MODULES.md:137` still says it is "used by `tools/umpire/realization`".
- **Suggestion**: Either delete the package in a follow-up or move its row into the new "Current Go tooling ownership" section with its real consumer named. Not blocking.

- **Severity**: P3
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/temporal/feature/activity/expected/gaps.txt.snap:3`
- **R-IDs**: []
- **Problem**: The committed snapshot carries `assertion_line: 90`. insta writes that field only into pending `.snap.new` candidates and strips it on accept, so this header shows the candidate was renamed into place rather than accepted. All readers tolerate it, but the next `cargo insta accept` or `INSTA_UPDATE=always` run will rewrite the header and produce metadata-only churn on a reviewed golden.
- **Suggestion**: Drop the `assertion_line` line, or re-run the importer and accept through insta so the header matches the sibling kettle snapshot.

**FYI (not findings)**
- The live Go host, Runner, Driver, command and canary ownership rows still sit under the "Historical Scala and Go migration ownership" heading. The new paragraph explicitly says those rows remain live, so the doc is consistent, but moving them under the new heading would remove the need for that caveat.
- The `.PHONY` line still names `fmt-model lint-model fix-model`, and the "Archived Scala targets" comment follows the deleted block. Both belong to task .2 (R2).
- The task's own status records that the complete `make umpire-check-live-tests` rerun against the renewed golden is still outstanding. The prior full attempt passed the slice and failed only on the stale golden, and the importer verification with `INSTA_UPDATE=no` passed on the renewed bytes. That rerun is the closure prerequisite the author already named; it is process state, not a code defect.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | partial | Six trees, backends targets, Quint helper and pins deleted; CI step narrowed to `cmd/umpire`, `temporaldriver`, Testpilot. Build, both vet modes, retained tests, canary build/harness, four separate dependency inventories recorded green at d960113963. Complete `umpire-check-live-tests` pass on the renewed golden still pending per the author's status record. |
| R2 | deferred | Task .2 scope (Scala Make block, `scala-cli` pin). |
| R3 | partial | AGENTS.md, model/README.md, UMPIRE_MODULES.md and tools/umpire/README.md updated and consistent with disk; historical sections marked historical; archived material and the forbidden-import fixture preserved. `model/archived/README.md` and the `.gitignore` comment are task .2. |
| R4 | deferred | Task .2 scope. |

Unaddressed R-IDs: []

Classification counts: 2 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>