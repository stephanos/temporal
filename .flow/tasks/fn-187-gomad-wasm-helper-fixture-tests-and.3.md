---
satisfies: [R3]
---
# fn-187-gomad-wasm-helper-fixture-tests-and.3 Delete inline tests the corpora already cover

## Description
Delete corpus-covered inline tests across network, volume, namespace, World,
clock, broker and the helper only after assertion equivalence and recorded
coverage. Add hand-named cases for genuine gaps; regenerate Go-generated fixtures
through Go oracles. Cover duplicate Go WASI namespace, volume, broker and
network_v2 tests. Record per-module before/after production coverage and each
retained invariant's reason. Coverage loss on a model path blocks deletion.

### Current source and authority

R3 checkout: /home/agent/wt/wasm-fn187-functional, HEAD
95eed5c83a26eb1e93f2b1ba2f971e33bc1d51bc. Accepted R1/R2 index is unchanged:
staged binary diff SHA2560a343ce3549c1b7ed42146caf23c7800f2cd48f9a60c78ebfd3954036833d299.
Primary owner checkpoint5951d8bb96f41fdd1b146ee4a8cbb17613495aa0 contains unrelated
owner work; preserve it. Keep commits with the owner. No stage/commit/rebase,
push, PR, CI or migration action is authorized. Integrate only after required
gates/review and exact source reconciliation. R3 is still in progress.

### Pre-repair verification

Evidence root: /home/agent/wt/evidence/fn187/r3-current/.
Session70048 terminated0: post-delete-conservation-2026-10-19.log ends with all
five current-source full comparisons passing. New phases are
before-post-delete-conservation-2026-10-19 and r3-post-delete-conservation-2026-10-19.
Each mode executes the complete original baseline and candidate full/corpus
arms, candidate retained invariants, exact compiled roster/partition checks,
fresh bounded cursor proof and strict production-line conservation.

Baseline-covered production lines preserved: default15626, diagnostic-profile14971,
diagnostic-fuel-off14963, guest-profile15103, combined guest/fuel-off14276.
Every loss/union-gap list is empty. Default additionally passes baseline/candidate
Go full+corpus, candidate Go invariants, post-Go Rust exports and Go statement
conservation. Mode-specific frozen tools, source snapshots, object hashes/build
IDs, raw profiles and command/exit receipts remain under those phases.

Current-source six-mode Rustfmt/strict Clippy passes (session76393,
post-delete-lint-rust-2026-10-19.log);69 coverage-tooling tests pass (session34557,
post-delete-tooling-2026-10-19.log). No new exemptions. All526 original contracts,
215 original helper case bytes,407 prior helper inputs,183 original network
inputs and CLOSED42 remain preserved. Exact source-bound feature cases are
additions, not rewrites of the original contracts or Go oracles.

### Final public-root removals

Seven public roots are removed: four files,295 deletions; every retained source
line is byte-identical to archived preimages. Packet
remaining-public-deletion-named-entry-2026-10-19/ retains exact whole-file
preimages, patch and assertion maps; application receipt is
seven-public-removals-application-2026-10-19.json. All142 original private binding
keys/reasons remain unchanged. No public scenario became a private waiver.
Five feature-private roots and two guest host invariants have explicit separate
reasons/source bindings in the frozen mode tools.

Independent periodic correctness/standards audits found0 findings and verified
the exact8,388,706-byte cluster stream,1,259,101-byte400-operation/20-chunk storage
payload, all original assertions and negative controls. Verbatim reports are in
primary .flow/artifacts/fn187-seven-public-removals-review-2026-10-19.md. This
scoped source audit is not the final task-wide implementation-review verdict.

Combined profiling baseline reproduced an unnamed-entry fixture race3/8 times.
The only fixture repair names its WAT entry $hot_entry: no instructions,
assertions or budgets change. Exact byte-reconstruction proof and20/20 focused
passes remain under guest-symbolization-*. Both current guest-mode full baseline
and candidate executions now pass. Earlier pre-naming/pre-deletion coverage
certificates remain historical; failed attempts are retained, not relabeled.

The approved namespace-readiness source selector now runs both predicate suites
in primary and candidate; outer execution passes and direct execution proves47
subtests without skips. Earlier5394-entry primary host freeze predates these two
test-file edits and is not current-source qualification.

### Remaining acceptance and constraints

Full working-tree correctness review found3 blocking assertion gaps: discarded
idle return time1000, missing constructor errors.Is(world.ErrCapacity), and lost
exact netmodel.ErrResourceExhausted identity. Verbatim reports and accepted
dispositions are in primary .flow/artifacts/fn187-r3-working-tree-review-2026-10-19.md.
All three are repaired after intended real-consumer RED and focused GREEN.
Four production-source overlay mutations reject changed idle returns and
same-text unrelated/wrapped errors without modifying actual source. All eight
normal corpus/control roots pass without skips. Two Go regenerations are
byte-identical. The exact61-case delta proof preserves all prior observations
and shared network Expected blocks; Go sentinel metadata is outside Expected.
Fresh preservation retains all526 original contracts and215 original helper
cases. Source rereview resolves all three findings with no new findings.

Owned session25445 terminated exit0 after fresh five-mode full/corpus/invariant/cursor/conservation
verification under before-typed-assertion-replay-2026-10-19 and
r3-typed-assertion-replay-2026-10-19. Log: typed-assertion-replay-2026-10-19.log.
All modes pass; production-line loss/union gaps are zero at15626/14971/14963/15103/14276
baseline-covered lines. Default Go full/corpus/invariants and statement conservation
also pass with zero losses/gaps. Session36566 subsequently verifies all ten phase
source/tool/object pins against current inputs. Authenticated frozen binaries are reused only with exact
unchanged Rust compiler inputs; test results and raw profiles are never reused.
Reviewed manifest SHA256a2a0fce819a1761d392aba7fe443de7e140065a240bb80199f7815b70ddbe38d
pins all three Go sources, fixture delta proof and each executed mode tool map.
Both replay guard findings are fixed and source-rereviewed. Input drift controls
reject altered manifests/tools before phase creation. All five earlier passing
phases remain scoped to exact pre-repair source, not rebound. Fresh repaired-source
full gates have passed; bounded final independent acceptance rereview is pending.

The canonical CLI's fan-out51eff737bb014145b605e88037d1e2a9 only reviewed an empty
committed range and explicitly did not certify R3. Do not use its mechanical
SHIP receipt for acceptance. Owner-only project commits prevent a checkpoint;
actual working-tree audits are the documented fallback, not an empty-range pass.

Exact primary integration remains before overall spec completion.
Use the per-module coverage-by-module.json and line/statement conservation
reports above, assertion packets and retained-root bindings in the handover.
Preserve failed foreign-root and LLVM diagnostic attempts. Exact registry-only
symbol/source exceptions do not admit unknown/project/nonzero mismatches.
Never rerun old hardcoded source_inventory.py or prepare_functional_selectors.py.

Use pinned Rust1.99.1/LLVM23.1.1 and stock Go1.27.1; sanitize inherited overrides.
Root serializes all compiler/runtime/LLVM work. Ordinary helper publication must
not overwrite a Cargo hardlink. WASM resource-sensitive checks are authorized
with unchanged admission/budgets. Native remains deferred under fn128/fn149;
fn185 profiling and ignored benchmarks are outside scope. R4 module restructuring
follows completed R3; R5 keeps original baseline counts, same-host canonical test
wall-time medians/ranges and full R1-R5 handover. Neither spec is complete.

**Touches:** tools/gomad/internal/backends/wasm/wasmhost/src/**,
tools/gomad/internal/backends/wasm/wasmhost/tests/**,
tools/gomad/internal/backends/wasm/wasi/**,
tools/gomad/internal/model/network/**, tools/gomad/internal/model/volume/**,
task-owned coverage scripts/evidence and documentation
## Acceptance
- [ ] Coverage report before and after per module retained; no production line loses coverage
- [ ] Every deleted test names the corpus cases that replace it
- [ ] Remaining unit tests listed as invariants with reasons
- [ ] Go-side duplicate tests handled the same way

## Done summary
Deleted corpus-covered inline Rust and duplicate Go tests, with deleted-assertion
maps, named replacement cases, per-module coverage reports and retained private
invariant reasons. All142 original private binding keys/reasons remain unchanged.
Seven final public-root removals preserve every retained source line; original526
contracts and215 helper case bytes remain unchanged. Added Go-authored cases close
coverage gaps without changing Go production oracles or shared Expected blocks.

Taskwide review caught three assertion losses beyond line coverage: actual broker
idle return1000, constructor errors.Is(world.ErrCapacity), and exact network
resource-exhaustion identity. Intended RED, repaired GREEN and four rejected
production-source overlay mutations prove restored checks. All eight normal
corpus/control roots pass without skips; two regenerations are byte-identical.

Fresh repaired-source five-mode replay terminated0 (session25445): full/corpus/
invariant, exact roster/disjoint partition, cursor mapping and strict production
line conservation pass. Baseline-covered lines15626/14971/14963/15103/14276 have
zero losses or union gaps. Default Go full/corpus/invariants and statement
conservation pass with zero losses/gaps. Session36566 verifies all ten source/
tool/object pins against current inputs. Rust compiler inputs are unchanged;
authenticated frozen objects are explicitly reused, never old outcomes/profiles.
Exact reviewed Go/proof/tool-map pins and rejection controls protect that reuse.
Six-mode Rustfmt/strict Clippy and69 tooling tests passed; no new exemptions.

Evidence root: /home/agent/wt/evidence/fn187/r3-current/.
Fresh phases: before-typed-assertion-replay-2026-10-19 and
r3-typed-assertion-replay-2026-10-19; log typed-assertion-replay-2026-10-19.log.
Assertion map, preservation, mutation and review pointers are consolidated in
.flow/artifacts/fn187-r3-working-tree-review-2026-10-19.md and
.flow/artifacts/rust-hygiene-2026-10-18.md.

Canonical CLI fanout inspected an empty committed range; its mechanical SHIP
receipt is explicitly rejected. Actual-worktree independent audits are the
documented fallback; standards0ShouldFix/1Consider is nonblocking. Final bounded
acceptance disposition is recorded in the working-tree review artifact.
Final correctness verdict: SHIP for R3,0 findings; all three assertion and both
provenance findings resolved. Reviewer and writer are the same model family.
stage: impl-review - ran (actual-worktree fallback; canonical empty range rejected) (model: gpt-6.1-sol)

No staging or commits: owner reserves project commits. Accepted R1/R2 staged
binary diff remains SHA2560a343ce3549c1b7ed42146caf23c7800f2cd48f9a60c78ebfd3954036833d299.
Primary integration, R4 module restructuring/identity qualification, R5 counts/
same-host canonical timing and combined fn189 qualification remain required
before overall completion. No native, fn185 profiling, ignored benchmarks,
push, PR, CI or migration actions were performed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: run-typed-assertion-replay-2026-10-19.py: five modes exit0, fresh full/corpus/invariant/cursor/conservation; default Go full/corpus/invariants and statement conservation exit0, phase_snapshot.verify: all ten current source/tool/object pins pass, session36566 exit0, typed-assertion-controls RED/GREEN and four overlay mutations; normal eight corpus/control roots pass with zero skips, six-mode Rustfmt/strict Clippy and69 tooling tests pass; Rust inputs unchanged, independent actual-worktree correctness SHIP; standards0ShouldFix1Consider; primary integration remains before spec completion
- PRs: