# Umpire in Rust: decision

The workspace now lives at `model/`, with the frozen Scala reference under `model/archived/`.
Run `make model-check` and `make model-time` from the repository root. Measurements below retain
the command names used when they were recorded, before the workspace move.

## Stock-toolchain wasm verdict (fn-210 R24)

Recommendation: **Rust**. On 2026-10-18 the trait-shaped Kettle compiled with stock
Rust 1.99.1 for wasm32-wasip1 and ran in wazero v1.12.0 with Go 1.27.0 and
`CGO_ENABLED=0`. The generated protobuf Explore report and all ordered Rows matched
the existing native trait fixture goldens byte-for-byte: 81 starts, 283 reached
states, 424 emitted rows and depth 62. This is the decisive R20 slice before the
attribute macros. The remaining fn-210 tasks continue; this note does not claim
their authoring, liveness or lifting work is finished.

The three application exports are `umpire_buffer_alloc`, `umpire_buffer_release`
and `umpire_call`. Both sides use
generated messages from `umpire/proto/umpire/oracle/v1/oracle.proto`; Refusal is a
case of the common Response envelope, separate from a checker verdict or trap.
The guest includes `umpire/tests/fixtures/kettle_trait/model.rs` directly.

Rust exports these names directly using the stock linker. The buffer-release name
avoids WASI libc's internal `free` symbol. The default gate compiles and runs the
real guest through the Go host, validates its export signatures, and rejects
missing, unexpected and mis-typed exports through compiled negative controls.
Direct locked Cargo builds need no custom linker or post-link processing.

## Capability and panic boundaries

The real guest imports only `environ_get`, `environ_sizes_get`, `fd_write` and
`proc_exit` from `wasi_snapshot_preview1`, with exact checked signatures. The host
provides an empty environment, discarded descriptor 1/2 output and an exit trap.
It links no general WASI implementation, directories, inherited I/O, clocks or
thread APIs. Unexpected imports and imported memory fail before instantiation.

Core-local Clippy policies reject resolved standard filesystem, clock and thread
methods/types, including aliases, in both DSL and checker. Host build scripts
retain narrowly scoped allowances for fixture discovery. Negative controls prove
that each policy rejects representative methods and types, and the gate lints
the actual native and wasm core libraries. The linked-import check complements
this source restriction: it covers surviving transitive guest code. Neither is
a universal analysis of dormant third-party code, cfg-disabled branches or every
future API. Dependency changes still require admission and review.

WASI uses panic=abort. Unexpected guest panics trap and invalidate the host
instance; host errors name the logical method and failing ABI function. Native
checker catch_unwind behavior and panic/overflow fixtures remain unchanged.
Invalid budgets and state IDs remain generated typed refusals, not panic tests.

## Attribute Kettle and the authoring sketch (fn-210.6)

The canonical `umpire/tests/fixtures/kettle/model.rs` uses the owner-approved
flat surface derived from `options/Kettle.attrs.rs`, retaining its separate
`User` and `Heater` enums, transition bodies and claim predicates.

Remaining differences from the hypothetical sketch are deliberate:

- Imports name the actual `umpire_dsl` crate instead of a hypothetical prelude.
- `#[machine]` registers one inline public `kettle` module with one `impl Model`.
  Per-item attributes mark methods, grouped contiguously in on, effect, invariant,
  postcondition, assumption and progress order. There are no group attributes or
  marker traits; attributes preserve authored statements and method/impl headers.
- The state is named `Model` to match Scala's `object Model`, replacing the
  sketch's `Kettle` name while leaving its fields and predicates unchanged.
- The module discovers action enums in declaration order rather than taking
  an actor tuple, preserving press, depress, heat, failover catalog order.
- Actor-level `#[internal]` replaces the hypothetical `Internal` derive.
- `State` supplies clone, equality, hashing, canonical `Spell` and ordered
  starts with defaults for unannotated booleans; `Action` supplies clone,
  equality, hashing, spelling and ordered enumeration.
