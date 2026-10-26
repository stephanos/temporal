# Unvalidated generated-source spike

Nothing here has been built or executed. This is retained implementation preparation,
not evidence, a measured design verdict, or an activated gate.

`bash run.sh /absolute/new/scratch --seed 255001 --cases 1` writes a standalone Cargo
crate containing seeded source Models. It refuses an existing destination. Add
`--execute` only during an explicitly admitted future run to compile offline through
the framework's pinned mise toolchain and execute the probe. Batch size is restricted
to 1–4, compile timeout 120 seconds, run timeout 45 seconds per selected case. These
are experimental ceilings, not measured R7 allocations. Backend processes have
10-second timeouts, 512 MiB default heap, bounded capture/dumps and worker count one.
Tool discovery admission is retained; the outer run timeout also covers discovery.

Each printed Model has two two-variant enum fields, an integer updated only up to
1 or 2, an Option of a two-value Finite wrapper, four unit actions, guarded arithmetic,
rejection, optional facts and one bounded invariant. The Cartesian state ceiling is
2*2*3*3 = 36, emitted edge ceiling 36*4 = 144, shortest reachable depth <=35.
There are no nondeterministic choices or input payloads in this first spike.

Independent adapter paths:

- Compiled: Checker over the compiled World's own starts and handlers, typed values
  exported independently of parsing and canonical labeled edges extracted from rows.
- Interpreter: Interpreted::new over the macro-embedded admitted tree, its own starts,
  same generic graph recorder; this does not call the existing compare helper that
  accepts matching refusals as zero-state agreement.
- TLC: Lift::parse of each complete printed source, followed by Tla::emit, ordinary
  complete check and a second independent full extraction exploration. No compiled
  states/starts/edges construct the TLC expectations. Extraction wraps the emitter's
  real Edge operator, retains predecessors and a start/step bit, and promotes the
  existing private action/choice/outcome/facts assignments to decoded state fields.
  The native/TLC comparison includes starts, states, full labeled edges and verdicts.
  Synthetic Complete transitions are deliberately excluded. The generated Refuse
  action always emits an enabled rejection edge, so this particular spike has no
  deadlocks. Extending to deadlock Models requires a separate explicit extraction
  completion design and control.

Raw augmented assignments are capped at 8192 after bounded dump decode. The capture
and wall limits constrain work before that postcondition; no pre-exploration TLC
assignment counter has been established. The proposed extraction shape does not yet
have execution proof. Ordinary claims plus extraction nominally need three model JVM
launches on healthy single-invariant cases, plus admitted tool discovery; recovery
and strategy-equivalence features can add launches. No launch-budget claim is made.

Lowering controls clone only the interpreter's embedded tree: swap one Lt to Gt in
a guard, and independently erase branch fact sequences. The latter must leave
states/verdicts equal while complete edge agreement fails. A dedicated comparator
control requires matching backend errors to fail. These are actual source controls,
not assertions that they passed. Outcome-only mutation, dropped-guard mutation,
automated recompiling source shrink, proptest tree construction, multi-Model TLC
batching and launch-count/process observations remain unimplemented spike work.

To minimize a future failing seeded case, retain inputs.json and use the deterministic
source parameters upper=1, start=Off, token=0 as bounded candidate reductions. No
shrinking result or minimality claim exists. Task .1 cannot close until executable
shrinking, complete TLC extraction proof, batching and measured bounds are supplied.
No Temporal dependencies are used. No production manifest or module registration
was modified.
