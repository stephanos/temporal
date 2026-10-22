---
satisfies: [R9, R13, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.10 qualify and report in Rust

**Touches:** tools/gomad/rust/gomad/src/qualify/{cells.rs,report.rs}, tools/gomad/rust/gomad/src/cli/qualify.rs

## Description
Own both `qualify --spec FILE` and the existing installation-free `report --result FILE`. Port report grammar, authenticated result opening, independent cost/verdict recomputation and corruption/unsupported-format statuses; use fresh own-lineage results in behavioral parity. Implement the parent linked-process resource-accounting and qualification-specific golden projection contracts. Every prior budget maps to an actual measurable quantity, without double-counting process self CPU or attributing combined helper RSS to an operator-only ceiling. Unavailable equivalent measurement leaves acceptance open rather than weakening limits.

Port `qualify`, `qualify-set`, `merge-set` and `compare-support`, preserving current WASM supported/refused cases: repeated seeds and fresh instances per cell, replay of every retained execution, report bound to source, helper, module, model, compiler, host, profile and limits in the new schema, for the fixture cell and the SQLite workload first, then the one-box and cluster cells. Add verdict comparison to the harness.

Provisional start (worker D, 2026-10-21): `flowctl start` refused because .9 is in progress, so this records the provisional start instead of forcing a claim. Development continues on branch fn232-qualify against the declared .7/.8/.9 interfaces. Acceptance waits on .9 and the integrated candidate.

Conductor decision (2026-10-21): Rust `qualify --spec` admits multi-instance scenario cells (`Target.Scenario`, `Workload: multi-instance`). The Go host's public verb refuses them (`invalid_input`, status 2), and Go qualifies clusters only through its in-package test API, which the Rust binary does not have. This adds availability and removes nothing, so feature preservation holds. CLI.md documents it as a deliberate availability change. The parity lane derives the Go cluster verdict from Go `explore --scenario` plus `replay`.

## Acceptance
Report runs without installation/source/compiler; tampered result/receipt observations or hashes refuse. Fixtures prove CPU counted once, concurrent-store accounting, threshold crossings, missing-measurement failure and independent report recomputation. Authenticate raw variable measurement envelopes before comparing the stable qualification projection; preserve execution-record hashes and traces exactly.

The owned behavior has reviewed golden fixtures, repeated-execution agreement and byte-identical second regeneration (R14); public verbs also run their fresh Go/Rust semantic parity lane (R13).
All owned public-command lanes have parity and golden fixtures; qualify matches verdicts per cell; a failed repetition or replay fails the cell with an explicit report; reports are retained under the new schema.

## Done summary
Accepted qualify and installation-free report on the integrated Rust host. All five fresh cell lanes passed with 20 independent Results, byte-identical independent golden generations, 155 authenticated original command receipts and 78 fresh ordinary-release reports. All 51 accounting cases and real Linux phase checks passed. Preserved three actual WASM command refusals with 18 repeated Go/Rust cases. Independent source and golden review: SHIP, no findings. Exact commands, exits, scope and identities: .flow/artifacts/fn232-10-qualification-closure-2026-10-22.md. Final whole-candidate qualification remains fn-232.13.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 780099bf90
- Tests: final-small-regen: exit 0, final-onebox-regen: exit 0, strict-retained small+onebox: exit 0, qualify-accounting-check: 51 cases and Linux phase checks exit 0, fixture and release grammar: exit 0, Rust focused tests, clippy, fmt: exit 0, Independent source and five-cell golden review: SHIP
- PRs: