# Task 29 private payload cleanup source progress

Private source and inline payload writes now observe cleanup errors while preserving successful publication and the existing primary errors when Close succeeds. The seven admitted errcheck findings are resolved. Actual unfiltered artifact lint changes from 28 findings to 21, with zero introduced diagnostics. This handover supplies developmental source progress for root review and commit. Original task and parent qualification remain open.

The source baseline is `e98c7a3e2ca845b92edcdef185f5c9ae67be4a3a` on authorized branch `gomad`. Only `tools/gomad3/artifact/store.go`, `store_test.go` and task-29 artifacts were written. Root owns independent review, Git/index/commit, parent/task tracking and MILESTONES. Worker commits and lifecycle writes are zero. Requested implementation route was `gpt-6.1-sol` at high, with root's retained `Tier: session (jev-unavailable(no_key))`. Executing-model metadata is unavailable. No routing or bridge call was repeated.

## Source correction

`copyPayload` and `writePayload` use a named error result and conditional cleanup. Each acquired handle has one cleanup defer. Source acquisition installs input cleanup first; destination acquisition installs output cleanup second, so failures close output before input. Successful operation still copies or writes, Syncs through the unchanged context owner, checks output Close, checks source Close for copies, then builds metadata. Ownership flags are cleared before each checked success Close, including an attempt that returns an error. The defer cannot retry that handle. A successful source copy therefore does not surface the former second-input-Close error.

Each defer calls `errors.Join(retErr, closeErr)` only for a nonnil close error. With nil cleanup, the existing primary return object is untouched. Actual cleanup errors follow the original operation error; simultaneous output and input cleanup failure retains operation, output, input traversal order. Error branches already return zero `record.File`, and successful metadata is constructed only after all checked success closes. The helpers retain partial destinations for their caller-owned cleanup.

The existing sixteen-site mapping remains the ownership inventory. This task changes sites 7-13 only.

| Baseline store.go line | Checked cleanup line | Owner |
| --- | --- | --- |
| 333 | 336 | copy input defer |
| 346 | 355 | copy output after Chmod failure |
| 352 | 355 | copy output after copy failure |
| 356 | 355 | copy output after Sync failure |
| 402 | 418 | inline file after Chmod failure |
| 406 | 418 | inline file after write failure |
| 410 | 418 | inline file after Sync failure |

Source checks compare every byte outside these two private helpers with BASE. All original Store test bodies remain byte-identical. Public `Opened.CopyPayload`, directory sync and shared verification are unchanged. Source/data routing, Stat acceptance, modes, exclusive creation, copy/hash/count, canonical finalization, manifest-last publication, no-replace rename, reuse, target-pool transactions and capacity accounting retain their existing owners.

## Evidence and limits

[evidence.json](evidence.json) contains all raw parsed before/after lint findings, the exact seven resolved sites and the complete residual list. The 21 residual path/message/linter identities match baseline. They comprise 17 errcheck, two exhaustive, one forbidigo and one staticcheck finding. The unchanged directory Close finding moves from line 470 to 490. Public copy's six production cleanup findings and shared verification's two remain separate owners. No lint suppression, config, rule, dependency or golden changed. Historical whole-scope 419 is not a fresh whole-scope count.

The pinned analyzer baseline is the task-authorized source-policy RED. Five real-file tests were added and passed before production edits. They catch changed copy/hash/count or returned bytes/modes, incorrect nil-close joining or second-input-close observation, changed cancellation wrapper/classification/zero metadata/partial-destination policy, overwritten collision sentinel or lost PathError shape, nonregular-source acceptance, and leaked Store staging. Their expectations use literal bytes, independently computed SHA256 literals, exact context wrappers and the standard already-cancelled context. These tests characterize real primary-error and success behavior; they do not claim runtime first-Close fault reproduction.

Baseline whole artifact package, focused Store/publication/pool/mode/capacity controls and errortype pass. Final whole artifact package, focused private/Store/publication/pool/mode/retained-byte controls, errortype, root architecture/public-signature/external Runner boundaries, `make validate` and scoped diff/gofmt checks pass. Lint exits 1 both before and after with the counts above. Every gate receipt binds command, cwd, environment, start/end, elapsed time, exit, log/tool/config hashes and stable complete artifact-source hashes. The current root suite no longer declares historical `TestRecordAndArtifactHaveSeparateOwners`; executed `TestPackageArchitecture` enforces distinct record and artifact ownership. Both existing external Runner compilation tests actually execute and pass.

The Makefile's VERSION_INPUTS, BOUNDARY_INPUTS, COMPATIBILITY_INPUTS and validation recipes were inspected before edits. Neither private helper is a generator input. `make validate` passes all current generated-input, protocol, boundary, patch, script, pack and qualification-manifest checks without regeneration. The 1,042 protected tracked files retain aggregate `40adf6923b35cc17eb5f76a189308bb506f41dfc4ab235a5d01736f2aed7f520`, matching root's preserved source-admission.json. This compact protection excludes only the two admitted source files.

Active cached Go1.27.1 `os/file_posix.go` and `os/file_unix.go` were inspected. `File.Close` delegates to the owned file close; repeated closure becomes a close PathError wrapping `os.ErrClosed`. No deterministic legitimate first-Close failure was reproduced. First-close dual-error ordering and one-attempt ownership on those rare failures are source-inspected. No fake file, public injection seam or descriptor race was added.

All commands use cached stock Go1.27.1 on linux/arm64, `GOWORK=off GOTOOLCHAIN=local GOPROXY=off GOFLAGS=''`, cleared `GOMADSEED` and `GOMAD3_CHILD_SEED`, and tests use `-count=1 -tags test_dep`. Stock development evidence does not qualify patched-runtime native darwin/arm64 or linux/amd64. Complete/full/formal, matched original first-baseline fixed identities, task 12/predecessors/task 21 and original R13/R18/R19 remain open wherever unproved. Unchanged broad rootfast/full/native failures were not rerun. Fresh independent review and commit are pending root; no formal SHIP is claimed.

Final source SHA256 is `a41297665317751bef4d62264eb8c68bdf0ea5cc3d79f234424ff8946978a7d1` for store.go and `fad48f38fe5344b4f6767321580bdf3af79c3245dd68cbc355fecfaf5e535d6d` for store_test.go. All owned execution handles are terminal. Spawned delegates, live handles and pending commands are zero.
