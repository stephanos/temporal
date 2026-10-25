---
satisfies: [R7]
---
# fn-254-fast-validation-gates-and-a-real-inner.11 Standalone control: framework gate passes on model/umpire alone


## Description
Prove the framework stands alone and stays Temporal-agnostic (spec R7 errors), as a check the framework gate runs itself.

**Size:** S
**Files:** a framework control under `model/umpire/dev/` (xtask subcommand or test), README.
**Touches:** [model/umpire/dev/**, model/umpire/README.md]

### Approach
- Copy the exact current candidate bytes of `model/umpire` into scratch outside the repository, retaining a source manifest; `git archive HEAD` alone omits candidate edits. Exclude generated caches and reject source symlink escapes. No Temporal tree, root Makefile/.bin, Git metadata or repository-specific environment may supply an undeclared source/tool dependency. Use a separately owned compilation target, not the main checkout's target.
- The outer control runs the framework entry point once in a named child mode that skips only this copy-control stage. All other framework stages run exactly once; recursion and a silently skipped stage fail. Record cost and tier, then register the stage for .12.
- Agnosticism checks follow .9's policy ownership: filesystem-based scanning works without Git; generic dependency/path checking has no Temporal knowledge; Temporal deny-vocabulary and any narrowly justified policy-data exceptions are explicit. Preserve the existing Temporal boundary test. Check both declared and resolved path dependencies; do not let copied caches supply absent inputs.
- Negative controls: a planted path dependency on a Temporal crate fails, a planted `temporal` identifier fails, and a planted `../../../..` repo-root path fails.

### Investigation targets
**Required:**
- `model/design/framework-gate.md`
- `model/dev/xtask/src/retired_names.rs` (or its moved successor)
- `model/scripts/test-capabilities.sh` — existing control pattern

## Owner validation cadence (2026-10-25)

The real directory-only framework entry-point run is the isolation gate being changed and remains required, alongside planted violation controls. Reuse equivalent evidence at .8 only if all relevant inputs remain unchanged. Deferred checks remain final .8 requirements and are never reported green. This owner instruction supersedes earlier broad per-task verification commands.

## Acceptance
- [ ] The framework gate passes on an isolated copy of `model/umpire`.
- [ ] All three planted violations fail with a named reason.
- [ ] Cost recorded; the control's tier (quick, check or validate) is stated in the README.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
