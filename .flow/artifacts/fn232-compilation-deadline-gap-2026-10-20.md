# Rust foundation: compilation and teardown deadline acceptance gap

Task fn-232.1 remains in progress. Its guest fuel, memory and epoch-watchdog
fixtures are not proof that the existing compilation-inclusive wall deadline is
preserved. No boundary or acceptance requirement has been changed.

The frozen Go path starts `context.WithTimeout` before helper input and runs the
helper through `hostexec.Run`, so an expired execution can kill compilation with
that process. The linked Rust seam starts its watcher before compilation and
refuses to instantiate after an observed overrun, but synchronous Wasmtime
compilation must return first. This is a real acceptance difference, not a
performance claim or a measured maximum compilation time.

Research against exact Wasmtime47.0.3 source found no supported safe in-process
compiler cancellation API. Epochs interrupt generated guest instructions, not
compiler work. A timed waiting thread would leave compilation running; killing
the entire host loses other slots and evidence finalization. Neither preserves
the old per-execution contract. Primary references:
[Module compilation](https://github.com/bytecodealliance/wasmtime/blob/v47.0.3/crates/wasmtime/src/runtime/module.rs#L247),
[synchronous builder](https://github.com/bytecodealliance/wasmtime/blob/v47.0.3/crates/wasmtime/src/compile/code_builder.rs#L790),
[epoch scope](https://github.com/bytecodealliance/wasmtime/blob/v47.0.3/crates/wasmtime/src/config.rs#L659).
The detailed read-only research is `.flow/tmp/fn232-compiler-cancellation-research.md`.

An owner decision is required before accepting task .1: authorize an isolation
exception that preserves cancellation and reclamation for every bounded phase,
explicitly change the deadline contract, or authorize cancellation work beyond
the current pinned dependency/API boundary. The earlier recommendation of a
compilation-only worker was incomplete: it addresses compilation, not proof of
bounded in-process teardown. It is not a complete repair or authorization to
introduce subprocesses. Disjoint evidence, store and parity checks may continue;
source integration and final qualification must not conceal this gap.

## Integrated independent review

Candidate `a618eddafc` passed the root's fresh foundation/raw-integrity/golden
check and authenticated 12/12 comparison against unchanged retained fresh Go
results. Root logs are `/home/agent/wt/evidence/fn232-foundation/root-r3{,-go-reference,-semantic}.log`.
The single batch handover is
`/home/agent/wt/evidence/fn232-foundation/handover/batch-interface.md`.

Independent round-two review, session `01a15bae-a4f4-7031-a672-a14f3f4b94b4`,
confirmed both R3 storage findings fixed and R2/R3/R13/R14 met within the
foundation scope. It remains **NEEDS_WORK**, with R1 the sole unaddressed
requirement; all four foundation tasks remain in progress. The receipt is
`.flow/tmp/impl-review-receipt-fn232-foundation.json`, raw output
`.flow/tmp/fn232-foundation-review-round2.log`. No future verb, qualification,
deadline waiver or task completion is certified by these results.
