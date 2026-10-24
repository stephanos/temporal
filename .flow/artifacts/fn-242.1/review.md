# fn-242-length-arithmetic-parity-across.1 review

Commit 7bd76093f9 on `work/fn2421-lengths`, base c3c95eb00b. Fresh-context review against the spec (R1, R2), the task, the handoff and the owner rules.

## Verdict: SHIP

No blockers or majors. R1 and R2 are met. Every handoff claim I checked holds. The findings below are one minor over-refusal with a misleading message, a minor gap in R2 coverage, and some test and wording nits.

## How I verified it

- Read the full diff: front, interpreter, fixtures, lifter tests, TLC runner, lib, controls and docs.
- With a private target and `UMPIRE_BACKENDS=require`, these passed:
  - `umpire-fixtures --test interpreted length`: both length controls.
  - All of `umpire-tlc`: 18 unit tests and 4 real-TLC controls.
  - `umpire-lift --test vector --test narrow_remainder`.
- Ran a throwaway lifter probe of 18 `len()` forms through `Lift::parse` and `Tla::emit`. I deleted it afterwards.
- Ran two throwaway real-TLC probes of 14 evaluation-error and counterexample shapes. I deleted them afterwards.
- The worktree is clean and the private target is deleted.

## Verified claims

### R1: the front check is type-based and covers every position
- The check is `left.ty == index_type()`. The right operand is always typed from the left, and an unsuffixed literal on the left takes the right's type, so a length on either side triggers it.
- `index_type()` (`Int{0, i32::MAX}`) is the only source of that type. `usize`, `u32` and `u64` are refused as wider than i32 (`typer.rs:18-25`), so no field, constant, World value or input can carry it.
- `-len()` doesn't compile as Rust.

### R1: probe results

Refused at the binary expression's span:

| Form | Span |
|---|---|
| `len() + 1` in a `check!` | |
| `len() + 1` in a match guard | `25:31` |
| `let n = len(); n + 1` | `25:63` |
| `self.count() + 1`, where `count` is a helper returning `len()` | |
| `(len() - 1) * 2` | |
| `2 * (len() / 2)` | |
| `len() + len()` | |

Refused elsewhere:
- `len() as i32` is refused as an unsupported expression.
- `saturating_add` and `wrapping_mul` are refused as unsupported methods.
- `len() < 3_000_000_000` is refused because the literal exceeds its declared type.

Admitted, with agreement checked by reasoning:
- `len() - 1`, `len() - 2147483647`, `/`, `%`, and `self.count() - 0`.
- Both operands lie in `0..=i32::MAX`. A difference therefore stays inside TLC's 32-bit range, and a negative one fails all three back ends:
  - Rust: `overflow-checks = true` in both the dev and release profiles (`model/Cargo.toml:123-127`).
  - Interpreter: the range check.
  - Lifter: an `Assert`.
- `__umpire_Div` and the `Rem` emission (`left - q*right`) don't overflow for non-negative operands.
- `GetOr` emits `s[i+1]` only under `i < Len(s)`, and TLC evaluates `IF` lazily.

### R1: agreement controls
- `differences` agrees compiled, interpreted and in TLC: halves Verified, below_two Refuted, 4 TLC states.
- `underflow` fails in every back end. TLC reports an `Evaluation` in `Invariant_remaining` at `values = <<1, 1>>`.

### R1: the interpreter change is sound
Dropping the `usize` special case in `interpret.rs` is exact for every form still admitted.

### R1: no admitted Model used the refused form
- I grepped every Model source for length `+` and `*`.
- The only hit is the fn-238.5 probe at `lengths.rs:40`, now the refused control at (40, 20).
- The Temporal activity uses no `Vector`.
- No golden, `.tla`, `.cfg`, `Cargo.toml` or lockfile changed.
- No Temporal text was added under `model/umpire`, and there is no new dependency (`thiserror` was already a workspace dependency of `umpire-tlc`).

### R2: what real TLC produced
- The nested-message parser removes "mismatched TLC message terminator" for TLC's own error output.
- Real TLC 2.19 runs with the new runner gave these results:

| Case | Result |
|---|---|
| Overflow in an invariant | `Evaluation`, property `Wide`, `x = 3` (the control) |
| Overflow in Next | `Evaluation`, next-state relation, state 3, `x = 2`, with positions |
| Overflow in an action property `[][x' * 1e9 # 7]_x` | `Evaluation`, property `Act`, `x = 3` |
| Overflow in `[](x * 1e9 # 7)` used as a PROPERTY | `Evaluation`, property `Prog` |
| Overflow in a multi-line invariant | `Evaluation`, property `Multi` |
| Tuple out-of-domain in an invariant | `Evaluation`, property `Dom`, `x = 1` |
| Invalid syntax | `Refusal`, "TLC execution error 3002" |
| Undefined operator | `Refusal`, "TLC execution error 3002" |
| A real invariant violation | `Refuted` (not misclassified) |
| Deadlock | Still refuses as before |

- The real-TLC control does drive TLC into "Overflow when computing 3*1000000000". The handoff's recorded unit replays match this real output.

## Findings

