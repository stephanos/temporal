---
satisfies: [R8, R9]
---
# fn-189-the-reify-plugin-keeps-only-the-body.8 Captured[F] capture arguments; remove post-pickler retargeting and per-arity overloads (after fn-174.8)

Touches: model/framework/dsl/Reify.scala, model/framework/dsl/Syntax.scala, model/framework/dsl/Machine.scala, model/framework/dsl/Budget.scala, model/framework/reify/plugin/ReifyPlugin.scala (CapturePhase, RuntimeCapturePhase, RulesPhase, EndpointsPhase retention)

## Description
**Note 2026-10-18 (one-language series):** Still after fn-174.8; counts are read from the in-process lift (fn-201).

Cross-spec prerequisite: fn-174.8 has retired the old TASTy reader, so TASTy no longer needs the original source trees. Then: capture parameters are typed `Captured[F]`; a `given Conversion[F, Captured[F]]` in the DSL keeps authors' spelling and throws without the plugin; the capture phase wraps the argument in place before pickler as `Captured(value, root, chunks)`. Delete RuntimeCapturePhase, the retention EndpointsPhase and RulesPhase, the StickyKey payload attachments, and the hidden overloads capturedRule0..n, capturedBinding0, capturedRebind, capturedExtend, capturedDisabledRules, retainOrigin and retainDerivation. The plugin ends with two phases: reifyStore and reifyCapture. Re-record the R9 counts.

**Size:** M

### Verification
- The six samples lift to the same IR as under R7.
- A capture whose argument is neither a function literal nor a reference keeps the refusal `a capture applies one marked function parameter at its call site`.
- Focused dsl, reify and export suites pass; the R9 counts show two phases.
## Acceptance
- [ ] Capture parameters are Captured[F] and are wrapped in place before pickler.
- [ ] RuntimeCapturePhase, both retention phases and every per-arity hidden overload are gone; two plugin phases remain.
- [ ] Samples lift to the same IR as under R7; the counts receipt is updated.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: