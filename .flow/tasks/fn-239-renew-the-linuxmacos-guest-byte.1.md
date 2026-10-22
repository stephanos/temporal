---
satisfies: [R1, R2]
---
# fn-239-renew-the-linuxmacos-guest-byte.1 Run and record the renewed Linux/macOS guest certification

## Description
fn-238.8 added a per-crate lockfile closure digest to every guest crate identity, so fn-232's certification at 3c314fb5b8 no longer covers how identities are derived. Renewal needs a native macOS run on the owner's Mac (the original ran Linux via `sbx exec dev-build` and macOS natively on the host; this sandbox has no macOS executor).

### Approach

Reuse `.flow/artifacts/fn-232-linux-macos-guest-byte-identity/certification-3c314fb5b8/` (`run-proof.sh`, `compare.py`) against a committed revision at or after the fn-238.8 merge: two clean detached clones per OS with independent initially absent targets, then the twelve cmp comparisons and both host suites. Cheap partial check first: compare `umpire-lock-<hash>.json` closure-map output between Linux and macOS for the same revision.

Exact invocations from the repository root, with `<rev>` the pinned commit:

```sh
sbx exec -w "$PWD" dev-build bash -c 'bash .flow/artifacts/fn-232-linux-macos-guest-byte-identity/certification-3c314fb5b8/run-proof.sh "$PWD" "$PWD/.flow/tmp/fn2389-linux-<rev>"'
bash .flow/artifacts/fn-232-linux-macos-guest-byte-identity/certification-3c314fb5b8/run-proof.sh "$PWD" "$PWD/.flow/tmp/fn2389-macos-<rev>"   # on macOS
python3 .flow/artifacts/fn-232-linux-macos-guest-byte-identity/certification-3c314fb5b8/compare.py "$PWD"
```

Retain proof, manifests and comparisons under `.flow/artifacts/fn-239-renew-the-linuxmacos-guest-byte/certification-<rev>/`.

## Acceptance
- [x] Four clean builds (two per OS) at one committed revision produce byte-identical stock and Temporal guests; all twelve comparisons pass.
- [x] Both host suites pass on both OSes; the Temporal build matches the committed asset.
- [x] Proof retained; DESIGN.md/GUEST_BUILDS.md caveat replaced by the renewed certification.

## Done summary
Renewed Linux/macOS guest byte-identity certification at committed source `a96a4c6550b468c5d7c3011a7961d4e118dacd93`. Four clean detached clones with independent initially absent targets produced identical stock (192489 bytes) and Temporal (3332314 bytes) guests. All twelve actual comparisons passed. Both stock/control and embedded Temporal host suites passed on Linux and native macOS; Temporal matches the committed asset. The precheck and all four build-time lockfile closure maps match. `make model-check` passed in the pinned Linux clone; default mode skips real TLC execution. Final source checks, twelve comparisons and gate-output comparisons passed after the gate. Proof, scripts, logs, manifests, closure-map checks and checksums are retained under `.flow/artifacts/fn-239-renew-the-linuxmacos-guest-byte/certification-a96a4c6550/`. Updated DESIGN.md, GUEST_BUILDS.md and MILESTONES.md. Changes remain uncommitted as instructed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: bash .flow/artifacts/fn-239-renew-the-linuxmacos-guest-byte/certification-a96a4c6550/run-proof.sh "$PWD" "$PWD/.flow/tmp/fn239-certified-macos-a96a4c6550", sbx exec -w "$PWD" dev-build bash -c 'bash .flow/artifacts/fn-239-renew-the-linuxmacos-guest-byte/certification-a96a4c6550/run-proof.sh "$PWD" "$PWD/.flow/tmp/fn239-certified-linux-a96a4c6550"', python3 .flow/artifacts/fn-239-renew-the-linuxmacos-guest-byte/certification-a96a4c6550/compare.py "$PWD", sbx exec -w "$PWD/.flow/tmp/fn239-certified-linux-a96a4c6550/source-a" -e CARGO_BUILD_JOBS=2 -e CARGO_TARGET_DIR="$PWD/.flow/tmp/fn239-certified-linux-a96a4c6550/target-a" dev-build make model-check, git diff --check
- PRs: