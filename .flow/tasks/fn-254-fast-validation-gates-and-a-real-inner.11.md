---
satisfies: [R7]
---
# fn-254-fast-validation-gates-and-a-real-inner.11 Standalone control: framework gate passes on model/umpire alone

Touches: [model/umpire/dev/**, model/umpire/README.md]

## Description
Prove the framework stands alone and stays Temporal-agnostic (spec R7 errors), as a check the framework gate runs itself.

**Size:** S
**Files:** a framework control under `model/umpire/dev/` (xtask subcommand or test), README.
**Touches:** [model/umpire/dev/**, model/umpire/README.md]

### Approach
- Export `model/umpire` alone (`git archive HEAD:model/umpire` or a filtered copy) into scratch, with no `model/temporal`, root `Makefile` or root `.bin/`. Run the framework entry point there with a shared target; it must pass. Run in the full gate only if its cost is high; record the cost.
- Agnosticism checks in the framework gate: no `umpire` crate depends on a Temporal crate (`cargo metadata`), and no Temporal names in `model/umpire` (reuse or extend the existing retired-names/no-Temporal check; see the No-Temporal-in-umpire rule).
- Negative controls: a planted path dependency on a Temporal crate fails, a planted `temporal` identifier fails, and a planted `../../../..` repo-root path fails.

### Investigation targets
**Required:**
- `model/design/framework-gate.md`
- `model/dev/xtask/src/retired_names.rs` (or its moved successor)
- `model/scripts/test-capabilities.sh` — existing control pattern

### Acceptance
- [ ] The framework gate passes on an isolated copy of `model/umpire`.
- [ ] All three planted violations fail with a named reason.
- [ ] Cost recorded; the control's tier (quick, check or validate) is stated in the README.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
