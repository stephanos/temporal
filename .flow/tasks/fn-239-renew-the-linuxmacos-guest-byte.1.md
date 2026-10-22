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
- [ ] Four clean builds (two per OS) at one committed revision produce byte-identical stock and Temporal guests; all twelve comparisons pass.
- [ ] Both host suites pass on both OSes; the Temporal build matches the committed asset.
- [ ] Proof retained; DESIGN.md/GUEST_BUILDS.md caveat replaced by the renewed certification.

## Done summary
Blocked:
BLOCKED: needs a native macOS run on the owner's Mac; this sandbox is Linux-only. Commands are in the task description.
## Evidence
- Commits:
- Tests:
- PRs:
