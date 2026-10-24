---
satisfies: [R2, R3]
---
# fn-247-one-language-for-the-framework-a-rust.7 Port retired-name filtering and immutable original-artifact verification to Rust

## Description
Port the retired-name stream filter and the independent original-artifact verifier (R2, R3). These bounded remaining consumers share the established native xtask/hash test support; no feature behavior changes.

**Size:** M
**Files:** `model/dev/xtask/{src/retired_names*,tests/retired_names*}`, `model/scripts/{retired-name-matches.py,retired-names.sh,tests/test_retired_names.py}`, `model/temporal/feature/activity/compatibility/{checkpoint.rs,verify.py,README.md}`.
**Touches:** [model/dev/xtask/**, model/scripts/retired-name-matches.py, model/scripts/retired-names.sh, model/scripts/tests/test_retired_names.py, model/temporal/feature/activity/compatibility/checkpoint.rs, model/temporal/feature/activity/compatibility/verify.py, model/temporal/feature/activity/compatibility/README.md]

### Approach

- Record the old filter/verifier corpus and preserve their source/fixture authorities before removal. `compatibility/original/**`, its manifest and authoritative `.flow/artifacts/.../original-artifacts.sha256` are read-only inputs.
- Port NUL-separated raw path/line-number/line bytes, external grep ERE recheck, author-facade test-token exemption and original raw diagnostic output. Preserve malformed-stream status2, forbidden-match1, no-match0 and grep failure propagation. Switch retired-names.sh only at its helper call.
- Fold verify.py's exact checks into the existing Rust checkpoint: pinned original commit, exactly15 entries, each manifest checksum equals independent authority, each retained original hashes identically, same success statement. Use shared approved hashing without making the generic framework depend on Temporal.
- Add disposable-copy negative controls for changed/missing original, wrong commit, count/authority mismatch and malformed authority/manifest. Test data may be corrupted only in temporary copies; do not alter original artifacts to prove failure. Keep the existing decoder/fixture comparisons and failure controls.
- Port all retired-name unittest cases, remove the two Python consumers and corresponding test only after parity, and update the compatibility README to identify the current Rust verifier while retaining historical evidence.

### Investigation targets

**Required**:
- `model/scripts/retired-name-matches.py:11` — raw stream/status contract.
- `model/scripts/retired-names.sh:14` — shell caller.
- `model/scripts/tests/test_retired_names.py:15` — historical exclusions/active spellings.
- `model/temporal/feature/activity/compatibility/verify.py:6` — immutable15-original authority.
- `model/temporal/feature/activity/compatibility/checkpoint.rs:84` — verifier/decoder caller.
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
- [ ] Filter corpus preserves raw bytes, exemptions, ERE behavior and 0/1/2/grep-error statuses; all old cases have named Rust counterparts.
- [ ] Rust verifier retains all independent authority checks and success text; each disposable corruption/absence/authority/count control fails.
- [ ] Original decoder/fixture controls remain and pass against unchanged originals, manifest, authority and post-.3 feature behavior.
- [ ] Both Python consumers and retired-name Python tests are removed after proof; shell and checkpoint invoke Rust and compatibility documentation matches.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
