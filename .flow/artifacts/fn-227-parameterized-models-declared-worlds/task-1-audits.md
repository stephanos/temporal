# Native World task audits

Frozen range: a76c3581620c39d8ccb760bc4af34d4c24b1d618..c3d85d9cf9.
Both axes were dispatched together against the committed range, read-only, without build/cache access.

### Correctness axis

- **model/umpire/suite/src/space.rs:15** (Should Fix, Conf 100): `Space<S: Associated>` breaks previously valid generic helpers such as `fn capped<S>(s: Space<S>) -> Space<S> { s.budget(10) }`; its new `Rc` storage also removes Send/Sync from legacy spaces. Preserve the unconstrained legacy Space/configuration surface and auto-traits while adding typed explicit-world storage. Evidence: walked.
- **model/umpire/dsl/macros/src/machine/declared.rs:472** (Should Fix, Conf 100): Progress helpers receive only Model declaration bounds. With unconstrained `Model<T>`/`World<T>` and `User<T: Token>`, generated `LeadsTo<T>` contains `Assumption<Action<T>>` without required `T: Token`, rejecting an otherwise valid bounded machine implementation. Propagate merged actor/World constraints into progress helper declarations and implementations. Evidence: walked.

Within-axis findings: 2. Worst tier: Should Fix. Selected-world retention and rejection checks preserve their behavior by inspection; existing test assertions remain intact. No builds or tests executed under the read-only dispatch.

### Standards axis

Files changed: 42 · Should Fix 1 · Consider 0 · Blocking: none possible (standards axis).

- **Should Fix — model/umpire/dsl/macros/src/machine.rs:505** (Conf 100): File crosses the 1,000-line threshold, growing from 938 to 1,018 lines (executed: `wc -l` plus diff numstat). Extract a cohesive group such as registration/signature admission into `machine/registration.rs`, preserving expansion output.

Walked: World derives reuse State’s value machinery; native suite binding reuses existing checker/search and retained ownership. The code-style review found no additional actionable duplication or unrequested public surface. README explicitly bounds delivery to native APIs and defers source/TLC support.

Builds and gates were not run, as dispatched.

Correctness: 2 findings, worst Should Fix.
Standards: 1 finding, worst Should Fix.

## Resolution evidence

All three findings were addressed before formal implementation review. Legacy spaces now store `()` by default with unconstrained configuration/From implementations; selected-world spaces carry `Rc<S::World>` through a sealed query binding. The retained suite pins unbounded generic helpers, non-Registered From conversions, Send/Sync, query generic argument counts, factory laziness, and non-Clone/non-Debug World behavior. The focused failing control and succeeding suite are retained in `.flow/tmp/fn2271-suite-storage-{red2,green2}.log`.

Progress helpers now receive the complete machine constraints. The generic control first failed for the missing actor bound (`.flow/tmp/fn2271-audit-progress-red.log`), then passed. Additional real Machine controls cover bounds authored only on implementations and alpha-renamed World/impl/actor binders; their red/green logs use `fn2271-{impl-bounds,world-binders,impl-binders,actor-binders}-*.log`.

Registration admission was extracted without changing legacy expansion bytes; `machine.rs` is 897 lines. The final combined Quick command passed (`.flow/tmp/fn2271-audit-quick.log`), and all 571 original artifact hashes plus 101 original UI stderr hashes passed (`fn2271-audit-{artifacts,ui}.log`). Original reports above are preserved verbatim; these resolutions do not replace the mandatory formal review.
