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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
