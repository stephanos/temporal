# fn-108.2 evidence: unused helpers, decimal/decoder, minimizer serialization, duplicate canonical validation

Platform darwin/arm64, host go1.27.1, patched toolchain `tools/gomad3/.toolchain/bin/go` (key
`8d28bd44…`). Base `6782b55f49a0317b230e827ea2a63a37d116d502`. Nothing is staged or committed
(`git diff --cached --stat` is empty); the change is the working-tree diff in
[task2.diff](task2.diff), SHA-256 `20bd74c008ad6376bf9aa97912051ede1f27e1d5799891b92643e507d31dc348`.
A concurrent worker (fn-108.4) edits `tools/gomad3/upgrade/**`, `architecture_test.go` and
`ARCHITECTURE.md`; those paths are not part of this task.

## Removed symbols

Each symbol was searched in the whole repository with `git grep --untracked -w` (tracked and
untracked-not-ignored files, so every build tag and GOOS file, generators, templates, the runtime
overlay, both sibling packages, `tests/`, Makefiles, `.github`, specs and plans) before the edit.
The output is [task2-consumer-search.txt](task2-consumer-search.txt). No symbol had a live
supported consumer (the only calls were inside deleted code or the deleted round-trip test lines,
as the table states); none was kept.

| Symbol | File | Physical lines removed | Consumer search result |
| --- | --- | --- | --- |
| `validateGoCapabilityClosure` | `target/capability.go` | 11 | declaration only; fn-108/fn-109 task text mentions its removal |
| `matchesExpectation` | `qualification/set/set.go` | 22 | declaration only |
| `firstReplay` | `qualification/set/set.go` | 9 | two calls, both inside `matchesExpectation` |
| `deterministicCapturedInputs` | `runner/deterministicio.go` | 8 | declaration only |
| `orderRunCompletions` | `runner/campaign.go` | 4 | declaration only |
| `removeCompletedPartial` | `runner/internal/campaign/campaign_journal.go` | 4 | declaration only |
| `decimal` type, `MarshalJSON`, `UnmarshalJSON` | `deterministicio/domain.go` | 30 | used only by its own methods; never a field or variable type |
| `decodeCanonicalJSON` | `deterministicio/domain.go` | 25 | declaration only |
| imports `io`, `strconv` | `deterministicio/domain.go` | 2 | unused after the two removals above |
| `Encode` | `runner/internal/minimizer/minimizer.go` | 7 | one call, in the round-trip lines of `minimizer_test.go` |
| `Decode` | `runner/internal/minimizer/minimizer.go` | 11 | one call, in the round-trip lines of `minimizer_test.go` |
| round-trip lines | `runner/internal/minimizer/minimizer_test.go` | 11 | test-only use of `Encode`/`Decode` |
| second `CanonicalJSON` + `bytes.Equal` block, import `bytes` | `runner/portable_plan.go` | 5 | see "Duplicate canonical validation" |
| `canonical`/`bytes.Equal` terms | `runner/internal/campaign/merge.go` | 1 (net) | see "Duplicate canonical validation" |

The only importer of the minimizer package is `runner/minimize_operation.go`; it uses `State`,
`New`, `Next`, `Commit`, `Reduction` and `ImplementationSHA256`. The first unqualified
`Encode(`/`Decode(` query in the search file used `\b`, which `git grep -E` on darwin does not
honour; the file keeps that query and the corrected `-w` rerun below it.

Kept unchanged: `canonicalJSON` and `validateCanonicalStrings` in `deterministicio/domain.go`;
minimizer `Validate`, `seal`, `stateIdentity` and the budget and stop-reason assertions of
`TestStateStopsAtAttemptBudgetAndRoundTrips`. The test keeps its name because the task removes only
the round-trip lines and the baseline disposition table is keyed by test name; the name now
overstates what the test covers.

## Duplicate canonical validation

`canonicaljson.DecodeCanonicalJSON` strict-decodes, re-encodes the destination and compares the
bytes. `openCampaignPlan` and `OpenMergedCampaign` repeated the re-encoding on the decoded value
directly afterwards. The decoder encodes through the pointer and the readers encoded the value; the
two encodings can differ only when a reachable type has a pointer-receiver `MarshalJSON` or
`MarshalText`. The module has none: `record.Uint64String` and the five `world` identifier types use
value receivers. The removed branches were therefore unreachable.

- `runner/portable_plan.go`: the block is gone; the schema, mapping, strategy, guidance, on-failure,
  plan-hash and shard condition and everything after it are unchanged.
