**Scope read:** task fn-256.1 and parent spec acceptance criteria, the source hunks (tdbg, Makefile, workflow, four docs, gaps golden), and the preflight artifact summaries. The removed Go trees are deletions only; I confirmed no live Go file imports them and only the Testpilot synthetic negative fixture names `tools/umpire/ir`, which the spec exempts.

## Findings

**P3 · confidence 75 · introduced · `.plans/UMPIRE_MODULES.md:137` · [R3]**
Problem: Inside the now-historical table, the Durations row still states it is "used by `tools/umpire/realization`", a deleted package. The new "Current Go tooling ownership" section at lines 68-82 declares every row below it historical, so a reader is told these paths are stale. Still, this row describes a retained live module (`common/testing/testpilot/duration`), so its stale consumer could mislead someone checking permitted imports for a retained package.
Suggestion: Drop the "used by" clause or move the Durations row into the current section. FYI only; the spec's "mark historical" requirement is met by the new section.

**P3 · confidence 75 · introduced · `tools/umpire/README.md:44` · [R1]**
Problem: The Checks block documents `./tools/umpire/...` while CI now runs the two explicit packages. Both resolve to the same two packages today, so no behavioural difference. If a package is later added under `tools/umpire/`, the README command will test it and CI will not.
Suggestion: Either mirror the explicit CI selector in the README or leave as is. FYI.

## Verified correct

- **tdbg `ListQueues`** at `tools/tdbg/dlq_v2_service.go:444-461`: pointer slice removes the protobuf mutex copy flagged by vet. Sort still orders by `MessageCount` through the pointer. `printTable` dereferences pointers before reflecting fields, and JSON encoding of a pointer element is identical to the value form. A nil element from the server would have panicked before too, so no new failure mode. The baseline artifact records the original vet failure at lines 450/459/460, and the DLQ V2 control passed after the fix.
- **`lint-code-fast` loop** at `Makefile:839`: the old `[ -d ] && printf` form returned nonzero when the last changed directory was deleted, which under `set -eu` inside the command substitution failed the target. The `if` form skips deleted directories without altering the loop's exit status. The positive wrapper run passed and the `/bin/false` linter control still returned status 2, so real linter failures are retained.
- **CI selector** at `.github/workflows/umpire.yml:60`: the explicit package list is exactly the two remaining packages under `tools/umpire/`. Live and canary jobs are unchanged.
- **Makefile hooks**: `umpire-check-backends` and `umpire-install-backends` removed from `.PHONY` and the target block. The surviving `UMPIRE_BACKENDS` references at `Makefile:1248,1279` belong to the Rust model section that fn-254 owns, not to the deleted Go targets.
- **Gaps golden renewal**: the committed `gaps.diff` artifact shows exactly 1178 removed `Generic case` lines, 2 added `Case` lines, and 4 `Prediction` lines replacing former `Unannotated` accepted rows, matching the stated accounting. The Summary line moves 79 to 81 confirmed with 0 mismatch and the retained body has no Mismatch or Accepted entries. The new insta header (`source:` plus `assertion_line:`) satisfies the Go reader in `model/umpire/hosts/go/snapshot/snapshot.go:11-33`, which only requires a `source: ` line and rejects binary kind. The importer verification run with `INSTA_UPDATE=no` returned status 0, and no Model or catalog file changed.
- **Dependency inventories**: the reject check over all four `go list` logs matched no removed prefix.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | partial | Six trees, backends targets, Quint helper and pin deleted; build, both vet modes, retained tests, canary build and harness pass per preflight artifacts. The complete `make umpire-check-live-tests` rerun on the renewed golden is still pending by the author's own statement, so closure cannot claim it yet. |
| R3 | partial | `AGENTS.md`, `model/README.md`, `.plans/UMPIRE_MODULES.md` and `tools/umpire/README.md` describe the retained packages and mark migration records historical. The `model/archived/README.md` and `.gitignore` comment portions of R3 are owned by task fn-256.2 (`satisfies: [R2, R3, R4]`) and remain unaddressed in that task's scope. |

Unaddressed R-IDs: []

Code-smell baseline: nothing notable in the changed source hunks; the Go and Make edits are minimal.

Classification counts: 2 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":[]}
```

Note for the author: this verdict judges the source change. It does not stand in for the pending full live gate, which the acceptance criteria still require before the task closes.

<verdict>SHIP</verdict>