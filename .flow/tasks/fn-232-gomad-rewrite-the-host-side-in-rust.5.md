---
satisfies: [R4, R13, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.5 inspect in Rust

Touches: tools/gomad/rust/gomad/src/cli/inspect.rs

## Description
Implement `inspect` over Rust-lineage artifacts with the same facts the Go inspect reports (record, identities, streams, choice summary and sites, world and simulation summaries, minimization block) and a documented `--json` shape; Go-lineage artifacts refused. Use task .3's shared authenticated pinned-root opener for every manifest/payload read, without path-based duplicate readers. Add inspect to the harness with field-semantic comparison against Go inspect on equivalent fresh executions, using task .4's declared golden view and separate raw-integrity checks.

## Acceptance
The owned behavior has reviewed golden fixtures, repeated-execution agreement and byte-identical second regeneration (R14); public verbs also run their fresh Go/Rust semantic parity lane (R13).
Harness inspect lane passes on the corpus; corrupt manifest, wrong lineage, root substitution/symlinks, traversal, wrong permissions/links, oversized or altered payloads and digest mismatches produce the existing open refusals; legitimate target-pool module hard links remain accepted; CLI.md diff lists every `--json` shape change.

## Done summary
Implemented Rust inspect through the shared authenticated pinned-root opener, documented its JSON shape, and added the mandatory fresh semantic/golden lane. R4, R13 and R14 are met within task .5.

Independent three-axis review found three R4 defects; donor 637925f9b6 integrated as e393fe7e89 repairs all three. Payload authentication now streams with a fixed 16 KiB buffer after consumer bounds; minimization validates kinds, dimensions, counts and effective reduction chains; choice terminal metadata agrees with execution outcomes. Actual authenticated CLI REDs precede the repair.

Root fresh mandatory lane at /home/agent/wt/evidence/fn232-inspect/root-e393fe7e89 passed (exit0): 12 base and four choice executions, including two actual overflow prefixes; 34 review regression commands have expected statuses (13 success, 21 refusal); public opener/consumer faults, raw integrity and exact goldens pass. All 16 golden files regenerated twice identically in the worker's frozen candidate4 checks; release rejects all 17 feature-only controls. Root independently verified candidate4 binary hashes and source equality to the integrated donor.

Independent re-review session 01a15bd6-b069-7060-85d9-30312587a1dc returned SHIP with R4/R13/R14 met and no remaining findings; receipt .flow/tmp/impl-review-receipt-fn232-inspect.json. One batch handover: /home/agent/wt/evidence/fn232-inspect/HANDOVER.md. Reviewer/writer are the same model family, with fresh reviewer contexts.

World/simulation/minimization consumer coverage is not producer or linked-host cluster qualification. Rust exploration/minimizer producers, final provisioning/qualification, and task .1's compilation/teardown deadline remain open under their existing owners. No Go-artifact migration, identity normalization, CI run or deadline waiver.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 4d8b934727, 56d39c316ce031220597bcaf0f63fc0c0607f873, e393fe7e89
- Tests: python3 tools/gomad/scripts/behavioral-parity --corpus tools/gomad/testdata/parity --verb inspect --inspect-binary /home/agent/wt/evidence/fn232-inspect/candidate4/gomad-fixture --inspect-baseline /home/agent/wt/evidence/fn232/go-baseline-1d7b37f00a --inspect-source /home/agent/wt/fn232-go-source/temporal-1d7b37f00a --inspect-go-compiler /Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/2198cf19890232fc/go/bin/go --output-root /home/agent/wt/evidence/fn232-inspect/root-e393fe7e89 (exit0), candidate4 immutable worker checks: 24 named numeric exit0 gates; exact argv/status/stdout/stderr and driver /home/agent/wt/evidence/fn232-inspect/candidate4-check.sh, git diff --exit-code 637925f9b6 HEAD -- tools/gomad/rust tools/gomad/testdata/parity tools/gomad/docs/CLI.md (exit0 at integration), git diff --check (exit0)
- PRs: