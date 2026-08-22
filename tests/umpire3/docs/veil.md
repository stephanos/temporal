# Umpire3 Veil backend

Veil is embedded in the primary Umpire3 Lean project. The canonical model and its Veil declarations
live together under `tests/umpire3/model`; Umpire3 does not generate a second Veil source model.

The primary Lake project pins Veil at `300c305e945750ab3fb62de4a79c23161b24da39` and Lean at 4.28.0.
Its `lake-manifest.json` locks the complete transitive dependency graph. To provision it:

```sh
cd tests/umpire3/model
mise install
MATHLIB_NO_CACHE_ON_UPDATE=1 mise exec -- lake update
```

From the repository root, run the complete source, backend, replay, and result drift gate with:

```sh
make umpire3-check-veil-results
```

The authored `.lean` files are inputs to every command. `make umpire3-export-veil-bindings` exports
their source-bound JSON bindings, while `make umpire3-record-veil-results` deliberately replaces the
retained normalized evidence after a reviewed semantic or toolchain change. Neither command emits
Veil source.

The gate checks Lean-exported declaration bindings against the first-order views, builds concrete
and symbolic jobs, executes both concrete checkers, replays every counterexample through the
canonical Lean `ExecutableView`, and compares normalized results in `testdata/retained/`.

Trust is recorded in data:

- Veil concrete exhaustion is `external-no-counterexample` with `tested-instance` trust. It is never
  finite-exhaustive because Veil's concrete state identity can merge fingerprint collisions.
- A counterexample becomes `trace-witness` only after the canonical Lean replay executable accepts
  its digest-bound action trace. Lean collects and records the replay theorem's axiom inventory.
- `reconstructed-solver-proof` means `veil.smt.trust` is disabled and all invariant verification
  conditions closed through reconstruction. The receipt is compiled from named Lean theorems and
  records their transitive axioms. It is deliberately distinct from
  `kernel` trust.
- `trusted-solver` means Veil accepted SMT UNSAT directly. This includes bounded symbolic search,
  whose pinned implementation can fall back to raw UNSAT even when `veil.smt.trust` is disabled,
  and the invariant mode that deliberately enables solver trust.

Veil's invariant theorems remain backend evidence rather than main-model Lean proof manifests. The
pinned dependency build also warns that a lean-smt bit-vector reconstruction
declaration uses `sorry`; this pilot does not use bit-vectors, and reconstructed receipts fail
closed if their actual axiom inventory contains `sorryAx`.
