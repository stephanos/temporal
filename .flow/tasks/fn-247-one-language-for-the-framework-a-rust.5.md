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

- Map useful guest/freshness controls to native tests before deleting sources; explicitly retire old Python-only quirks and local/CI missing-tool skips under the owner amendment. Record the predecessor guest hash as historical reference, not an old-byte acceptance oracle.
- Replace all three guest.sh Python heredocs with internal xtask responsibilities: cache hash (in-process `sha2`), atomic launcher creation and Cargo JSON artifact selection/build/check. Keep useful ordinary shell invocations, caller-relative output paths, custom cdylib names, zero/multiple/malformed artifact refusal and never-write check mode; document native argument/output/error changes and update callers without exact old diagnostic proofs.
- Bootstrap the native tool before guest `RUSTC_WRAPPER`, remap/encoded flags and guest-cache target are installed. The launcher executes the exact built binary by an absolute path and preserves macOS dynamic-library variables; it never uses `/usr/bin/env` or recursively invokes Cargo. Cover whitespace checkout/tool paths and custom target directories according to the old caller behavior.
- Put the actual Make freshness entrypoint through minimal `guest-freshness.sh` bootstrap. Export `MISE_AUTO_INSTALL=false` and `MISE_EXEC_AUTO_INSTALL=false` before the first probe. Document Rust/Cargo/mise as prerequisites; missing bootstrap tools and bootstrap compiler failures fail clearly, regardless of CI. Build/execute native xtask before guest wrapper/flags; Rust owns native TOML/JSON/installed-tool metadata validation, pin consistency and Rust/rustup/Go/protoc/rust-src/wasm readiness. Do not duplicate semantic checks in Bash or preserve the old local/CI warning/skip matrix. Cached xtask must not bypass native validation.
- Test the actual cold Make/shell entrypoint with no cached xtask: absent bootstrap tools, bootstrap compiler failure, runtime missing tools, malformed metadata/pins and probe errors fail clearly; auto-install is disabled from the first probe; failures occur before guest build/publication and check mode never modifies the asset. Exercise cached-path native validation too. CI absent/empty/nonempty no longer changes helper missing-tool failure; R5 backend default/skip semantics remain separate.
- Switch callers, then remove guest helper Python/tests after useful native controls pass. Regenerate committed guest/goldens/pins when necessary for changed inputs; document cause and old/new hashes, retain semantic expectations and pass Go host tests. Prove current-input reproducibility with two clean builds (fresh cache, different checkout path) and fresh check mode against the committed asset. Prefer one integrated regeneration; repeat only for changed inputs or a specific failed proof. Use disposable output for additional build/check controls, not repeated full gates.

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
- [ ] Useful guest/freshness responsibilities have Rust controls and explicitly retired Python-only quirks; fake-tool build/check/status/path and native malformed-input controls pass.
- [ ] No embedded Python remains in guest.sh; cold xtask bootstrap, absolute launcher, previous-wrapper/jobserver inheritance and macOS environment controls pass without recursive Cargo.
- [ ] Artifact discovery rejects malformed/multiple/missing wasm candidates and reports stale/native failures with documented statuses and useful diagnostics; check mode never modifies output.
- [ ] Rebuilt current-input guest is reproducible (two clean builds agree) and check mode reports fresh against the committed asset; necessary guest/golden/pin regenerations document cause and old/new hashes, preserve semantic/negative expectations and pass Go host tests. No old/new byte-parity proof is required.

- [ ] Actual cold Make/shell entrypoint, with no cached xtask, disables auto-install before its first probe and fails clearly for missing bootstrap tools/compiler failure without guest build/publication. Rust validates runtime prerequisites/metadata/pins; no Bash semantic-parser duplication or local/CI missing-tool skip matrix remains. Cached paths also validate; check mode never publishes.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
