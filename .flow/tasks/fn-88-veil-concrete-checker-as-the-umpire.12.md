---
satisfies: [R22]
---
# fn-88-veil-concrete-checker-as-the-umpire.12 Move the model to Veil's declared toolchain

## Description
Only in R22 `adopt` mode: move the real model to Veil's declared toolchain. Set `model/lean-toolchain`, move Batteries, protobuf and binary in `model/lakefile.lean` and `model/lake-manifest.json`, and apply the model-side diff the R22 receipt recorded. Update every place that pins the toolchain (`mise.toml`, CI setup, docs such as `model/README.md` and CLAUDE.md's toolchain note if present). In any `defer-*` mode, close as not applicable citing the R22 receipt.

**Size:** M
**Files:** `model/lean-toolchain`, `model/lakefile.lean`, `model/lake-manifest.json`, `mise.toml`, model files named by the R22 receipt, `.github/workflows/umpire.yml` (if it pins the toolchain), `model/README.md`
**Touches:** [model/lean-toolchain, model/lakefile.lean, model/lake-manifest.json, mise.toml, model/**, .github/workflows/umpire.yml, model/README.md]

### Key context
- Goldens, fingerprints and every Property's meaning stay byte-identical: this is a toolchain move, not a semantic change.
- Gates: `make umpire-build-model`, `make umpire-check-goldens`, `LEAN_NUM_THREADS=1 make lint-model`, `make umpire-check-regression`, `make lint-code-fast`.

## Acceptance
- [ ] In `adopt` mode the model builds on Veil's declared toolchain with goldens, fingerprints and regression gates unchanged; in `defer-*` mode the task closes citing the R22 receipt.


## Done summary
Moved the model to Lean 4.32.0, Veil's declared toolchain. model/lean-toolchain and mise.toml (which CI's mise-action reads) now pin 4.32.0, and Batteries moved from v4.33.0 to v4.32.0 in lakefile.lean and lake-manifest.json. Fixtures.lean takes the one proof line the R22 receipt recorded. protobuf and binary already declare 4.32.0 and stay put. b90b63e072's lint fixes build unchanged on 4.32.0: Level.zero, String.trimAscii, and unused_variables_ignore_fn/IgnoreFunction all exist there. The cold build and lint-model both report 0 warnings. Goldens, the generated API, and the case-runtime fixtures are byte-identical, and regression is green. No other file pins the version. CI installs Lean through mise, and model/README.md and CLAUDE.md name no version. Follow-up: .plans/lean/UMPIRE4_DIRECTION.md:137 still says model/lean-toolchain pins 4.33.1. That line is outside this task's Touches, and another session has .plans edits in flight.

stage: impl-review - ran (claude:opus:high, round 1, SHIP)
## Evidence
- Commits: 0decb95262cbb50a017040274e3b756e7bab8a31
- Tests: baseline: green (make umpire-check-goldens on 4.33.1, rc=0, 27s), CC=/usr/bin/clang make umpire-build-model (cold, Lean 4.32.0, 599 jobs, 0 warnings, rc=0, 15m18s), make umpire-check-goldens (rc=0, goldens byte-identical), make umpire-check-lean-api (rc=0, generated API byte-identical), make umpire-check-case-runtime-conformance (rc=0, fixtures byte-identical), LEAN_NUM_THREADS=1 make lint-model (rc=0, 0 warnings, run alone, 28m00s), make umpire-check-regression (rc=0, incl. live tests, 29m04s), make lint-code-fast (rc=0, 0 issues), gate classify: FULL (unmatched .plans/UMPIRE4_ORDER.md, another session uncommitted edit); full gates ran on the committed bytes before commit, gate receipt: NO_RECEIPT (worktree dirty outside ignore set from another session) - non-blocking
- PRs: