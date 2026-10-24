---
satisfies: [R2, R3]
---
# fn-247-one-language-for-the-framework-a-rust.7 Port retired-name filtering to Rust and retire the original-artifact hash verifier

## Description
Port the retired-name stream filter and retire the byte-only original-artifact hash verifier (R2, R3). `verify.py` only proved checksum agreement of retained originals, which the owner dropped (2026-10-24); it is removed, not ported. No feature behavior changes.

**Size:** M
**Files:** `model/dev/xtask/{src/retired_names*,tests/retired_names*}`, `model/scripts/{retired-name-matches.py,retired-names.sh,tests/test_retired_names.py}`, `model/temporal/feature/activity/compatibility/{checkpoint.rs,verify.py,manifest.json,README.md}`.
**Touches:** [model/dev/xtask/**, model/scripts/retired-name-matches.py, model/scripts/retired-names.sh, model/scripts/tests/test_retired_names.py, model/temporal/feature/activity/compatibility/checkpoint.rs, model/temporal/feature/activity/compatibility/verify.py, model/temporal/feature/activity/compatibility/manifest.json, model/temporal/feature/activity/compatibility/README.md]

### Approach

- Record useful old filter controls before removal. `compatibility/original/**` and historical `.flow/artifacts/.../original-artifacts.sha256` remain unmodified historical evidence; the active `manifest.json` is deleted with the retired hash verifier, not consulted as an acceptance gate.
- Port NUL-separated path/line-number/line handling, the external `grep -E` recheck (kept: Rust `regex` syntax differs from POSIX ERE), author-facade test-token exemption and diagnostic output. Preserve malformed-stream status2, forbidden-match1, no-match0 and grep failure propagation. Switch retired-names.sh only at its helper call.
- Remove the `python3 verify.py` call from `checkpoint.rs`, delete `verify.py` and `manifest.json`; do not port their commit/count/checksum assertions (owner, 2026-10-24: the hash integrity check is thrown away). Keep the semantic lifecycle comparison against `original/rows.snap.txt` and the changed-edge mutation control unchanged.
- Port useful retired-name control responsibilities (record any retired Python-only quirks with owner provenance), remove the Python filter and its test after the Rust counterparts pass, and update the compatibility README to drop the `verify.py`, `manifest.json` and checksum-verification statements while retaining the originals as historical evidence.

### Investigation targets

**Required**:
- `model/scripts/retired-name-matches.py:11` — raw stream/status contract.
- `model/scripts/retired-names.sh:14` — shell caller.
- `model/scripts/tests/test_retired_names.py:15` — historical exclusions/active spellings.
- `model/temporal/feature/activity/compatibility/checkpoint.rs:149` — `python3 verify.py` call to remove.
**Optional**:
- `model/temporal/feature/activity/compatibility/README.md:6` — current verification instructions.

### Quick commands

```bash
cd model && mise exec -- cargo test -p xtask --test retired_names --locked
cd model && mise exec -- cargo test -p umpire-temporal-tests compatibility --locked
bash model/scripts/retired-names.sh
```

Reanchor the actual compatibility test selector to the post-.3 source before running; preserve its full decoder/fixture regression scope.

## Acceptance
- [ ] Filter corpus preserves matches, exemptions, `grep -E` ERE behavior and 0/1/2/grep-error statuses; useful old controls have named Rust counterparts; exact old diagnostic bytes are not required.
- [ ] `verify.py`, `manifest.json` and the Python verifier call are gone; the lifecycle comparison and changed-edge mutation control remain and pass against the unchanged originals.
- [ ] Both Python files and retired-name Python tests are removed; the shell invokes Rust and the compatibility README matches.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
