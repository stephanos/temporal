---
satisfies: [R5, R6, R7]
---
# fn-1-lean-regression-dsl-and-nexus.3 Wire regression checks and model documentation

## Description
Finish R5-R7 by exposing one repository check, verifying the isolated dependency boundary, and documenting the developer workflow. This task owns integration/docs rather than changing compiler semantics.

**Size:** M
**Files:** `Makefile`, `model/Makefile`, `model/README.md`
**Touches:** [Makefile, model/Makefile, model/README.md]

### Approach
- Add a focused top-level regression check that builds the compiler/pilot and exercises inspector determinism and negative behavior.
- Keep the new Lean module imports and build-target references on an explicit positive allowlist of the current model, generated API catalog, and standard Lean dependencies. Do not add forbidden-system-aware code or inspect any out-of-scope source tree.
- Keep all model build recipes in the top-level `Makefile`; remove the model-local makefile and keep the existing API-check model build direct from the top level.
- Extend the model guide with the Lean-first declaration entrypoint, bounded Nexus pilot, compile/check/inspect commands, artifact fields, and attempt-versus-transition authority boundary.
- Preserve the structural-only meaning of generated declarations without invoking an out-of-scope generator check in this task.

### Investigation targets
**Required** (read before coding):
- `Makefile:962-971` — existing API generation/check target pattern and model-check delegation point.
- `model/Makefile:1-4` at task base — model-local recipe being moved to the top-level Makefile.
- `model/README.md:1-27` — current semantic-authority and developer-command documentation.

### Key context
- Verify the dependency boundary positively from imports and the new top-level target recipes; do not search, inspect, or compare any Umpire3 material.
- Documentation must not use Umpire3 as history, comparison, or implementation guidance.
- Add no contributor prerequisite and no third-party dependency.
- Preserve all existing comments in touched files.

### Task-scoped verification
- The pre-edit baseline command is `cd model && mise exec -- lake build`; the absence of the top-level regression target is the planned gap this task closes.
- Completion requires the spec Quick command `make umpire-check-regression`.

## Acceptance
- [ ] `make umpire-check-regression` checks the compiler, Nexus pilot, canonical inspector output, and negative inspector behavior.
- [ ] The new experiment modules import only the explicitly allowed current modules and standard Lean dependencies; the top-level recipes invoke only current-model commands.
- [ ] The model-local makefile is removed, the existing API-check recipe delegates to the new top-level model target, and generated API declarations remain structural-only.
- [ ] `model/README.md` explains how to declare, compile/check, and inspect the bounded pilot without a running Temporal server.
- [ ] Documentation lists the inspectable contract fields and preserves requested-action versus successful-transition semantics.
- [ ] The new top-level regression target invokes no Umpire3 code, target, artifact, test, or runtime behavior.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
