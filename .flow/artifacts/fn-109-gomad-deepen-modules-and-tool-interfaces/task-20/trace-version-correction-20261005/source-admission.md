# Current choice trace documentation

fn-109.20 owns this R9 guidance correction against committed candidate `d108c313dc`. README lines 135/867, ARCHITECTURE line 278 and TUTORIAL lines 389/395/497 still name v2. Current `DecodeStoredTrace` refuses v2 because it has no select readiness; `ProjectReplayPlan` requires complete v3 evidence and projects readiness onto matching poll decisions. Legacy-v1 traces remain decodable for inspection without an exact-replay tape. Their runnable and poll records can retain decision flags.

Change only those six claims and a short architecture explanation. Preserve unrelated v2 schemas, commands, source, qualification dispositions, task dependencies and acceptance. This checkpoint adds no legacy support and proves no native replay or R18 reconciliation. Use the task's named controls, check-only generator validation and a source/claim preservation audit. Root commits only after an independent source-progress review; original acceptance remains open.

Scheduling: wave (task-id run)
Ready frontier: no acceptance-ready task20 because .19 remains blocked; MILESTONES permits source progress from the integrated reviewed predecessor.
Selected wave: fn-109.20
Selection rule: one factual guidance owner with unchanged source; no overlapping writer.
Isolation: existing checkout, single documentation/evidence and Go-cache writer.
Dispatch count: 1
Tier: session (jev-unavailable(no_key)); explicit AGENTS implementer routing retained.
