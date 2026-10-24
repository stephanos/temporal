---
satisfies: [R12]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.33 Mandatory workload cells install without --compile-cache; the enforced warm-resident budget refuses every cell

## Description
Found 2026-10-24 by running the sqlite success cell standalone through rust/workload_tools/producer.py on gomad-dad35ab548-8m (evidence: scratchpad cells/sqlite-success-01 and -02; summary in .flow/artifacts when committed). producer.py runs 'install' with --compiler/--runtime-root/--cache/--output/--profile/--config/--fuel/--memory-bytes but never --compile-cache, so installation.admission.compile_cache_root is None. campaign/prepare::compile_cache_root then yields None for the executions (campaign/execution.rs:413) and the cell replays (qualify/cells.rs:681); every store compiles cold, all samples carry cache_hit=false, and qualify::cost::budget refuses with 'actual warm helper peak measurement is required for each resident guest' (cost.rs:456; QUALIFICATION.md 'Warm measurement for every resident guest' is Enforced). qualify still exits 0 and report classifies 'qualified', but result.budget is null and budget_failure is set, so producer.py's own check ('actual mandatory workload budget is unavailable or failed') fails. Rerunning the identical cell with install --compile-cache ROOT/engine-cache gives within_budget=true (8 checks within) with cache_hit=true on execute 2 and both replays.

Affected cycle stages: the producer.py dispatches in primary_stages.py:402, i.e. storage (recovery workload), network (retry) and clusters (replication, plus --original-clusters). The ten workload-* cells are NOT affected: their sealed installations (bindings.json 'installations') already carry compile_cache_root under compiled-code/<cell>. temporal-inspection runs explore and has no budget gate.

Scope: (a) producer.py install argv gains --compile-cache <root>/engine-cache for every workload, mirroring onebox-gates.py:381 and installed_lifecycle_families.py:46; (b) qualify refuses at admission (status 2, named reason) when a non-provisional cell's installation has no compile cache root, so the gap is explicit instead of a late budget refusal; (c) QUALIFICATION.md states that mandatory cells run with the installation compile cache; (d) a workload_tools test pins the producer install argv. No budget, bound or message changes. Cross-spec: fn-246 final qualification depends on this; record there. Same family as .29/.30/.31: Rust enforces a check the retired Go runner satisfied through its own installation wiring.
## Acceptance
Standalone workload-sqlite-cooperative-success and -canary through producer.py exit 0 with result.budget.within_budget true and cache_hit true on every non-first window; qualify on an installation without compile_cache_root for a non-provisional cell exits 2 with a named reason (test); driver test pins --compile-cache in the producer install argv; QUALIFICATION.md updated; no change to cost.rs budget semantics.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
