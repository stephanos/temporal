# Publication cleanup source-progress review

Outcome is acceptable-source-progress. Adapter regeneration preserves completed publication when scratch cleanup fails and reports cleanup causes before publication. No actionable P0-P3 findings remain within the admitted source correction. This verdict supports the conductor's separate source-progress checkpoint. Original acceptance remains open.

Reviewer is Codex gpt-6.1-sol at high in a fresh context. Writer and reviewer use the same Codex family. Review scope is the fn-113.2 Publication-aware scratch cleanup source admission and its three source/test files against cefbe173514a4dde8b263462d3fc5504df340bf1. The reviewer read AGENTS.md, tools/gomad3/README.md, MILESTONES.md, the admission, the correctness criteria and the prose contract. The reviewer inspected the final handover, evidence and retained compact logs after the worker completed them.

## Source assessment

- adapterregen.go lines 153-163 and transaction.go lines 103-113 invoke removeWritable once through the original deferred boundaries. Deferred stage cleanup runs before the unchanged transaction.go line 89 lock release. Run's deferred download cleanup follows apply's return. Recover's second lock release remains unchanged at line 219.
- adapterregen.go lines 344-366 collect WalkDir callback errors and chmod errors without stopping traversal, check the WalkDir return and always attempt RemoveAll. Zero causes return nil, one cause returns the original error directly and multiple causes use errors.Join.
- Before completeJournal succeeds, ordinary errors return the original zero Result. Nil cleanup keeps exact primary identity. Multiple failures retain the primary first, followed by stage and download cleanup causes. InputError and BlockedError remain reachable through errors.As, including the unchanged command adapter's status classifier.
- transaction.go line 147 assigns a nonnil published slice only after completeJournal succeeds at lines 144-146. Later cleanup errors append warnings. Run preserves Applied, Published, Staged, residual data, earlier warnings and nil operation error. Stage cleanup precedes download cleanup in warning order.
- StageOnly returns nonnil staged information from apply. Run's lines 198-203 preserve that information when the newly exposed stage cleanup fails. Every pre-existing apply error still returns an empty publication, preserving its prior zero-result path.
- The diff changes only the two cleanup boundaries, the helper and the necessary named-result handling. Both Lock.Release calls, recovery decisions, journal retirement branches, comments, public parameter/result types, schemas, pins, generators, native guards and toolchain inputs remain unchanged.

## Public regression and evidence assessment

The tests call public Run with the existing fixture checkout and file proxy. Both are created before the private TMPDIR. Checked permission restoration is registered at publication_cleanup_test.go lines 213-227 before Run. The probe at lines 228-245 verifies actual unlink denial and explicitly skips privileged or unsupported permission behavior. The fault changes only the parent to mode 0500 after staging or residual scanning.

The tests independently check each retained named scratch root, confirm its contents were removed and attempt os.Remove at lines 266-291. This proves real root-unlink EACCES rather than substituting a cleanup error. Every public leaf checks lock reacquisition. Prepublication controls compare checkout snapshots and exact error objects. Postpublication controls verify the adapter version, pinned fixture and generated output changed on disk at lines 179-183, plus the expected eight staged/published files. These checks establish publication behavior independently of the warning assertions.

The reviewer independently ran the following command with login=false on Linux ARM64 uid 1000.

```sh
PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOWORK=off GOENV=off go -C tools/gomad3 test -tags test_dep -count=1 -v -run '^TestRunScratchCleanup' ./upgrade/adapterregen
```

Exit was 0 in 4.417 Go-test seconds. All 13 leaves passed, with 12 actual EACCES observations and zero skips. reviewer-green.log retains the complete output, SHA-256 e20109154d4a565357ba617aff34abd2ef2140df70bd50746015c230902c13f7. The conductor's separate green receipt reports the same 13/12/0 counts.

The reviewer verified all final source, unchanged fixture, baseline source and tool/config hashes against evidence.json. The five durable compact logs listed by retained_compact_logs match the recorded hashes and their ignored raw copies. RED contains six passing controls and seven failing fault leaves. Final GREEN contains 13 passing leaves. Package evidence records 35 passing leaves, 38 summaries including parents and zero skips. Scoped lint retains precisely the same two existing Lock.Release errcheck findings, with the second shifting from line 209 to 219. It removes zero configured findings. Four architecture checks, check-only validate and mandatory changed-line fast lint have passing retained output. The configured full host lint still contains 308 findings before the changed-line filter; the stock-Go ./... overlay load failure is retained as an unsuccessful observation.

## Frozen binding and acceptance limits

| File | SHA-256 |
| --- | --- |
| adapterregen.go | a29340699742fb6ce2245d32bc3864c39fcb4fc122e520bb6c96e378018f8fea |
| transaction.go | 7b1daf2868cac62ef0740c0ec63b143c49e2db8c84a34e85fb096e91dd7e65f8 |
| publication_cleanup_test.go | 82346aa1055ae812690860e2a333fd545a7cf6d5ba77065370263cefd831e5a0 |
| reviewed evidence.json | 20a68a7e34ba3fa08aada4d5aef5b78f5ad62249f64bb025437b7d6609432cc5 |
| reviewed handover.md | 052e4d1c95a04117c91b4968ba3c74a1d21c9a29a4fc3d2b2754008547145217 |

Task1's dependency and original task2 regeneration, native Darwin, full-host, default, functional, affected-consumer and formal requirements remain open wherever unproved. Full lint remains red. This stock-Go Linux ARM64 run provides portable source evidence only. Native Linux qualification remains deferred and unverified under fn-128. The reviewer made no source, index, HEAD or Flow lifecycle changes and executed no unavailable patched-runtime fixtures. The conductor retains commit and lifecycle ownership. No formal SHIP, ReadyToMerge or task completion follows from this review.