### 1. minor: literal-only `+`/`*` in an index context is refused, and the message blames a Vector length
- **Where:** `model/umpire/front/src/body/expr.rs:209-216`
- **Failure scenario:** These forms are refused with "`+` and `*` on a Vector length are refused: ...":
  - `check!(self.values.len() < 1 + 1)`
  - `check!(self.values.len() <= 2 * 2)`
  - `self.values.get(0 + 1).is_none()`

  Confirmed by the probe at `40:72`, `40:73` and `40:68`. The literals take the index type from the comparison or the `get` argument, so the check fires even though no length takes part. All three back ends agree on these forms. Over-refusing is within the owner's "cut features" rule, but the diagnostic misleads: in `get(0 + 1)` there is no length to "compare instead". It also contradicts the README sentence "`+` and `*` on a length refuse".
- **Suggested fix:** Either:
  - Exempt the case where both operands are `ExprKind::Int`: fold the result and refuse only if it exceeds `i32::MAX`, which keeps `get(2147483647 + 1)` refused.
  - Or reword the message to "on a usize length or index" and note the literal case in the README.

  Add `len() < 1 + 1` to the lift test table with the chosen outcome.

### 2. minor: evaluation failures during liveness checking still surface as an untyped refusal
- **Where:** `model/umpire/backends/tlc/src/runner.rs:477-481`, in `failure`'s fallback.
- **Failure scenario:** On real TLC, `PROPERTY Live` with `Live == <>(IF Assert(x < 2, "...") THEN x = 3 ELSE FALSE)` returns `Refusal("TLC execution error 1000: TLC threw an unexpected exception ... tlc2.tool.EvalException")`. TLC prints no 2103 positions and no 2111/2113 prefix there.
  - The spec says "overflow and other evaluation failures". Today's lifted modules are covered in practice: the only liveness they emit (`Maximal_*`, `~>` over `ENABLED scoped`, `emit.rs:351-363`) evaluates actions that exploration already evaluated on the same states, so an Assert or overflow there surfaces typed during exploration first.
  - A hand-written module, or a future lifted temporal operator over state arithmetic, would still get an opaque refusal.
- **Suggested fix:** Either:
  - Treat a 1000 error whose body names `EvalException` in a property run as an `Evaluation` of that run's property, with `trace: None`.
  - Or record this as an explicit limit in DESIGN.md's TLC paragraph and in the handoff's open items.

### 3. nit: a located 2111/2113 name isn't cross-checked against the run's property, and a multi-line location becomes the property name
- **Where:** `model/umpire/backends/tlc/src/runner.rs:483-488`
- **Failure scenario:** `failure` turns this input:

  `Evaluating invariant line 3, col 1 to line 4, col 2 of module M failed.`

  into `property = "line 3, col 1 to line 4, col 2 of module M"`, a location string instead of the run's known property. A single-line location resolves to whatever operator is defined on that line, even when it isn't the selected property.
  - `selected_violation` refuses both cases for 2110, so the two paths disagree.
  - I didn't see real TLC print a located 2111 in my probes. TLC named `Multi` and `Prog` directly, so this is latent.
- **Suggested fix:** Use `located(rest, tla).or(property)` when `property` is `Some`. When the located operator differs from the selected property, refuse with a located message, mirroring `selected_violation`.

### 4. nit: the tool-error control no longer pins the error kind
- **Where:** `model/umpire/backends/tlc/tests/controls.rs:258-259`
- **Failure scenario:** `error.contains("error") || error.contains("failed")` also matches "TLC evaluation error in ...". A regression that classified a SANY parse failure (3002) as an `Evaluation` would still pass.
- **Suggested fix:** `assert!(matches!(error, Error::Refusal(_)), ...)` before converting to a string. Do the same for the deadlock case.

### 5. nit: the R1 three-way control compares TLC against literals, not against the native run
- **Where:** `model/umpire/fixtures/interpreted.rs:245` and `:258-262`
- **Failure scenario:**
  - `result.states.len() == 4` and the TLC failing state `values = <<1, 1>>` are hard-coded.
  - The native state set and the native underflow state are never compared. Only the panic text is asserted (`:228`).
  - If the native and TLC failing states ever diverged, for example after a lifter change that moved the Assert, the control would catch it only if the literal happened to break.
- **Suggested fix:** Compare TLC's states to `Checker::explore(...)`'s state set, as `real_vector_native_and_tlc_states_and_properties_agree` does. Derive the expected underflow state from the native failure's state.

### 6. nit: refusal tests cover only `check!` claims
- **Where:** `model/umpire/lift/tests/vector.rs:24-49`
- **Failure scenario:** No test pins the refusal in a match guard, in a `let`-bound length, or through a pure helper. My probe confirmed that all three refuse at the right span today (`25:31`, `25:63`, `40:52`), but nothing guards them against regression. There is also no unit replay of a 2113 "Evaluating action property X failed." message, so the `prefix` arm for 2113 is untested.
- **Suggested fix:** Add the guard, `let` and helper forms to the table, plus one 2113 recording to `evaluation_errors_name_the_property_and_state`.

### 7. nit: the comment says a difference "stays in range"
- **Where:** `model/umpire/front/src/body/expr.rs:206-208`
- **Problem:** A difference can leave `0..=i32::MAX` (for example `1 - len()`). It stays inside TLC's 32-bit integers, and the lifter's `Assert` catches the negative case alike in every back end.
- **Suggested fix:** Reword to "stays within TLC's 32-bit integers; a negative difference fails every back end alike."

## Outside this task
On real TLC through this runner, a `<>` or `~>` property with any arithmetic on a variable, even `<>(x * 1 = 3)`, refuses with "TLC execution error 2280: In evaluation, the identifier x is either undefined or not an operator". This is not caused by this change: the code is unhandled before and after it, and the lifter doesn't emit such properties. It may be worth a separate look, since lifted liveness could one day contain checked arithmetic.
