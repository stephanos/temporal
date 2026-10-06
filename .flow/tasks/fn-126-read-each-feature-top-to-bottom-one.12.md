---
satisfies: [R21]
---
# fn-126-read-each-feature-top-to-bottom-one.12 Cover composition stuck states and construct only needed witnesses

## Description
Close the original R21 coverage gap documented in task .9. The default stuck-state kind must inspect reachable states of machines and compositions, with the existing hole/unknown-behavior interpretation, and report every non-end state with no enabled action class and a shortest witness. A prior receipt's machine-only implementation was not an amendment of R21. Inspect available composition tables without silently excluding all compositions or using a failed refinement/property verdict to hide an otherwise constructible transition table. Keep genuine structural/build failures explicit through the existing lint/model diagnostics.

The standards audit also found that stuckStates eagerly constructs spellPath(table.PathTo(state)) for passing states. PathTo performs a fresh BFS. Count passing states without constructing witness arguments; compute the path only for a finding, preserving populations, messages, positions and shortest witnesses. Reuse the reader's existing composition/table/path machinery. If a minimal reader API is needed, keep it generic; do not implement the pending fn-124 package split here.

**Touches:** tools/umpire/lint/**, tools/umpire/model/**, model/README.md, model/ir/*.lint.json

Reader changes are limited to directly required composition/table/path helpers and tests. Lint acceptance changes require a justified finding on an existing deliberate control. Do not change a Model to silence a new finding. Report an actual Model bug to the conductor. Add a small composed deadlock regression and healthy/terminal/hole coverage. Measure the witness optimization on equivalent healthy inputs before claiming a speedup; no new profiling/caching framework.

Run focused red/green tests, current-IR lint and task-base read-only Go lint before per-task review. The expensive full model/Go/smoke gates run once after dependent task .13 at the next batch boundary. Reuse applicable .11 evidence for unchanged inputs and record this deferral explicitly.
## Acceptance
- [ ] Default stuck-state checks cover machine and composition reachable states. A composed non-end deadlock is reported even when no member is individually stuck; its shortest witness and declaration position are correct.
- [ ] Terminal, enabled and genuinely unmodeled/hole states do not produce false findings. Composition construction/refinement failures remain visible and do not become a blanket exemption for otherwise inspectable composition tables.
- [ ] PathTo and witness formatting run only for findings. Existing machine messages, populations and shortest-witness assertions remain intact; equivalent-input timing evidence supports any performance claim.
- [ ] Each newly exposed finding on current Models is accepted with a specific existing-design reason or reported as a Model bug; no Model behavior is fitted to silence the lint.
- [ ] Focused lint/reader tests, current-IR lint and read-only task-base Go lint pass. Full required batch gates are explicitly deferred to .13, and per-task implementation review reaches SHIP before verified Flow completion.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
