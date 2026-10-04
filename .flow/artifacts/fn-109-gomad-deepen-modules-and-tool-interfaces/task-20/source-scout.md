# Architectural guidance source scout

Read-only scout of current source at HEAD
`0dd05b313acd0986312da7fd3159520e6a21f1bf` plus task-13/14/15 candidates.
Requested Codex thinking scout: gpt-6.1-sol high.
Tier: session (jev-unavailable(no_key)); explicit routing authoritative.
No edits, tests, builds, generation, Flow mutations or native qualification.
Task 20 remains unstarted; describe tasks 16-19 only after final source exists.

## Already-correct statements

Supported platforms: SPEC:174-176, ARCHITECTURE:17-23,732-734, README:13-19,
CLI:60 and TUTORIAL:111-112; descriptor version.json:9-12 is authoritative.
Choice tracing/exact replay/exploration: SPEC:44-54,247-249,
ARCHITECTURE:252-287, README:132-165, CLI:141,186-196,254,
TUTORIAL:376-402. Backend fidelity: SPEC:104-110, ARCHITECTURE:98-113,152-155,
README:1206-1222 and TUTORIAL:350-356. Vocabulary: SPEC:24-170,
ARCHITECTURE:4-5, README:1196-1204; deleted glossary terms live in SPEC.
ARCHITECTURE:793-797 separates research from implemented tracing/exploration.

Reuse fn-111/task-3/acceptance-summary.md:18-23,49-55 for current claims;
parent task-1/task-2 receipts and hashes are explicitly historical. Current
content and source still need verification at task 20's final freeze.

## Stable owner guidance gaps

- ARCHITECTURE after campaign selection or Runner containment: campaign options
  group serialized intent separately from runtime construction. Actual source
  `runner/coordinator.go:21-39` uses coordinatorConfig embedding campaignOptions;
  go-interface-changes.md:74-80 describes an older concrete shape. Cite source,
  not the old shape. Existing IDs: CAMPAIGN.SELECTION, CAMPAIGN.EXECUTION,
  PLATFORM.IDENTITY.
- Preparation discussion: internal/preparation composes complete preparation
  and capability inspection, attaches adapter identities and validates results.
  Inspection owns its private workspace until Close; durable destinations keep
  their campaign/portable owners. Source preparation.go:50-91, inspection.go:13-66;
  task-7/task-8 design decisions. IDs: TARGET.PREPARATION, TARGET.CAPABILITY.
- Host tooling boundary: target/internal/gocommand retains supervised processes,
  complete bounded structured output versus bounded diagnostics/full hashes,
  cancellation/watchdog/cleanup distinctions. Source command.go:34-42,70-115;
  task-9 decision.
- Installation discussion: toolchain/installation Layout and validated
  Description own layout and launcher/key/pinned-build knowledge; stable adapter
  paths remain identity inputs. Source installation.go:21-126, task-10 decision;
  IDs PLATFORM.INSTALLATION, PLATFORM.IDENTITY. README:57-89 already covers
  user-facing resolution.
- Capability discussion: collection, pure evaluation and linked projection
  remain separate; neutral sourceinventory hashing serves target/adapters.
  Source capability_collection.go:37-104, capability_evaluation.go:18-94,
  capability_linked.go:25, capabilitypolicy/policy.go:72. ID TARGET.CAPABILITY.
- Runner/artifact discussion: document intentional executor-injection migration
  and usable preparation/replay substitutions. Detached Artifact differs from
  owned Opened, which pins its root, copies manifests and requires closure.
  Sources artifact/open.go:19-33,84-120, store.go:62 and interface-change inventory;
  IDs EVIDENCE.ARTIFACT, EVIDENCE.REPLAY, TARGET.PREPARATION.
- Binary protocol ownership: one versioned timewire.json generates host/runtime
  codecs; runtime descriptor I/O, native timers and quiescence stay handwritten.
  Source schema:2-28 and task-13 handover. IDs RUNTIME.TIME, SIMULATION.BACKENDS,
  MAINTENANCE.GOVERNANCE. Source implementation is not native acceptance.

## Later guidance and evidence cautions

Task 16 belongs under process arbitration; task 17/18 under backend/model
sections, with explicit mapping/mount/capacity/fidelity distinctions; task 19
under Maintenance gates. Existing design or scout findings do not prove delivery.
Only fn-109's MILESTONES tracking row/status may be updated by task 20.

Keep capability support (SPEC:202-208), repeatability (README:277-280), verified
exact replay (README:285-294) and expectation matching (README:312-342,
CLI:309-328) separate. D12 remains open, untraced/capacity suites retain their
dispositions, and clock escapes remain limitations. D14 keeps its recorded
Darwin correction; see disposition-reconciliation.md.

ARCHITECTURE:209-227 still describes a separate forward-clock offset, while
README:787-810 documents the shared-clock D26 candidate. This discrepancy needs
current-source/evidence reconciliation with D26's owner, not a claim that a
historical fn-111 hash proves current guide correctness or that D26 qualifies.
No fn-109 measurement supports new performance or native-support claims.
