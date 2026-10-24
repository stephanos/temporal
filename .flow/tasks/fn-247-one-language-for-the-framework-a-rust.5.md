---
satisfies: [R2, R3]
---
# fn-247-one-language-for-the-framework-a-rust.5 Replace embedded guest Python and freshness policy with native xtask orchestration

## Description
Port guest freshness and embedded shell Python to xtask (R2, R3), consuming existing .2's identity implementation. This serial owner switches the guest callers and proves the rebuilt post-predecessor artifact.

**Size:** M
**Files:** `model/dev/xtask/{src/guest*,tests/guest*}`, `model/scripts/{guest.sh,guest-freshness.py,guest-lock.py,guest-rustc.py,tests/test_guest.py,tests/test_guest_freshness.py,tests/test_guest_lock.py,tests/test_guest_rustc.py}`, `Makefile`.
**Touches:** [model/dev/xtask/**, model/scripts/guest.sh, model/scripts/guest-freshness.py, model/scripts/guest-lock.py, model/scripts/guest-rustc.py, model/scripts/tests/test_guest*.py, Makefile]

### Approach

- Record the old guest/freshness corpus and complete the missing-counterpart ledger before deleting tests. Retain the exact post-fn-237.3 guest separately; capture hashes and bytes, not only git status or an earlier guest identity.
- Replace all three guest.sh Python heredocs with internal xtask responsibilities: cache hash, atomic launcher creation and Cargo JSON artifact selection/build/check. Preserve shell arguments, caller-relative output paths, custom cdylib names, zero/multiple/malformed artifact failure, first-difference diagnostics and never-write check mode.
- Bootstrap the native tool before guest `RUSTC_WRAPPER`, remap/encoded flags and guest-cache target are installed. The launcher executes the exact built binary by an absolute path and preserves macOS dynamic-library variables; it never uses `/usr/bin/env` or recursively invokes Cargo. Cover whitespace checkout/tool paths and custom target directories according to the old caller behavior.
- Port `guest-freshness` with full approved TOML parsing and exact local/CI distinction, installed-tool metadata validation, rustup shim/toolchain/rust-src/wasm probes and disabled auto-install. Wire the Makefile freshness entrypoint to native xtask; compiler failure and stale/missing bytes keep their statuses.
- Switch callers, then remove the guest helper Python and tests after differential controls pass. Compare actual rebuilt guest bytes with the retained baseline via check mode. Source-derived cache IDs may change; the artifact cannot. Use disposable output for `make umpire-guest` behavior proof and retain the committed asset unchanged.

### Investigation targets

**Required**:
- `model/scripts/guest.sh:23` — embedded hashing/launcher/artifact selection.
- `model/scripts/guest-freshness.py:19` — preflight policy.
- `model/scripts/tests/test_guest.py:15` — fake-tool shell controls.
- `model/scripts/tests/test_guest_freshness.py:85` — local/CI/failure controls.
- `Makefile:1278` — freshness caller.
**Optional**:
- `model/GUEST_BUILDS.md:24` — compiler-input and reproduction contract.

### Quick commands

```bash
cd model && mise exec -- cargo test -p xtask --test guest_pipeline --locked
bash -n model/scripts/guest.sh model/scripts/tests/build-guests.sh
bash model/scripts/guest.sh umpire-temporal-guest model/temporal/hosts/go/umpire_temporal_guest.wasm --check
```

Run from the documented root for each command; the named integration target is introduced here. Cold host bootstrap and actual guest comparison use the conductor's isolated target, serialized with other Cargo work.

## Acceptance
- [ ] All guest/freshness Python cases have Rust counterparts; fake-tool build/check/status/path controls and empty-versus-absent CI controls pass.
- [ ] No embedded Python remains in guest.sh; cold xtask bootstrap, absolute launcher, previous-wrapper/jobserver inheritance and macOS environment controls pass without recursive Cargo.
- [ ] Artifact discovery rejects malformed/multiple/missing wasm candidates and preserves exact stale diagnostics/statuses; check mode never modifies output.
- [ ] Actual rebuilt guest compares byte-for-byte with the retained post-.3 baseline and committed asset; no pin/golden/hash refresh occurs. Any unavailable native macOS comparison is explicitly reported.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
