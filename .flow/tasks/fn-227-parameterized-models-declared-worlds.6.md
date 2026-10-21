---
satisfies: [R7]
---
# fn-227-parameterized-models-declared-worlds.6 After .5: DESIGN.md parameterized worlds write-up and readiness table closure

## Description

Record R7's delivered surface and measured evidence after .5 and its transitive
support tasks. Consolidate documentation and final gate evidence in this existing
task; no separate finalization tasks or new behavior. MILESTONES remains root-owned.

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
3. Record the activity World/state table, retry/deadline priority and zero/reset
   laws, cooperative cancellation and exact-action fair-worker assumption.
   Describe the retained lifecycle projection rather than claiming old activity
   byte equality. Preserve the request-evidence and transactional admission bounds.
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
- Native/source `parameterized_world`, `action_inputs`, optional-value/fact
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

- [ ] DESIGN documents the actual parameterized surface and activity table, and
  delivered readiness statements are backed by completed native/source/live tasks.
- [ ] One compiling README example covers the selected World, computed starts,
  field-level inputs and explicit suite binding without fabricated defaults.
- [ ] Actual native/TLC counts per selected World, live retry proof and final
  model-time output are retained with all misses and refusal boundaries explicit.
- [ ] DESIGN remains within 1000 lines, existing documentation/formatting checks
  and full gates pass, and unrelated evidence/goldens and fn-232 status are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
