---
satisfies: [R7]
---
# fn-227-parameterized-models-declared-worlds.6 After .5: DESIGN.md parameterized worlds write-up and readiness table closure

## Description
Record R7's delivered surface and measured evidence after .5 and its transitive
support tasks. Consolidate documentation and final gate evidence in this existing
task; no separate finalization tasks or new behavior. MILESTONES remains root-owned.
On 2026-10-21 the timeout and cooperative-cancellation write-up moved to
fn-237-activity-timeouts-heartbeat-and.5; this task covers parameterized worlds,
inputs and retries only.

**Size:** M. Adapt two existing authoring/design documents to the completed
vertical fixtures and actual live proof, then verify the complete public example
and the existing final gates. No new framework or source admission.
**Dependencies:** Retain fn-227-parameterized-models-declared-worlds.5.
**Files:** `model/DESIGN.md`, `model/README.md`; focused authoring/doc-test fixture
only if needed to compile the documented parameterized example; task-owned Flow
verification artifacts through the normal lifecycle.
**Touches:** [model/DESIGN.md, model/README.md, model/umpire/dsl/tests/**]

### Approach

1. Reanchor all completed task receipts and actual implementation spellings.
   Replace the stale unit-only and unsupported readiness statements in place.
   Keep DESIGN within its 1000-line limit, preserving unrelated architecture
   and the separate fn-232 failed certification result.
2. Describe declared World ownership, typed callbacks, computed starts, Finite,
   field-level domain spellings, explicit selected-World domains and checked
   catalog limits. Explain checked scalar construction without claiming generic
   World/state validation. Compile the one README parameterized example using
   the established doc-test or focused DSL fixture path.
3. Record the activity World/state table and retry laws. Describe the retained
   lifecycle projection at the retry checkpoint (one attempt, direct deadline
   actions, atomic cancellation) rather than claiming old activity byte equality.
   Preserve the request-evidence and transactional admission bounds, including
   that request timeout fields are recorded but not yet admitted. Timeouts,
   deadline priority and zero/reset laws, cooperative cancellation and the
   fair-worker assumption belong to fn-237.5.
4. Obtain native/TLC counts per actually selected fixture World and the actual
   live retry/Describe evidence from retained outputs. Execute model-time on the
   final tree and retain its full output. State soft-target misses honestly; a
   zero command exit does not mean a performance target was met.
5. Mark readiness rows delivered only when their owning tasks and actual proof
   establish it. Keep source refusal boundaries and open unrelated specs explicit.
   Retain the final full-gate evidence for the parent's completion review.

### Investigation targets

Required:

- `model/DESIGN.md` parameterized readiness and current capability boundaries.
- `model/README.md` Writing a Model examples and backend execution instructions.
- Native/source `parameterized_world`, `action_inputs` and `phased_rejection`
  fixtures and their completed task receipts for exact counts and public APIs.
- `model/temporal/feature/activity/{model.rs,tests.rs,temporal.rs}` and the
  compatibility harness for actual state/action/fact projection and live binding.
- The .5 live slice/gap/privacy evidence and its exact embedded guest hash.

### Verification

Quick from model/: `mise exec -- cargo test -p umpire-dsl --doc`, plus the focused
compiled README example if it lives in a DSL test. Run formatting, documentation
link checks and DESIGN line-count verification using existing commands.

Full from root: `make model-check`, `UMPIRE_BACKENDS=require make model-check`,
`make model-time`. No global model-update, protocol/dependency/panic-policy change
or stale counts. Preserve final timing output and prior evidence, including any
unmet soft targets, without claiming fn-232 certification.
## Acceptance
- [ ] DESIGN documents the actual parameterized surface and the retry-era activity table, and
  delivered readiness statements are backed by completed native/source/live tasks.
- [ ] One compiling README example covers the selected World, computed starts,
  field-level inputs and explicit suite binding without fabricated defaults.
- [ ] Actual native/TLC counts per selected World, live retry proof and final
  model-time output are retained with all misses and refusal boundaries explicit.
- [ ] DESIGN remains within 1000 lines, existing documentation/formatting checks
  and full gates pass, and unrelated evidence/goldens and fn-232 status are preserved.
## Done summary
DESIGN and README now document delivered declared Worlds, typed callbacks, computed starts, finite/domain inputs and catalog admission, selected native/TLC counts, and the retry-era activity World/state table. The exact README block compiles as a DSL fixture with explicit selected-World binding; its synchronization test failed against the old block before replacement, then both example tests passed.

Baseline: green focused declared_world/finite_domains tests (9 cases); the requested DSL doctest command passed with zero doctests and is not presented as example coverage. Initial cwd/discovery mistakes, the intended README mismatch and intermediate lint failures remain explicitly non-green in task6-final-verification.json and hashed compressed logs. DESIGN is 969 lines; local Markdown links, fmt, strict workspace clippy, docs and all 15 original activity artifacts pass. No unrelated golden or production file changed.

One UMPIRE_BACKENDS=require make model-time passed all 17 stages on source checkpoint cb8b9a76d2: gate 582.209 s missed the 60 s soft target, inner 6.112 s met 10 s, outer 588.537 s. These are shared-host observations, not isolated performance measurements. Full verbatim output, per-stage timings, raw/gzip hashes and exact command exit are retained in .flow/artifacts/fn-227-parameterized-models-declared-worlds/task6-final-verification.json and task6-log-integrity.json. All 215 production-surface files and the f832a33c3024a8752976d146be839b62a95afde4790fa0db60f1dac9004b4e7e guest match .5; its actual seven-scenario retry, gaps and privacy evidence is reused with hash and full-history proof. Historical failed fn-232 observations and later fn-232/fn-239 certifications remain at their own pins.

stage: impl-review - ran. First fanout NEEDS_WORK identified only the pending final evidence; after the one full gate, the same-cycle codex:gpt-6.1-sol:high follow-up returned SHIP, all R7 requirements met and no remaining findings. Writer and explicitly configured reviewer are the same GPT family. Mechanical evidence population required no nontrivial memory capture.
GATE_SKIPPED:unittest:green-receipt 8e84f41c - final required model-time reused; subsequent history changes only .flow/

Tier: session (jev-unavailable(no_key)); AGENTS implementer override above.
stage: plan-sync - skipped(config: planSync.enabled=false). Tracker sync is inactive; the post-done lifecycle audit uses the actual work.done event and .6 claim timestamp 2026-10-22T23:29:13.977542Z. Parent completion review/closure and MILESTONES remain conductor-owned. fn-237 and fn-238.5/.6 stay explicitly deferred. All owned long-running commands have exited.
## Evidence
- Commits: cb8b9a76d20410339fda4e2efa6d086ebd5012df, 8e84f41cf95bc67f24ce0f5536ae764a42292191, e4f17cbd5fb61a91585d239ee46591e34cfbf6d7
- Tests: mise exec -- cargo test -p umpire-dsl --doc, mise exec -- cargo test -p umpire-dsl --test declared_world --test finite_domains, mise exec -- cargo test -p umpire-dsl --test readme, mise exec -- cargo fmt --all --check, mise exec -- cargo clippy --workspace --all-targets --all-features -- -D warnings, python3 .flow/artifacts/fn-227-parameterized-models-declared-worlds/task6-check-docs.py, python3 model/temporal/feature/activity/compatibility/verify.py, UMPIRE_BACKENDS=require make model-time, GATE_SKIPPED:unittest:green-receipt 8e84f41c - final required model-time reused; subsequent history changes only .flow/
- PRs: