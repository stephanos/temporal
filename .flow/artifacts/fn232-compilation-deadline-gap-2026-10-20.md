# Rust foundation: compilation deadline acceptance gap

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

An owner decision is required before accepting task .1: permit a killable
compilation-only worker while retaining in-process guest execution, explicitly
change the compilation deadline contract, or authorize compiler cancellation
work beyond the current pinned dependency/API boundary. The conductor recommends
the compilation-only worker to preserve the existing bound. This recommendation
is not authorization, an implemented fix or an acceptance pass. Disjoint evidence,
store and parity checks may continue; source integration and final qualification
must not conceal this gap.
