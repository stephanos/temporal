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

- Implement independently of .5/.6 against completed .2 in an isolated checkout/index/target. The stdin filter and compatibility-checkpoint retirement require no guest bootstrap or installer output. Wire the filter in this lane's dispatcher/lib/shell copy; conductor merges overlaps centrally and checks the joined caller plus lifecycle/mutation controls before task completion. Lane-local evidence does not certify an unmerged combined tool.
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


- [ ] The conductor has joined .5/.6/.7 dispatcher/caller edits and verified the filter and semantic lifecycle/mutation controls against that integrated source before task completion; no installer or guest whole-task prerequisite is required.

## Done summary
Ported the raw retired-name stream filter to native xtask, preserved external POSIX ERE matching/exemptions/status contracts, and removed the checksum-only verifier/manifest/Python controls. Retained original lifecycle snapshots are unchanged; both lifecycle comparison and changed-edge mutation controls pass. Joined dispatcher/bootstrap/caller integration is proven by integration/joined-validation.json (81 suite tests,2 lifecycle controls, Clippy0 and actual cold-bootstrap scan0).

Baseline was green; lane red/green and six original filter controls are retained under task7. Primary three codex:gpt-6.1-sol:high draws returned SHIP/NEEDS_WORK/SHIP; the contracts draw found one introduced pipeline status defect. The large-input real-shell control first failed141 instead of2. Fixc0a76a4749 changes only retired-names.sh and its test: an induced upstream SIGPIPE yields to a downstream failure, while genuine upstream errors retain precedence. All7 filter controls, scoped Clippy, shell syntax and actual scan pass. Same primary-session re-review returned SHIP; reviewers also ran direct existing-binary/status/lifecycle controls where possible, but Cargo rebuilding was blocked by read-only locks. Same-family provenance is recorded honestly.

The two changed files are not guest/build/compiler/identity inputs; conductor explicitly approved their scope. guest-proof-continuity.json verifies that all other Model/build inputs match523d131 except the already proven canonical publication, whose SHA remainsbc27af7d5b6b1ed8fa5c524015046f3a8b7c12d2d488e9ce7cb3df3ea9217872. No guest proof or full Model gate was repeated. Verify classified FULL/unmatched conductor-owned MILESTONES.md; this task defines focused checks only, no fabricated full-suite receipt or skip line. Final3 owns the single semantic required pipeline before parent closure.

Durable audit/control mappings, raw failures/passes, merged review findings and receipt are under task7/ and task7/review/. The review uses the exact744128984f534ef7b081e78e07cd539e2ca8c7a7 integration base; it honestly includes joined tasks5/6 and root bookkeeping, not an artificial task-only tree. Original source checkpoint e4899e270f remains the task7 implementation. Nontrivial status-propagation memory was captured. No MILESTONES/guest/build/Cargo/bootstrap source or remote operation changed in the fix.

stage: impl-review - ran (first-round NEEDS_WORK with1 finding; committed red-to-green shell fix; same-session SHIP; task7/review/receipt.json)
stage: memory-capture - ran (preserve-filter-errors-when-upstream-2026-10-25)
Tier: session (jev-unavailable(no_key))

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e4899e270f052ff24e864fe4617e18911fb348de, 813f1dde1456fc8e8808687a6af411001188ca71, 535f6bd90ea83cadda452262aab93d2b790e0466, c21571b8acf881892309ffc6cb63b56ba716ffab, 17bce75e1cbd28bb7e6939a869ee961c0df784a7, 1bff716b76455c357c3db15a981aa75fe71e9939, 523d131d2cf992b04a43b4d595ea0f1a434f44b6, fb9b9ef7f142f8d1ae5f404be018c8888eef0f95, 6c4e7f39ffd285a679fc1172d5f72dc4cd006709, a7071670854876c1949b92582fc77c295d2af2a2, 8cbe8ee453025d0a494586ecd122af3ba6f11525, c0a76a47490d6ec2d6b3ac447caf54c08e7b4fb4
- Tests: baseline: green; python3 -m unittest discover -s model/scripts/tests -p test_retired_names.py (exit 0), baseline: cd model && mise exec -- cargo test -p xtask --locked (exit 0), baseline: cd model && mise exec -- cargo test -p umpire-temporal-tests compatibility --locked (exit 0; 2 fixture tests), red: cd model && mise exec -- cargo test -p xtask --test retired_names --locked (exit 101; 3 intended absent-command failures), cd model && mise exec -- cargo test -p xtask --test retired_names --locked (exit 0; 6 tests), cd model && mise exec -- cargo test -p umpire-temporal-tests compatibility --locked (exit 0; 2 fixture tests), cd model && mise exec -- cargo clippy -p xtask --all-targets --locked -- -D warnings (exit 0), cd model && mise exec -- cargo fmt -p xtask (exit 0), UMPIRE_XTASK=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2477-target/debug/xtask bash model/scripts/retired-names.sh (exit 0), git diff --check (exit 0), git diff --name-only -- model/temporal/feature/activity/compatibility/original .flow/artifacts (empty; originals unchanged), cd model && mise exec -- cargo test -p xtask -p umpire-apalache --locked, cd model && mise exec -- cargo clippy -p xtask -p umpire-apalache --all-targets --locked -- -D warnings, bash model/scripts/retired-names.sh, cd model && mise exec -- cargo test -p umpire-temporal-tests compatibility --locked, cd model && BASH_ENV=/dev/null CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2477-target MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo test -p xtask --test retired_names --locked (exit0,7 tests), cd model && BASH_ENV=/dev/null CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2477-target MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo clippy -p xtask --all-targets --locked -- -D warnings (exit0), BASH_ENV=/dev/null bash -n model/scripts/retired-names.sh (exit0), BASH_ENV=/dev/null UMPIRE_XTASK=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2477-target/debug/xtask bash model/scripts/retired-names.sh (exit0)
- PRs: