---
satisfies: [R5, R6, R7]
---
# fn-1-lean-regression-dsl-and-nexus.3 Wire regression checks and model documentation

## Description
Finish R5-R7 by exposing one repository check, verifying the isolated dependency boundary, and documenting the developer workflow. This task owns integration/docs rather than changing compiler semantics.

**Size:** M
**Files:** `model/Makefile`, `model/README.md`
**Touches:** [model/Makefile, model/README.md]

### Approach
- Add a focused model check that builds the compiler/pilot and exercises inspector determinism and negative behavior.
- Keep the new Lean module imports and build-target references on an explicit positive allowlist of the current model, generated API catalog, and standard Lean dependencies. Do not add forbidden-system-aware code or inspect any out-of-scope source tree.
- Keep the command model-local so invoking it does not parse unrelated root build machinery.
- Extend the model guide with the Lean-first declaration entrypoint, bounded Nexus pilot, compile/check/inspect commands, artifact fields, and attempt-versus-transition authority boundary.
- Preserve the structural-only meaning of generated declarations without invoking an out-of-scope generator check in this task.

### Investigation targets
**Required** (read before coding):
- `model/Makefile:1-4` — existing Lean build entrypoint.
- `model/README.md:1-27` — current semantic-authority and developer-command documentation.

### Key context
- Verify the dependency boundary positively from imports and model-local build references in the touched scope; do not search, inspect, or compare any Umpire3 material.
- Documentation must not use Umpire3 as history, comparison, or implementation guidance.
- Add no contributor prerequisite and no third-party dependency.
- Preserve all existing comments in touched files.

### Task-scoped verification
- The pre-edit baseline command is `make -C model check`; the absence of `make -C model check-regression` is the planned gap this task closes.
- Completion requires both spec Quick commands: `make -C model check` and `make -C model check-regression`.

## Acceptance
- [ ] `make -C model check-regression` builds and checks the pure Lean compiler, Nexus pilot, canonical inspector output, and negative inspector behavior.
- [ ] The new experiment modules import only the explicitly allowed current modules and standard Lean dependencies; the model recipe invokes only the new current-model targets.
- [ ] The model-local check does not parse or invoke root build machinery, and generated API declarations remain structural-only.
- [ ] `model/README.md` explains how to declare, compile/check, and inspect the bounded pilot without a running Temporal server.
- [ ] Documentation lists the inspectable contract fields and preserves requested-action versus successful-transition semantics.
- [ ] The new model target invokes no Umpire3 code, target, artifact, test, or runtime behavior.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
