---
satisfies: [R2, R3]
---
# fn-247-one-language-for-the-framework-a-rust.5 Replace embedded guest Python and freshness policy with native xtask orchestration

## Description
Port guest freshness and embedded shell Python to xtask (R2, R3), consuming existing .2's identity implementation. This serial owner switches the guest callers and proves the rebuilt guest reproducible.

**Size:** M
**Files:** `model/dev/xtask/{src/guest*,tests/guest*}`, `model/scripts/{guest.sh,guest-freshness.py,guest-lock.py,guest-rustc.py,tests/test_guest.py,tests/test_guest_freshness.py,tests/test_guest_lock.py,tests/test_guest_rustc.py}`, `model/scripts/guest-freshness.sh`, `Makefile`.
**Touches:** [model/dev/xtask/**, model/scripts/guest.sh, model/scripts/guest-freshness.sh, model/scripts/guest-freshness.py, model/scripts/guest-lock.py, model/scripts/guest-rustc.py, model/scripts/tests/test_guest*.py, Makefile, model/temporal/hosts/go/umpire_temporal_guest.wasm]

### Approach

- Record the old guest/freshness corpus and complete the missing-counterpart ledger before deleting tests. Record the post-fn-237.3 guest hash as reference evidence.
- Replace all three guest.sh Python heredocs with internal xtask responsibilities: cache hash (in-process `sha2`), atomic launcher creation and Cargo JSON artifact selection/build/check. Preserve shell arguments, caller-relative output paths, custom cdylib names, zero/multiple/malformed artifact failure, first-difference diagnostics and never-write check mode.
- Bootstrap the native tool before guest `RUSTC_WRAPPER`, remap/encoded flags and guest-cache target are installed. The launcher executes the exact built binary by an absolute path and preserves macOS dynamic-library variables; it never uses `/usr/bin/env` or recursively invokes Cargo. Cover whitespace checkout/tool paths and custom target directories according to the old caller behavior.
- Put the actual Make freshness entrypoint through a small Bash admission launcher (`guest-freshness.sh`) BEFORE xtask lookup/bootstrap, exporting both auto-install=false settings before its first command. Use existing mise configuration queries (including both TOML pin files) and requested/active/installed tool queries; no new parser/tool dependency. Positively identify missing mise/pinned Rust/Cargo/rustup toolchain/Go/protoc/rust-src/wasm target before Cargo starts. Missing prerequisites skip with the existing warning/0 only for absent CI; empty/nonempty CI fail/1. Nonzero probe status alone is never proof of absence: probe/config/metadata/pin errors always fail/1. Strictly validate query responses and pin consistency, then bootstrap xtask; bootstrap build failure is always failure. Retain complete approved TOML/installed JSON metadata checks in xtask after bootstrap, revalidating before byte comparison. An existing xtask cache must not bypass entry admission.
- Test the actual Make/shell freshness entrypoint with no cached xtask and each old missing prerequisite separately, both CI absent and CI empty/nonempty. Assert exact diagnostics/status, disabled auto-install even during pre-bootstrap probes, no Cargo/guest build/publication on skips, unchanged guest, and failure for probe/malformed metadata/pin/build errors. A test that directly invokes an already-built xtask does not discharge this cold contract.
- Switch callers, then remove the guest helper Python and tests after the Rust controls pass. If the Rust identity encoding changes guest bytes, regenerate the committed guest and record old/new hashes; Go host tests must pass with it. Prove two clean rebuilds (fresh cache, different checkout path) are byte-identical and check mode then reports fresh. Use disposable output for further `make umpire-guest` behavior proof.

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
- [ ] Rebuilt guest is reproducible (two clean builds byte-identical) and check mode reports fresh against the committed asset; each regeneration records old/new hashes and passes Go host tests; no golden refresh occurs.

- [ ] Actual cold Make/shell freshness entrypoint has no cached xtask: each missing bootstrap/runtime prerequisite skips only with CI absent, fails with empty/nonempty CI, disables auto-install before every probe, never calls Cargo on a skip, and leaves the asset untouched. Probe/config/metadata/pin/build failures remain failures; warm-binary-only tests do not count.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