- `#[postcondition(after = Heater::Failover)]` selects the failover action;
  its unused `before` parameter is omitted while the warming claim keeps it.
- The named assumption is a static function returning a module-local `Assumption`
  value through `without([User::DepressButton])`, replacing the associated
  constant and hiding action adaptation.
- `whenever(self.heating_on).eventually(self.beeping).assuming(Self::never_switched_off())`
  builds a module-local `LeadsTo` value with boolean trigger and target and
  names its assumption in the expression. This builder remains; `path!` is deferred to fn-215.
- Private generated world/action adapters implement the existing `Machine`
  protocol; the public `world()` function returns an opaque machine exposing
  its state and actor conversions without exposing generated type names.
- Each transition runs on a clone and emits one successor with unit outcome
  and no facts only when its final state differs from the original state.
  A stuttering transition emits no edge, so its filtered postconditions are not checked.
- State fields are public so fixture controls can construct their own starts;
  full and cut producers sit outside the public Model module and use `world()`.
- The existing no-heat control is generic over that public machine interface,
  using `From<User>` and `From<Heater>` for actions while preserving its
  single 20-degree start, disabled heat step and scoped progress claim.
- The shared fixture runner replaces the sketch's hypothetical test APIs,
  verifying all claims including `boils` and rendering the `never_beeps`
  refutation of the sketch's `!beeping` predicate.
- Registration retains no syntax tokens and performs no lifting or subset
  analysis, following R36 rather than the sketch's lifting comment.

Scala's `transitions` block selects an action with `on` and guards its effect:

```scala
object transitions extends Transitions:
  on(user.pressButton):
    if !heatingOn then effects.pressButton()
  on(heater.failover):
    if heatingOn && temperature >= boilingTemperature then effects.failover()
  on(heater.heat):
    if heatingOn && temperature < boilingTemperature then effects.heat()
  on(user.depressButton):
    if heatingOn then effects.depressButton()
```

Rust groups the same actions into one match per actor, with guards and effects inline:

```rust
impl Model {
    #[on(User)]
    fn user(&mut self, action: User) {
        match action {
            User::PressButton if !self.heating_on => {
                self.heating_on = true;
                self.beeping = false;
            }
            User::DepressButton if self.heating_on => {
                self.heating_on = false;
                self.beeping = false;
            }
            _ => {}
        }
    }

    #[on(Heater)]
    fn heater(&mut self, action: Heater) {
        match action {
            Heater::Heat if self.heating_on && self.temperature < BOILING_TEMPERATURE => {
                self.temperature += 1;
            }
            Heater::Failover if self.heating_on && self.temperature >= BOILING_TEMPERATURE => {
                self.heating_on = false;
                self.beeping = true;
            }
            _ => {}
        }
    }
}
```

`#[machine]` binds each `#[on(Actor)]` method to the named actor, checks its parameter type,
and dispatches `step` through it. The fallback leaves the state
unchanged and disables that action. Omitting it lets rustc check exhaustiveness. Actor declaration
order still determines catalog order, so grouping the arms changes no canonical payload.
The concrete unit-only actor enums also derive `Copy`, keeping owned action parameters idiomatic
under Clippy without changing `derive(Action)` or the lint policy.

Scala's `object effects` corresponds to optional `#[effect]` methods immediately after the
`#[on]` methods. These are ordinary shared helpers such as
`fn switch_off(&mut self, beep: bool)`; Kettle needs none. Only `#[on]` and `#[effect]` methods
may take `&mut self`. Every other `Model` method, including an unmarked helper, takes `&self`
or no receiver. Compiler-resolved read-only checks cover qualified, nested, imported and aliased
`Model` impls while leaving unrelated helper types usable. Under the approved checking exception,
compiler-only declarations may be added inside local function or const scopes to resolve aliases
and trait projections. Original authored statements and method/impl headers remain unchanged;
these declarations add no runtime guard. The generated module carries
`#[forbid(unsafe_code)]`.

