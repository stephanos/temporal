**Scope reviewed:** task fn-258.2 (R2/R3) standalone evidence under `standalone/`, the aggregated `source-adaptation.patch` (ac79114 vs raw export a92), the two verifier scripts, and the MIGRATE/MILESTONES hunks. Consumer, disk-cleanup and validation directories were read only for cross-reference.

**What I verified**

- **Boundary exception is sound and exact.** The three admitted descriptor lines are exactly the ones where the length-prefix byte before `github.com/` is alphanumeric (`A`, `D`, `D`), which defeats the existing token-boundary organization exception in `forbidden`. The other four generated lines are preceded by `?` or `=` and pass without the exception. Masking only fires on an exact path plus line-hash match, and the masked line still goes through `forbidden`, so the negative control that re-hashes an injected line correctly stays red. The LICENSE exception is file-path plus full-content hash, and symlinks are refused upstream in `inventory`.
- **History-join split assumption holds.** The base `boundary.rs` has a single `#[cfg(test)]` at the test module, and the patch adds none before it, so the production-text comparison covers all production code.
- **Quick multi-repository selection.** Single-repository behaviour is unchanged (`options.base` is always `Some` after parse). Multi-repository mode applies BASE only to the caller's repository, uses `HEAD` elsewhere, and refuses callers outside the admitted set. Tests cover the ambiguity, invalid-base and sibling-base cases.
- **Inventory arithmetic.** 1189 = 1132 + 57; 37 normalized Go/proto checks = 57 minus the 20 docs, policy, lock-pinned generated and `go.mod` paths; 122 removals and 133 additions reconcile with the parser-refusal relocation plus eight new gate sources. Descriptor length prefixes in the patch (`BA/Z?`, `BC/ZA`, `B?/Z=`, `BF/ZD`) are consistent with real protoc output rather than hand edits.

## Findings

- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/standalone/adaptation-equivalence.md:9`
- **R-IDs**: []
- **Problem**: Task acceptance item 5 requires "protobuf freshness pass", but no receipt bound to `ac79114` exists under `standalone/` for `scripts/proto-go.sh core --check` / `kettle --check` or the `protobuf`/`source-boundary` gate stages. The equivalence prose explicitly disclaims current regeneration freshness, and the generated blobs are only pinned to fn-257's earlier approved SHA-256. The focus brief asserts these controls are green, but the committed evidence does not show it.
- **Suggestion**: Before closing task 2, commit a receipt of `bash scripts/proto-go.sh core --check && bash scripts/proto-go.sh kettle --check` (or `cargo xtask gate --stage protobuf --stage source-boundary`) run at `ac79114`, or record in the done summary that this item is deferred to task 4's fresh gate.

- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/standalone/source-adaptation.patch` (hunk for `dev/xtask/src/gate/boundary.rs`, `check`)
- **R-IDs**: []
- **Problem**: The `generated` header and `descriptor` run-state tracking is redundant with the exact path plus line-hash admission in `descriptor_line`; a line can only be masked if its full bytes and path match the policy, so the context machine never changes the outcome. It is small and tested, so this is FYI only.
- **Suggestion**: Optional simplification: drop the state machine and call `descriptor_line` on every line of `.pb.go` files; keep the existing tests.

**FYI, outside task 2 scope:** `MIGRATE.md` and the MILESTONES row name consumer candidate `4035bb70`, while the guest reproduction and local-edit proofs in this range are bound to consumer `7e655da8`. The consumer checkpoint shows `4035bb70` as the branch tip, so this is likely the bundle commit on top, but task 3's closure should state the relationship explicitly.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R2 | met | `history-join-verification.json`: ordered parents `5ee8911` + `a92db06`, join tree equals first-parent tree, all 12 ancestry commits reachable from join and final, export selected records equal frozen manifest; original `cf8bec4` export retained. Commit map coverage is task 1's closed evidence, referenced not re-derived. |
| R3 | met | Framework-only checkout with root Make, AGENTS/DESIGN/GUEST_BUILDS, consumer-interface doc and CI; canonical `github.com/temporalio/umpire/{hosts/go,reference/kettle}` identities in `go.mod`, imports, `go_package` and regenerated descriptors; 499 protected payload/snapshot/lock/LICENSE records exact; boundary controls retained and extended with planted refusals. Current-pair regeneration freshness and full gate are deferred to task 4 per the spec's interim-ownership section. |

Unaddressed R-IDs: []

Suppressed findings: 1
Classification counts: 2 introduced, 0 pre_existing.

```json
{"suppressed_count":{"25":1},"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>