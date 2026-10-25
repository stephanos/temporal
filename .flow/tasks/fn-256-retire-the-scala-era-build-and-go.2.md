---
satisfies: [R2, R3, R4]
---
# fn-256-retire-the-scala-era-build-and-go.2 Delete the archived Scala Make targets and the scala-cli pin; mark model/archived read-only

## Description
Delete the archived Scala Make targets. That is the `check-framework` wrapper (Makefile lines 1–8 and its matching `endif` at 1268) and the block from `# Archived Scala targets` to just before `goimports:` (541–817). Also delete `fmt-model`, `lint-model`, `fix-model` and any other Scala targets on the `.PHONY` line at 440. Drop `scala-cli` from `mise.toml`. Keep the `.flow/tmp` scratch half of `umpire-clean-scratch` if agents still use it, and move it out of the deleted block. Add a short read-only note to `model/archived/README.md` naming the removing commit. Leave everything else in `model/archived` untouched.

**Size:** S
**Files:** `Makefile`, `mise.toml`, `model/archived/README.md`, `AGENTS.md`, `model/README.md`

## Acceptance
- [ ] `rg -n 'scala-cli|MODEL_JAR|MODEL_ROOT|scalafix|scalapb|check-framework' Makefile mise.toml` is empty; `git diff --stat -- model/archived` shows only README.md.
- [ ] `make -n model-check`, `make -n umpire-check-live-tests` and `make canary-build` work. Record parse time (`time make -n canary-build >/dev/null`) before and after.
- [ ] fn-254's model section (from line 1270) is byte-identical.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
