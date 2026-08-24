# Root-owned Umpire model build

## Goal

Keep `make umpire-check-api` as the sole Make shortcut for checking the generated Temporal API model and building its Lean project. Remove the redundant `model/Makefile`.

## Design

The root `Makefile` will continue to generate and check the Umpire API artifacts. Its `umpire-check-api` target will then change directory to `model/` and run `mise exec -- lake build` directly.

The `model/` directory remains a normal Lake project. Its `lean-toolchain`, `lakefile.toml`, and Mise tool pin continue to define the Lean build; it no longer exposes a second Make interface.

## Error handling

The root recipe will preserve the existing failure behavior: a failed generated-artifact check or Lean build returns a nonzero status and fails `make umpire-check-api`.

## Verification

- Run `make umpire-check-api`.
- Confirm no repository file references `model/Makefile` or invokes `make -C model` for this model.
- Confirm `model/Makefile` is absent.

## Scope

This change does not alter generated artifacts, Lean declarations, generator behavior, Umpire3 model build wiring, or other Make targets.
