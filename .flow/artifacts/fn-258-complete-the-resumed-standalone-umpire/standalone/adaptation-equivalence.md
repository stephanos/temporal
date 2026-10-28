# Frozen-source equivalence after standalone adaptations

Standalone commit `05473210b48c3db862f22051eb7154ce98258f98`, tree `46829511b6d582b40f3fa5095f42bdf980c7d466`, passes comparison with the 1,189-file manifest frozen at Temporal `ac60daaf47d2513b41eeff612f0e2758d33e186d`.

There are 1,130 byte/mode-identical source files and 59 explicitly approved modifications. The [allowlist](adaptation-allowlist.json) pins a reason, provenance commits and exact approved blob/mode for every adaptation. The two additions since the previously reviewed `ac79114` tree are exactly `dev/xtask/src/gate.rs` and `dev/xtask/tests/gate_wiring.rs`: required tests supervise the existing TLC fetch before execution and propagate setup failure; the caller controls supply required policy fields and cover setup ordering, failure, skip behavior and receipts. Neither path is allowed by a wildcard or by live-head discovery.

All 54 historical adaptation claims are verified from the unchanged fn-257 inventory, its frozen c88 source manifest/Git tree, and the approved `f7b42ce25558344ea2d38f1e05d002259427a8f6` Git tree. The verifier rehashes every historical before/after blob, binds the inventory SHA-256, checks each claimed prior SHA-256, and checks both directions of the unchanged flag against current bytes and modes. Exactly 51 remain unchanged and three differ. The [machine report](adaptation-equivalence.json) records each checked historical blob, mode, digest and equality result; these counts are verified rather than copied from allowlist flags.

There are zero missing frozen-source files, unexpected modifications, mode differences, unexpected additions or protected differences. All 499 protected records remain byte/mode-identical. This migration intentionally freezes entire fixture directories and the complete `dev/fixtures` crate, including its Rust source, Cargo metadata and `build.rs`, as well as expected output, snapshots, metadata, locks, sums and LICENSE. That stricter fixture-source freeze is deliberate and has not been weakened. The source's 133 additions since c88 are present, and all 122 deleted former source paths are absent. Six standalone-only support files are pinned separately.

Thirty-seven non-generated Go/protobuf source checks pass: Go changes normalize exactly to canonical import identity and ordering; protobuf source changes normalize exactly to the go_package option. Fourteen in-memory negative controls verify missing/unexpected source, protected LICENSE/fixtures/locks, fixture-crate source and build-script changes, modified allowlisted content, missing support, unexpected additions, deleted-source resurrection, a stale prior adaptation hash, and both mismatched unchanged-flag directions. No checkout is mutated by these controls.

The [history-join proof](history-join-verification.json) preserves the historical `c71c3090fa0f89e744631c6590f1d8b4b0ffd94f` join: ordered parents `5ee8911480c8acb429625bef3f3183e572b9d18f` and `a92db0610f13c9f623b526f2aa721abda69aa84d`, with the exact first-parent tree `9324a0e75c455bd060cc1faeb328769a556d0f2d`. Historical `ac79114` remains its direct child and changes only boundary test text. The current receiver `05473210` is separately proved to be the direct child of `ac79114`, with exactly the two approved gate paths and their pinned modes/blobs; its complete binary diff is SHA-256 recorded. Original export `cf8bec4`, current filtered export `a92db061`, the join and all portable preparation ancestors remain reachable from the current receiver. The historical boundary-only claim applies to `ac79114`; the current receiver has its separately enumerated gate changes.

Reproduce from the Temporal repository root:

```sh
python3 .flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/standalone/verify-adaptation.py
python3 .flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/standalone/verify-history-join.py
```

The adaptation verifier defaults to the explicit reviewed pin; `--head <revision>` compares another tree against the same exact-content allowlist. These reports prove Git bytes, modes, ancestry and static normalization. They do not certify runtime gates, current protobuf generation freshness, guest reproduction, uncommitted worktree contents, publication, candidate transfer or source cutover. Current-head execution receipts are retained separately by the root lane.
