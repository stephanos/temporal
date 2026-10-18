---
satisfies: [R6]
---
# fn-190-models-are-plain-scala-state-as-vars.15 DSL completion: native runtime moves to the product checker under fn-196.1

## Description
**Carried by fn-196.1 (owner, 2026-10-17):** the destination is the product checker module `model/framework/checker`, package `framework.checker`. Execute this move once through fn-196.1, not through a separate test-support implementation. The measurement decision in fn-196.3 still gates Go-checker retirement under fn-196.4; this move alone grants no Go/IR/Temporal integration credit.

**DSL completion phase (owner, 2026-10-17).** Move native execution (`effectOf`, `guardHolds`, input-effect adapters, finite/domain/state enumeration and budget runtime, `NativeCheck` and var snapshots) out of `model/framework/dsl`. Keep declarations and the references the plugin reads in the DSL; expose the native questions and unified result through the product checker as fn-196.1 specifies. Preserve the author's declaration surface, and make tests import the checker explicitly. The earlier test-support destination and its later deletion by an IR evaluator are superseded by fn-196; they are not acceptance requirements for this move.

**Size:** L

### Verification
- No production source in model/framework/dsl invokes an effect or guard callable (grep recorded).
- The native dsl/realize/samples suites pass with the same test count; lint RuleSuite and boundary suites pass.

## Acceptance
- [ ] Native execution lives outside framework.dsl in the product checker module, as fn-196.1 specifies
- [ ] Native suites pass with unchanged counts
- [ ] fn-196.1's source and verified committed closure carry the move; this tracking task is closed afterward, without a second implementation

## Done summary
Carried by fn-196.1, whose source 0dd5737301eaf2193d3ae89ea71a3dac2110c7ae and verified closure 58fa14a2056cc50b9d120c0ce1185ad6e757f5e5 are committed on umpire. The native evaluator, domains/inputs/budgets, snapshots, claims and composition runtime now live in production framework.checker; the DSL retains declarations and callable references. No second implementation or test-support destination was introduced.

The conductor read the handover and check record, verified fn-196.1 done and its spec structurally valid, and compared original test-name multisets: all653 baseline tests remain, with four additive checker controls (657 total). Framework checks46, actual lint31, corpus162, declaration-only compilation, fresh core/plugin and combined/separate-library realization sample compilations passed. Exact commands, observed exits and intermediate reds are retained in .flow/tmp/checks/fn1961/record.md and fn-196.1's committed evidence.

This closes only the carried runtime move. Parent R6's deferred integration obligations, full IR/export proofs, Go retirement and disabled Temporal work remain open. Independent review is owner-deferred to the phase batch; no SHIP verdict is asserted.

stage: impl-review - skipped(policy: independent review deferred to the phase batch by owner)
stage: plan-sync - skipped(config: planSync.enabled=false)
## Evidence
- Commits: 0dd5737301eaf2193d3ae89ea71a3dac2110c7ae, 58fa14a2056cc50b9d120c0ce1185ad6e757f5e5
- Tests: Carried verified implementation: fn-196-the-scala-checker-is-the-oracle.1; exact commands and observed exits in .flow/tmp/checks/fn1961/record.md, Native657: original653 test-name multiset preserved, four additive checker controls; framework checks46; actual lint31; corpus162; fresh declaration/core/plugin/sample gates passed, flowctl show fn-196-the-scala-checker-is-the-oracle.1 --json: done; flowctl validate --spec fn-196-the-scala-checker-is-the-oracle --json: valid, no errors or warnings
- PRs: