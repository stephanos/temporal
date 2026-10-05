---
satisfies: [R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.39 Preserve target file lifetimes while checking cleanup failures

## Description
Check the nine target cleanup findings under R18/R19 without broadening other refactors. The adapter source-set helper originated in fn-113.2 at 076cdcc344ced6e1f6e195df84540c8ca74ca2f1; this owner covers only its deferred GOPATH removal. Fn-113 retains regeneration approval, source pins, publication and qualification.

**Size:** M
**Files:** adapter_source_set.go, target.go, target_test.go and a focused real-file cleanup test file.
**Touches:** [tools/gomad3/target/adapter_source_set.go, tools/gomad3/target/target.go, tools/gomad3/target/target_test.go, tools/gomad3/target/cleanup_test.go]

### Approach

- Root admits this single source/cache writer only after the digest/import corrective candidate is integrated, independently source-reviewed and separately committed. That owner's full acceptance may remain open. Revalidate current target research bindings. Root owns Flow, index, evidence, review and commits.
- Before production edits, reproduce actual unfiltered target lint and characterize real-file hash/copy/private-write behavior on unchanged BASE. Preserve executable contents/digest/size/modes, absent source, symlink/non-executable rejection, exclusive destination collision, missing parent, exact error text/type/order and unchanged existing destination bytes. Preserve every original fixture assertion, including the mutation fixture's primary Write failure.
- Check the deferred RemoveAll, reader Close in hashRegularFile/copyRegularFile, five output cleanup Close sites, and the mutation fixture Close once at their original lifetime boundaries. Retain destination-before-source release, original validation/Chmod/copy/Write/Sync order, modes, partial-destination policy and the existing close prepared exec target: %w branch.
- Follow conditional composition at tools/gomad3/artifact/open.go:271. Nil cleanup preserves the exact primary error object and unwrap shape; sole cleanup failure returns directly; simultaneous errors retain primary-first order. A hash-reader cleanup failure clears digest/size; GOPATH removal failure clears the returned digest. Checking formerly ignored cleanup exposes a bounded additional failure surface and must be disclosed rather than described as universal byte equivalence.
- No lawful target filesystem fault seam exists in the current design. Real first-Close, simultaneous primary/cleanup, post-open Chmod/Write/Sync and RemoveAll faults remain explicit proof gaps until genuine execution evidence exists. Ordinary-file success, second-Close, descriptor theft, arbitrary callback injection or source-text assertions cannot stand in for them. A new private seam requires separate bounded admission; this task adds no public seam or filesystem framework.

### Investigation targets

**Required:**
- tools/gomad3/target/target.go:882-955
- tools/gomad3/target/adapter_source_set.go:31-72
- tools/gomad3/target/target_test.go:508-526
- tools/gomad3/artifact/open.go:271
- tools/gomad3/deterministicio/adapter_regenerate_test.go:40

### Quick commands

From tools/gomad3 with the same pinned stock-Go environment as the digest owner:

```sh
go test -count=1 -tags test_dep ./target -run 'TestTargetFileCleanup|TestPreparedCacheDigest|TestCapabilityReviewGoldenCanonicalBytes|TestCompatibilityPackProjectionPreserves'
go test -count=1 -tags test_dep ./deterministicio -run '^TestAdapterPreparedSourceSetPinsReproduceOnEveryHost$'
/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 run --config=../../.github/.golangci.yml --build-tags=test_dep --timeout=10m --fix=false ./target
```

Use only available cached adapter inputs with network disabled, or retain the precise unavailable-input result. Run errortype, relevant architecture boundaries, formatting and applicable generator checks. Retain frozen command/source/tool/config bindings, exits, elapsed times, baseline/final controls and all lint findings. Independent source review must examine cleanup order, once-only ownership, conditional composition and each unexecuted fault path.

Root commits independently reviewed source progress before another writer. Original R18/R19, task21/predecessors, matched first-baseline identities and full/default/functional/affected-consumer/formal/native Darwin requirements remain open wherever unproved. Linux-native qualification remains owned by fn-128 and cannot block this task. No production policy, Go pin, compatibility grant, public API, prior acceptance or immutable qualification evidence changes.

## Acceptance
- [ ] All nine mapped cleanup returns are checked exactly once at original lifetimes; independent inspection verifies operation ordering, destination-before-source release, existing explicit-close messages, modes and partial-destination policy.
- [ ] BASE/final real-file controls preserve literal contents/digests/size/modes and exact reachable validation/error/side-effect cases. Nil cleanup retains primary error identity; genuine cleanup failures follow the disclosed sole/direct or primary-first composition and clear derived digest/size results.
- [ ] Actual unfiltered pinned target lint reproduces and removes exactly the nine mapped cleanup findings without new diagnostics or suppressions. Focused/ordinary portable controls, errortype, boundaries, formatting and applicable generator validation pass on frozen sources; adapter-pin execution or its exact unavailable-input blocker is retained.
- [ ] A fresh independent source review finds no actionable introduced defect and explicitly lists genuine fault-execution gaps. Root commits source, tests and owned Flow evidence separately before another writer; progress review supplies no formal qualification claim.
- [ ] Required original R18/R19 and predecessor/full/default/functional/affected-consumer/formal/native Darwin acceptance is proved before completion. Task21 consumes this evidence and does not gate source admission. Missing/red source-owned acceptance stays open; transferred Linux execution is nonblocking.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