- `runner/internal/campaign/merge.go`: the schema, schema-version, plan-hash, selection and count
  terms stay in one condition. The return is now `errors.New("merged campaign record is invalid")`
  where it was `errors.Join(errors.New(<same text>), err)`; `err` could only be nil once the two
  removed terms are gone. The deferred `errors.Join(retErr, root.Close())` wraps the result as
  before, so the message and `errors.Is`/`errors.As` results are the same.

Neither reader had a noncanonical-input rejection test. One table-driven test per reader pins the
exact error for an incomplete object, trailing whitespace, trailing data, an unknown field, malformed
input and an invalid protocol identity: `TestOpenCampaignPlanRejectsNoncanonicalAndInvalidDocuments`
(`runner/portable_plan_test.go`) and `TestOpenMergedCampaignRejectsNoncanonicalAndInvalidRecords`
(`runner/internal/campaign/merge_capacity_test.go`). Replacing the remaining decoder call with
`StrictDecode` turns the two noncanonical cases red in both tests
([task2-mutation-red.txt](task2-mutation-red.txt)); the mutation was reverted.

## Size

Counting rule v2, [task2-size-per-file.txt](task2-size-per-file.txt) (per file, by owner) and
[task2-size-compare.txt](task2-size-compare.txt) (whole tree, which also contains fn-108.4's
uncommitted edits). Listing: [task2-size-files.txt](task2-size-files.txt).

| Scope | Class | Physical | Code lines | Code bytes |
| --- | --- | --- | --- | --- |
| this task's files | production-go | -139 | -127 | -4110 |
| this task's files | test-go | +63 | +61 | +2649 |
| whole tree (both tasks) | production-go | -158 | -146 | -4631 |

Overlay, generated, protocol-input and non-Markdown other files did not change; no file changed
class; `size-compare.sh` exits 0 (residual code -146, code bytes -4631).

`api-capture.sh` output compared with `api-baseline/` by `diff -r`: empty (exit 0), so `go doc -all`
of the 19 public packages plus `gomad3sim` and the 39 CLI usage/help captures are unchanged.

## Commands and results (darwin/arm64)

Run from `tools/gomad3` with `env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off`.

| Command | Before the edit | After the edit |
| --- | --- | --- |
| `.toolchain/bin/go build ./...` | exit 1 | exit 1 |
| `.toolchain/bin/go build ./cmd/... ./runner/... ./qualification/... ./target/... ./record/... ./artifact/... ./choice/... ./deterministicio/... ./world/... ./toolchain ./toolchain/version ./internal/...` | not run | exit 0 |
| `.toolchain/bin/go vet -tags test_dep ./target ./qualification/set ./runner/... ./deterministicio` | exit 0 | exit 0 |
| same vet with `GOOS=linux GOARCH=amd64` (cross type-check on darwin) | not run | exit 0 |
| `.toolchain/bin/go test -tags test_dep . ./target ./qualification/set ./runner/... ./deterministicio ./internal/canonicaljson` | exit 0, 14 packages ok | exit 0, 14 packages ok |
| `.toolchain/bin/go test -count=1 -v -tags test_dep -run '^(TestPackageArchitecture\|TestCanonicalJSONHasOnePrivateOwner)$' .` | not run separately | exit 0, both PASS |
| `gofmt -l target qualification/set runner deterministicio` | not run | no output |
| `make -C tools/gomad3 validate` | exit 0 (task 1 baseline) | exit 0 |
| `make -C tools/gomad3 test-host` | exit 0, 45 packages ok (task 1 baseline) | exit 0, 45 packages ok |

baseline: red (`go build ./...` failed pre-edit). `./...` matches the packages under
`toolchain/runtime/overlay`, which import Go-internal packages and build only inside the patched
GOROOT ("use of internal package … not allowed"). The failure is the same before and after this
change and is a defect of the task's Quick command, not of the code. The host package set that
`make test-host` uses builds with exit 0. Gate timestamps: [task2-gates.txt](task2-gates.txt).
`test-host` ran once, with fn-108.4's in-progress edits present, and needed no rerun.

Not run:
- linux/amd64: every gate, not run (no host). The linux vet row above is a cross type-check from
  darwin, not a linux gate.
- darwin/arm64: the other `make -C tools/gomad3 test` tiers, `world-test`, `test-harness`,
  `gomad3sim`, integration, qualification and smoke. The conductor excluded the full
  `make -C tools/gomad3 test` while fn-108.4 runs; fn-108.7 owns the final gates.

No gate receipt was written: `flowctl gate classify` reports FULL because of uncommitted
`.plans/GOMAD_MILESTONES.md` from earlier tasks, and the commands above are focused runs.

## Remaining failures

None from this change. The `go build ./...` Quick command is unusable as written in tasks fn-108.2
and later; the owner of the fn-108 task text should replace it with the host package set.
