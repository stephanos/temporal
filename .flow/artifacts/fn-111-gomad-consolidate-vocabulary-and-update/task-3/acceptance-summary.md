# fn-111 current documentation acceptance

Refreshed on 2026-10-02 against HEAD `67dbe02666afd68c064c6c3cb9197b03d4664687` and the uncommitted
six-document tree hashed below. The comparison baseline remains
`29917069e089dc0739ec091b18e99161245b9bd5`. [guide-audit.json](guide-audit.json) binds 91
inputs, the three audit scripts, binaries, probe source, and toolchain key;
its SHA-256 is `f6eebf2e9d42d093c9b3944617fc7a0f2987a7e2c45a745a3bde980e514dac95`.

The final [verification-result.json](verification-result.json) records the exact
command, exit 0, 4.506 seconds, counts, and changed paths. The command was:

```sh
python3 .flow/artifacts/fn-111-gomad-consolidate-vocabulary-and-update/verify-guides.py /tmp/gomad-fn111-bin /Users/stephan/.codex/plugins/cache/flow-next-marketplace/flow-next/4.5.1/scripts/flowctl .flow/artifacts/fn-111-gomad-consolidate-vocabulary-and-update/task-3
```

| Requirement | Current verification |
| --- | --- |
| R1 | vocabulary-audit.json: 25 original entries assessed; 24 current concepts retained; Parity Case remains historical. |
| R2 | All 123 semantic identifiers preserved in order, including 28 command-table identifiers; SPEC owns vocabulary; no glossary file remains. |
| R3 | vocabulary-audit.json distinctions and guide claim matrix retain target/preparation, campaign/plan, trace/tape, complete/prefix replay, Backend/Fidelity, and World boundaries. |
| R4 | Platform descriptors and guide names match darwin/arm64 and linux/amd64; source checks cover tracing, exploration, backends, and current workload expectations; strict/forward clock probe observed all expected readings on darwin/arm64. |
| R5 | 30 indexed commands, 68 documented flags, 137 command examples, 31 Make examples, and 98/98 source claims pass; diagnostics constraints, diagnostic-diff, simulation gate, and divergence statuses are checked. |
| R6 | Tutorial terminology, replay distinctions, clock behavior, confidence-failure status, and preserved GOMAD_NEXT roadmap pass. |
| R7 | 89 local links/fragments across the six documents resolve; typed fences balance; stale vocabulary absent. Historical receipts remain accessible. |
| R8 | This summary, input/document/binary/toolchain hashes, CLI controls, companion checks, and whitespace receipts bind the checked tree. |

[cli-controls.json](cli-controls.json) retains actual current-binary comparisons:
equal diagnostics return 0, different diagnostics return 1, malformed diagnostics
return 2; diagnostics with forced-prefix exploration return 2. It includes input
and binary hashes. Its SHA-256 is `a7b1af2ca0b951085e4fff66fa39c9778ec536a68c6157b03498e2f0afe02710`.
The audit also ran both vocabulary/documentation companions, Flow spec validation,
and whitespace checks against HEAD and the vocabulary baseline; each returned 0.

Routine qualification defaults to no trace and no successful replay: 146 of 147
workloads are untraced. This is separate from the eight generator overrides whose
64 MiB choice capacity prevents tracing. The one traced exception,
`TestSignalWorkflowTestSuiteChasm`, and every capacity exception, non-qualified
expectation, and skipped subtest are reconciled with current MILESTONES.md.

| Document | SHA-256 |
| --- | --- |
| `tools/gomad3/SPEC.md` | `cc96c283ff1131cba5e217ba4773cd4a8d196c90132ee6e47cce62cfe8da533d` |
| `tools/gomad3/ARCHITECTURE.md` | `72d73a3f1d9ceec745c3aadd0ffb3580c9eec0539b7d4d18fdf428a497d929b5` |
| `tools/gomad3/CLI.md` | `e86de69dd32a5b999f99ee58eabe669c07a8d779ae7d9327ed2a7e6c14311eee` |
| `tools/gomad3/TUTORIAL.md` | `068d62ff9285115fbb7c1b085cb1a5d11231d03333c5b4c4977171366e383cf0` |
| `tools/gomad3/README.md` | `d3aa879db03cf21bbbe029213ea77556340f83b9978f3ecb03593e7dfbd744f6` |
| `MILESTONES.md` | `5308613089f2c698bafd42aa51396f8cc2197e795f8bb897b0da249092417577` |

Only the current-documentation portions of fn-109 R9 and fn-105 D5 reuse this
evidence: supported platforms, implemented ownership/contracts, evidence-claim
distinctions, and residual workload dispositions. Broader interface migrations
and their acceptance remain tracked by fn-109, outside this audit.
D13's routine untraced policy is implemented and D14's darwin correction is
recorded; native Linux replay verification (D12) and larger traces (D15) remain
outside this documentation acceptance.

The parent directory's task-1/task-2 receipts are historical snapshots and were
preserved; its acceptance summary now labels them explicitly. The retained
[empty-range review](historical-empty-range-review.md) did not establish renewed
current acceptance. Independent review of these actual repairs is owned by the
parent and was not performed by this worker.

No runtime or qualification expectations changed, and no qualification workload
was executed. Linux claims here come from source, manifests, and the workflow;
the clock and help checks ran on darwin/arm64. No commit, staging, push, or Flow
status/spec mutation was performed. Recheck bound hashes after document/source
edits; a later commit changes HEAD but not these recorded content hashes.