The `State: Hash` requirement already excludes standard interior-mutable state such as `Cell`
and `RefCell`. Thread-locals and atomics in statics fall under the capability Clippy policy,
rather than the receiver-signature check.

Invariant and postcondition methods are synchronous, return nothing and use `check!`
and `unknown!` statements, remaining ordinary methods when called directly.
A scoped per-evaluation recorder collects failed check expressions and holes without
panics or unwinding, and generated predicates turn the recorded result into a verdict.
Progress methods run once per real reached state; each claim's assumption must stay
fixed within one exploration, with a mismatch reported as Unknown with a diagnostic.
An empty world evaluates no progress method and is vacuously Verified; manual fixed
assumptions are still validated before exploration.
The legacy `Scope` alias and trait builders remain for manual trait sources only;
the attribute surface exposes `Assumption`, `#[assumption]`, `LeadsTo` and `.assuming(..)`.
The core `Progress` builder remains unchanged for manual trait Models.

The owner approved exactly five canonical payload substitutions on 2026-10-18:
`Act::UserPressButton` → `User::PressButton`, `Act::UserDepressButton` →
`User::DepressButton`, `Act::HeaterHeat` → `Heater::Heat`, and
`Act::HeaterFailover` → `Heater::Failover`, and state prefix `Kettle { ` →
`Model { `. This includes dependent `kettle_cut`
vacuity output and `kettle_no_heat` rows and progress counterexample. Counts,
state values, catalog and discovery order, claims,
verdicts and counterexample structure stay unchanged. Snapshot metadata and
symlink targets stay unchanged too. The separate `kettle_trait` fixture and
its native/WASM reference payloads remain byte-identical.

## Measurements

The direct-export guest built for fn-210.16 on 2026-10-18 is 3,370,765 bytes,
including debug information, using the same release profile and toolchain below.

The following host latency and memory measurements were taken for fn-210.9 on
2026-10-18, before the direct-export rename in fn-210.16. They do not measure the
current binary; fn-210.7 will refresh them.

Measured on Linux arm64 with stock Rust 1.99.1, Go 1.27.0, protoc 29.5,
prost/prost-build 0.14.4, protoc-gen-go v1.36.6 and protobuf Go v1.36.11.
The release profile uses overflow checks, debug=1, thin LTO and one codegen unit;
that earlier wasm file was 2,872,544 bytes including debug information.

Five cold compile/instantiate/close runs, excluding file reading, took 29.217,
28.059, 27.874, 26.774 and 26.766 ms (median 27.874 ms).
Five runs of exactly 100,000 real Rows calls took 14,003, 13,838, 13,686, 24,061
and 24,269 ns/op (median 14,003 ns/op; range 13,686–24,269). These are aggregate
per-operation means, not latency percentiles. Each call includes protobuf
marshal, alloc/write/call/read-copy/free and protobuf decode, cycles state IDs
modulo 283, and consumes row text. Each run consumed 17,705,239 bytes; guest
memory stayed at 17 pages before and after every run. Go allocation counts were
28 allocs/op and 1,437–1,447 B/op. Runs include ordinary host scheduling noise.

Reproduce from `umpire/hosts/go` inside `model/`, with `UMPIRE_WASM` naming the built guest:

```sh
CGO_ENABLED=0 mise exec -- go test -run '^$' -bench '^BenchmarkColdLoad$' -benchtime=1x -count=5
CGO_ENABLED=0 mise exec -- go test -run '^$' -bench '^BenchmarkRows$' -benchtime=100000x -count=5 -benchmem
```

In fn-210.9, the integrated `make model2-time` passed with a 5.069 s warm full gate (30 s
budget) and 1.210 s checker edit/rebuild/test loop (3 s budget). This includes
capability/export controls, generation freshness, native checks, wasm builds
and uncached CGO-disabled Go parity tests. Raw measurements and final gate
output are retained in the fn-210.9 task evidence under `.flow/tmp/fn210-9-*`.
