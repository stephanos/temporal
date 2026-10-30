---
satisfies: [R2, R3, R4]
---
# fn-256-retire-the-scala-era-build-and-go.2 Delete the archived Scala Make targets and the scala-cli pin; mark model/archived read-only

## Description
Delete the archived Scala Make targets. That is the `check-framework` wrapper (Makefile lines 1–8 and its matching `endif` at 1268) and the block from `# Archived Scala targets` to just before `goimports:` (541–817). Also delete `fmt-model`, `lint-model`, `fix-model` and any other Scala targets on the `.PHONY` line at 440. Drop `scala-cli` from `mise.toml` and remove only its tool/platform tables from root `mise.lock`, preserving all other lock bytes. Keep the `.flow/tmp` scratch half of `umpire-clean-scratch` if agents still use it, and move it out of the deleted block. Record the admitted integrated baseline SHA before edits. Commit the tooling removal, then add the short read-only note to `model/archived/README.md` in a subsequent commit naming that existing removal commit. Leave everything else in `model/archived` untouched. Correct only the `.gitignore` comment that currently says `umpire-clean-scratch` removes Scala build state; preserve `.bsp/` and `.scala-build/` ignore patterns. Preserve `model_quote`, which the live `SOURCE_FIND` definition still uses outside the archived block. Serialize shared Make/guidance/pin edits with fn-254 and preserve its integrated changes.

**Size:** S
**Files:** `Makefile`, `mise.toml`, `mise.lock`, `model/archived/README.md`, `AGENTS.md`, `model/README.md`, `.gitignore` (comment only)
**Touches:** [Makefile, mise.toml, mise.lock, model/archived/README.md, AGENTS.md, model/README.md, .gitignore]

## Acceptance
- [ ] `rg -n 'scala-cli|MODEL_JAR|MODEL_ROOT|scalafix|scalapb|check-framework' Makefile mise.toml` is empty. Record explicit admitted baseline and final commit SHAs. `git diff --name-status <baseline> <final> -- model/archived` shows only README.md; compare every other archived tracked path and blob identity against that baseline, and report any uncommitted archive changes separately. The README names the existing tooling-removal commit from its subsequent documentation commit. The `.gitignore` diff changes only the obsolete cleanup comment and preserves both Scala ignore patterns.
- [ ] `make -n model-check`, `make -n umpire-check-live-tests` and `make canary-build` work. Record parse time (`time make -n canary-build >/dev/null`) before and after.
- [ ] fn-254's Rust model Make section, located by its stable marker rather than stale line numbers, is byte-identical against the admitted integrated baseline. R4 also needs an actual `make model-check` result or explicitly identified equivalent integrated fn-254 evidence on matching inputs. A dry run is not behavioral proof. Batch any still-required broad run with fn-254's final checkpoint; keep the obligation open until that evidence exists.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
