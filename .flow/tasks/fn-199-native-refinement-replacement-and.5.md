---
satisfies: [R6]
---
# fn-199-native-refinement-replacement-and.5 Verdict kinds: resource limit, unsupported, declaration error; Report coverage

## Description
Add the three verdict kinds, make `verifyAll` report every refinement, replacement and composed progress result, and keep the six samples' reports unchanged where they declare none.

**Files:** model/framework/checker/Result.scala, model/framework/checker/NativeCheck.scala, sample tests.

**Quick commands:** `mise exec -- scala-cli test --server=false model/framework/checker model/framework/samples --require-tests`

## Acceptance
- Every existing consumer of `Verdict` compiles unchanged.
- Sample reports byte-identical where no refinement or composition is declared.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: