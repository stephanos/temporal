**Scope read:** task file, parent spec R-IDs, every source hunk (CI selector, Makefile, tdbg, four docs, gaps golden), the preflight summaries and gap-acceptance/import records, and the unchanged callers the hunks meet (`printTable`, `jsonEncoder.Encode`, `snapshot.Payload`, `dependency_boundary_test.go`).

## Findings

**Severity:** P3
**Confidence:** 75
**Classification:** introduced
**File:Line:** `.plans/UMPIRE_MODULES.md:137`
**R-IDs:** [R3]
**Problem:** The Durations row for the retained package `common/testing/testpilot/duration` still says it is "used by `tools/umpire/realization`". That importer is deleted in this change, so a current-package row now names a consumer that does not exist. The new "Current Go tooling ownership" paragraph says retained rows below still describe live ownership, which makes this row read as current.
**Suggestion:** Change the dependency cell to state the package has no current in-repo importer (or name whichever one the owner intends to keep), or move the row under an explicitly historical heading.

## FYI (non-blocking)

- **Orphaned package.** `common/testing/testpilot/duration` had exactly one importer, `tools/umpire/realization/duration.go`, which this change deletes. It still builds and its tests pass in the retained-tests preflight. Removing it is outside this task's stated files; the owner decides.
- **Snapshot header carries `assertion_line: 90`.** The renewed `gaps.txt.snap` was committed from the insta `.snap.new` verbatim rather than through `cargo insta accept`. Both consumers tolerate it: `model/umpire/hosts/go/snapshot/snapshot.go:11` only requires a `source:` line, and the no-update verify run passed. Two other committed snapshots already carry the same key, so there is no failing scenario.
- **Live gate still pending.** The task's second acceptance box requires an actual `make umpire-check-live-tests` pass. The preflight status record says the complete rerun is still running and makes no green claim. The spec's Decision Context already orders closure behind that gate, so this is a settled-plan condition for integration, not a code finding.

## Verified integration points

- **tdbg pointer switch** (`tools/tdbg/dlq_v2_service.go:444`): `items` is `[]any`, so `jsonEncoder.Encode` falls through to `encoding/json`, which dereferences pointers and emits the same struct-tag field names as before. `printTable` already unwraps pointers at `tools/tdbg/util.go:251`. Sort reads fields through pointers unchanged. JSON and table output are preserved, and the list-queues control passed.
- **lint-code-fast loop** (`Makefile:839`): the `if` form makes a missing final directory return status 0 instead of leaking the `[ -d ]` failure into the `$(...)` assignment under `set -eu`. The false-linter control still returns status 2, so real failures propagate.
- **CI selector** names exactly the two retained packages, which matches the current `tools/umpire` tree.
- **No live references** to the six removed trees or the backends targets remain outside `model/archived`, historical `.flow`/`.plans` records, and the synthetic forbidden-import fixture at `common/testing/testpilot/dependency_boundary_test.go:119`, which still exercises `forbiddenPrefixes` correctly since the prefix is `tools/umpire`.
- **`UMPIRE_BACKENDS=require`** in the current Rust section of the modules map refers to the Rust TLC agreement gate at `Makefile:1248`, not the removed Go hook.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | partial | Six trees, both backends targets, `quint.sh` and the CI step are gone; build, both vet modes, four dependency inventories and `canary-build` pass in the preflight records. The actual `umpire-check-live-tests` pass is still pending by the spec's settled ordering. |
| R2 | deferred | Owned by task .2 per the spec. |
| R3 | met | `AGENTS.md`, `model/README.md`, `.plans/UMPIRE_MODULES.md` and `tools/umpire/README.md` describe `cmd/umpire` and `temporaldriver`; archived material and the negative fixture are preserved and recorded as exemptions. One stale row noted above. |
| R4 | deferred | Rust gate section untouched; owned by fn-254. |

Unaddressed R-IDs: []

Suppressed findings: 0
Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>