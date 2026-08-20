# Umpire3 Veil backend

This directory is an isolated adapter for the versioned `FirstOrderView/v1` seam. The canonical
Umpire3 model remains in `tests/umpire3/model`; generated Veil modules are disposable views of that
model and never become semantic authority.

The Lake project pins Veil at `300c305e945750ab3fb62de4a79c23161b24da39` and Lean at 4.28.0.
`lake-manifest.json` locks the complete transitive dependency graph. To provision it from this
directory:

```sh
mise install
MATHLIB_NO_CACHE_ON_UPDATE=1 mise exec -- lake update
```

From the repository root, run the complete source, backend, replay, and result drift gate with:

```sh
make umpire3-check-veil-results
```

The gate regenerates the first-order views and Veil source, builds concrete and symbolic jobs,
executes both concrete checkers, replays every counterexample through the canonical Lean
`ExecutableView`, and compares normalized results in `results/`.

Trust is recorded in data:

- Veil concrete exhaustion is `external-no-counterexample` with `tested-instance` trust. It is never
  finite-exhaustive because Veil's concrete state identity can merge fingerprint collisions.
- A counterexample becomes `trace-witness` only after the canonical Lean replay executable accepts
  its digest-bound action trace. Lean collects and records the replay theorem's axiom inventory.
- `reconstructed-solver-proof` means `veil.smt.trust` is disabled and all generated verification
  conditions closed through reconstruction. It is deliberately distinct from `kernel` trust.
- `trusted-solver` means Veil accepted SMT UNSAT directly. The current Nexus pilot reports 11 such
  trusted goals.

Veil does not export a persistent theorem for `#check_invariants`, so its reconstructed job receipt
cannot be treated as a main-model Lean proof manifest. The pinned dependency build also warns that a
lean-smt bit-vector reconstruction declaration uses `sorry`; this pilot does not use bit-vectors,
but the warning is another reason not to upgrade reconstructed Veil evidence to `kernel` trust.
