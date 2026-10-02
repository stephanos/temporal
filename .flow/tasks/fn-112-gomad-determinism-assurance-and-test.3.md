---
satisfies: [R3, R4]
---
# fn-112-gomad-determinism-assurance-and-test.3 Record a runtime-state digest at each choice point in a diagnostic trace

## Description
Runtime and wire half of the localiser (R4): an opt-in diagnostic trace record carrying a state digest at every choice point. This is the spec's early proof point. Runner plumbing and the differ are task 4.

**Size:** M
**Files:** `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go`, `tools/gomad3/choice/schema/choicewire.json` and templates, generated `wire_generated.go` (host and overlay), `tools/gomad3/toolchain/version/version.json` if a new overlay file is added
**Touches:** [tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go, tools/gomad3/choice/schema/**, tools/gomad3/choice/internal/wire/**, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadchoicewire/**, tools/gomad3/toolchain/version/version.json, tools/gomad3/internal/gomadtool/generation/protocol/**]

### Approach
- First step (R3): re-anchor assessment finding Q1 against the current record struct and mark it confirmed, changed, or refuted.
- Reuse the choice append path; add a separate record kind on its own inherited descriptor with its own byte bound, read at env intake beside the existing choice variables.
- Digest fields: seeded draw counters, allocation count, GC cycle and phase, virtual time, run-queue length. Use only state the overlay can read without editing a collector file; drop a field that needs one and say so.
- Recording must not allocate on the Go heap or draw from the seeded stream.
- Define the layout in the wire schema and regenerate with `make -C tools/gomad3 generate`; never hand-edit generated files.
- Add an overlay-only, diagnostics-only environment switch that perturbs one draw at a chosen ordinal, for the fixtures in tasks 4 and 5. No new patch hunk.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:313-347` — record struct and append path
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:199-253` — env intake for trace descriptors
- `tools/gomad3/choice/schema/choicewire.json` — wire schema
- `tools/gomad3/internal/gomadtool/generation/protocol/protocol.go:344-357` — codec generator

**Optional** (reference as needed):
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:693-743` — replay divergence reporting
- `tools/gomad3/toolchain/runtime/go1.27.1.patch:614-644` — runtime rand helpers whose counters the digest reads

### Key context
- Choice records are fixed-size in one mapping capped at 64 MiB; the diagnostic trace must not share that budget.
- fn-110 tasks 2-4, fn-109 task 13, fn-114 C2/E3/E4, and fn-105 D26 edit the same files (spec Open Questions 2). Check their state before starting and rebase onto whichever landed.
- Any runtime edit changes the toolchain build key and the choice implementation digest, both part of execution evidence. Cross-build comparison therefore uses the behavioral projection in the acceptance list, never raw evidence digests.
## Acceptance
- [ ] Finding Q1 re-anchored with file and line, marked confirmed, changed, or refuted
- [ ] With diagnostics on, every choice point emits a digest record with the listed fields; dropped fields are named with the reason
- [ ] With diagnostics off, the core qualification set matches the pre-change toolchain on a behavioral projection: stdout and stderr hashes, I/O transcript hash, World identity, outcome, virtual time, peak goroutines, and choice-trace decision content. Toolchain build key, choice implementation digest, and identities derived from them are the allowed differences, and the new identities are checked to be recorded correctly
- [ ] Within the new toolchain identity, canonical bytes with diagnostics off are identical with and without the diagnostic code path compiled in use (flag absent versus never requested)
- [ ] A workload that qualifies with diagnostics off also qualifies with them on
- [ ] Recording performs no Go-heap allocation and no seeded draw, shown by a test
- [ ] Diagnostic-trace overflow stops the target with a typed failure
- [ ] `make -C tools/gomad3 validate test-toolchain test-runtime overlay-test` pass on darwin/arm64; linux status recorded
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
