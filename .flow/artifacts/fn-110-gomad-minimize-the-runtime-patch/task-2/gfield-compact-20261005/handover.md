# Verified source checkpoint for fn-110.2

The private runtime `g` field compaction saves 854 bytes in genuine canonical U3 output and 777 bytes in U1. U3 is 33,294 bytes, still 642 bytes above the unchanged original 32,652-byte comparator. R8 and native qualification remain open. This handover records bounded source progress only.

Task `fn-110-gomad-minimize-the-runtime-patch.2` remains `in_progress` at worker return. BASE and HEAD are `1b0bc277589d141aca8b534b03135ab3e57fc050`; the worker made no commits or Flow changes. The conductor owns review, staging, commits and lifecycle decisions.

Tier: session (jev-unavailable(no_key)); explicit project implementer routing gpt-6.1-sol/high.

stage: impl-review - skipped(policy: host-deferred - conductor owns the gate)

## Measured change

| Representation | BASE bytes / lines | Final bytes / lines | Saving |
| --- | --- | --- | --- |
| Canonical U1 | 24,894 / 706 | 24,117 / 692 | 777 bytes / 14 lines |
| Canonical U3 | 34,148 / 1,040 | 33,294 / 1,026 | 854 bytes / 14 lines |

Exactly five private field names changed at 30 declaration/selector sites. `runtime2.go` contains five declarations, `proc.go` contains three selectors, and `gomad.go` contains 22 selectors. The function/linkname `gomadSimulationDomain()` and global `gomadSimulationTransportSyscalls` keep their names. Complete alpha-renaming plus pinned-gofmt equality holds for all 20 patched source files and the overlay. Field order, types and surrounding layout remain equal under that normalization. Exact patch and overlay allowlists remain 20 and 79 paths.

The seven admitted product paths are the patch, runtime overlay, four generated identity mirrors and choices-only diagnostic identity fixture. No shipped tests, profiles, schemas, descriptors, policies or tapes changed. The fixture equals independently derived canonical bytes, changes exactly seven admitted identity pointers, and retains its 13,271 bytes without a final newline. The native guard and whole-byte assertions remain unchanged. See [preservation.json](preservation.json), [closure.json](closure.json) and [independent identity audit](identity-audit/audit.md).

## Verification

The TDD skill kept the measurement additive and task-local. The pre-edit alignment check failed for exactly seven otherwise unchanged `g` fields. The stable final focused suite passes 21 top-level tests, including the zero-alignment measurement, original pinned regeneration/checksum/rejection tests, zero-fuzz U1/U3 source equivalence, inventories and original negative controls. The code-style skill directed reuse of existing materialization/regeneration helpers and canonical mirrors.

Both supported source sets retain 273 draw, 86 seeded-draw, 48 clock and 10 goroutine rows. Fresh final materialization matches the edited candidate after complete alpha/gofmt normalization. [final-focused-frozen.json](final-focused-frozen.json) and its stdout retain actual exits and counts. [final-generate-validate-frozen.json](final-generate-validate-frozen.json) records stable final `make -C tools/gomad3 generate validate`, exit 0.

The affected pure-host selection passes eight packages with 322 pass events and eight skips, matching the preceding candidate's outcomes. Its preceding source map equals all 5,076 BASE product hashes. Existing architecture and both host-vet source-set subtests passed. [final-pure-host-observations-data.json](final-pure-host-observations-data.json) retains compact comparisons; bulk JSON output stays in ignored scratch.

Explicit host-only vet passed. An earlier recursive `./toolchain/...` command selected overlay standard-library packages in the wrong host-module surface and failed before vet analysis. [final-scoped-vet.json](final-scoped-vet.json) remains inconclusive; [final-scoped-vet-host-only.json](final-scoped-vet-host-only.json) records the corrected host selection. Overlay/runtime native vet remains unavailable.

Documented changed-only `make lint-code-fast` passed across 55 host packages using native golangci-lint 2.13.0 and errortype 0.0.7, BASE `1b0bc27758` and fixes disabled. Its raw stderr records `diff: 317/0`. All 317 preexisting full-lint issues remain open. This is no full-lint pass.

All stable final checks use stock Go 1.27.1 on Linux/aarch64 with network-disabled Go resolution, explicit environment whitelist and unchanged tool hashes. No instrumented toolchain build, downloads or native qualification ran. Three diagnostic controls still fail for the absent instrumented `.toolchain/bin/go`; the unchanged Darwin identity guard skips. [final-diagnostic-controls-observations.json](final-diagnostic-controls-observations.json) binds their exact prior failure outputs and skips to BASE.

## Open gates and evidence boundaries

The preceding full host timeout and 106 inherited deterministicio failures remain open; the unchanged unsupported-host hang was not retried. The [prior checkpoint](../../../fn-112-gomad-determinism-assurance-and-test/task-5/host-draw-field-compact-20261005/) retains its immutable receipts. Source-progress checks do not qualify Darwin runtime execution, deterministic diagnostics, stock/patched behavior, soak or Linux/amd64. fn-128 owns Linux qualification. The original patch-size target remains unmet.

The first post-edit alignment/inventory command overlapped the authorized golden publication, and its receipt honestly records that single fixture change. The stable `final-focused-frozen` run supersedes it. A report-name collision originally retained the pure-host comparison stdout but replaced its data JSON with the capture receipt; the helper now rejects existing destinations and the read-only report was regenerated from the original raw log under a distinct name. No suite reran for this repair. The first closure helper counted renamed domain selectors as retained function occurrences; the failed helper receipt remains, and corrected scoped counting passes without product changes.

Full canonical patches, 21 source snapshots and 5,076-file source maps stay under ignored `tools/gomad3/.toolchain/fn-110/gfield-compact.gxrPVUi5/`. Task-local receipts retain compact counts, digests, helper reproduction and meaningful raw output. Both unrelated user `.turbo` files retain their hashes. No command handles remain live, and the worker has released the Go/cache lane.
