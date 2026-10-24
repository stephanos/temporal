# Optional values shared interface checkpoint

The task's approved ordinal contract is represented by `umpire_ir::Bounded` and
`Type::{Bounded,Optional}`. The descriptor retains the authored wrapper/member,
primitive spelling, primitive bounds and authored finite bounds. `valid`,
`optional_valid`, `contains`, `encode` and `decode` centralize exact boundary checks;
`backing_type` supplies the primitive arithmetic type. Runtime IR values remain Int.

`ExprKind::{Bounded,Payload,Some,OptionalOr}` are explicit typed conversion points.
Bounded and Payload preserve the scalar. Some adds one; OptionalOr eagerly evaluates
both children and returns the fallback for zero, otherwise the code minus one.
Native direct wrapper construction is not finite-domain validation.

`umpire_dsl::Value::Bounded { name, raw }` and `Value::Optional(Option<Box<Value>>)`
carry typed native binding values. The machine macro emits one IntoValue implementation
per local Finite newtype using `declare::finite`; standalone Finite derive output and
checked constructors remain unchanged. This placement preserves the existing finite
expansion golden. The native accessor bridge reconstructs tuple/named wrappers using
backing conversion, without calling try_new, so authored invariants can refute invalid
native transitions. No generic Machine validation hook is introduced.

The shared IR/DSL crates compile at this checkpoint; backend exhaustive consumers are
intentionally not yet adapted. This checkpoint is not a workspace gate or task completion.
The pre-implementation source control failed at optional_ticks.rs:7:46 with
`unsupported type Option<Ticks>`. The unchanged full default baseline exited 0;
its orchestration wrapper was intentionally terminated only after the gate ended,
preventing queued redundant baseline runs. The wrapper status 143 is distinct from
the observed make kernel wait status 0. Final default/required/timing gates remain due.

The preserved manifest contains 827 artifacts pinned at 739c07a322. The audit also checks
all 15 original compatibility artifacts independently. Only new optional_ticks artifacts
may be added. No Activity policy or cancellation-boundary work belongs to this task.
