# Gomad: shrink the qualification machinery and move the cycle driver into Rust

## Problem

The fn-232.13 size ledger (counter `gomad-source-size/v1`, `.flow/artifacts/fn232-retirement/source-size-2026-10-23.md`) puts the Rust host outside the engine at about 84k production and 23k test lines of Rust, against 51k production lines for the Go host it replaced. Around it sits a qualification pipeline that has no Go counterpart: `tools/gomad/rust/driver/` is about 34k lines of Python (roughly 20k production, 12k tests). It holds the 96-stage scheduler, stage adapters, receivers, readers, custody checks and `required-final.json` obligations. There are also about 100 Python files of parity lanes under `testdata/parity/` and a 25k-line `measurement/` module that repeats much of the same checking. Rerun the counter at start; these figures are the 2026-10-23 row.

Nobody decided to write this in Python. It started as throwaway stage scripts in `.flow/artifacts/wasm-qualification-driver/` and was moved into the product tree during fn-232.13 (commit b31b463ca5, 2026-10-22). It then grew one obligation at a time. Reviews found obligations that only re-prove other obligations, and readers that check receipts instead of behavior.

## Goal

Qualification stays at least as strong as today, with much less code:
1. **Every check catches a real failure or goes.** For each obligation, reader and receipt check, name the concrete failure it detects and show a test or mutation that it catches. Delete what re-proves another check or only checks bookkeeping. Waive no stage, bound, assertion or the one-hour limit.
2. **The cycle driver becomes a Rust `gomad` verb.** The scheduler, stage dispatch and stage readers move into the Rust workspace, as one verb that runs the cycle and validates its results. The Python driver is deleted. Stage commands, bounds, resource policy and the one-hour limit stay the same.
3. **The parity harness is reviewed separately.** The black-box Python parity lanes drive both binaries from outside, which is a reason to keep them out of Rust. With the Go host retired and its goldens frozen, decide lane by lane: keep it as a golden regression, fold it into Rust tests, or retire it with its Go baseline.
4. **Measurement is checked for duplication.** Remove `measurement/` logic that duplicates qualify/result checking.

## Constraints

- **Starting point.** This begins after the fn-232.13 closing cycle, the fn-233 formal rerun and the fn-221 full sweep have run on `gomad`. It does not wait for fn-188.5 stress (MILESTONES ruling 7) or for fn-185.2 profiles, which have no scheduled window; those run on whatever commit is current when their window opens. There is no frozen candidate (opportunistic-merge decision, 2026-10-24): each window's evidence binds to the commit it ran on, and this spec's first commit starts a new binding.
- **Ordering with fn-240 (conductor ruling 2026-10-24).** fn-240's extractions also rotate identities. This spec runs first: it deletes the Python driver and the duplicated `measurement/` checking, which shrinks what fn-240 has to move and lets the new cycle verb be the tool that requalifies the split. fn-240 .2/.3 follow immediately, and one fresh full cycle under the Rust verb closes both this spec's requalification and fn-240's identity rotation. The owner may reorder; the cost of the other order is a second full cycle.
- **Requalification.** The result must pass one fresh full cycle under the new Rust verb, with the same stage set and bounds, and with a side-by-side comparison against the frozen-candidate cycle's verdicts.
- **Testing.** Golden and e2e tests come before unit tests, and a unit test must name its invariant.

## Acceptance Criteria

- **R1:** An obligation inventory exists for every `required-final.json` subgate (145 today), every driver reader/receiver module and every receipt check. Each entry names the concrete failure it detects and either its catching test or mutation, or its deletion reason. Errors: an obligation without a named failure, or a kept check without a catching test or mutation, is a finding.
- **R2:** A Rust `gomad` cycle verb replaces `tools/gomad/rust/driver/`: the same 96 stages, argv, bounds, resource policy (cores, memory, locks, scratch, live-memory guard, exclusive timing stages) and the 3,600-second limit, in serial and parallel modes, plus the kept readers and receivers as its validation step. The Python driver and its tests are deleted; Make, CI, QUALIFICATION.md and the determinism inventory's `driver:` evidence references point at the verb and its Rust tests. Errors: a stage whose argv, bound or resource declaration differs from the retired driver's plan fails the plan golden; a deleted check without an R1 deletion reason is a finding.
- **R3:** Every parity lane under `testdata/parity/` and `scripts/` has a recorded disposition (keep as golden regression, fold into Rust tests, or retire with the Go baseline), and that disposition has been carried out. Errors: a lane without a disposition, or a retired lane whose checks are not covered elsewhere, is a finding.
- **R4:** `measurement/` logic that duplicates qualify/result checking is removed; what remains is inventoried under R1. Errors: a removed check without an inventory entry is a finding.
- **R5:** Before/after file, byte and code-line counts are recorded with the fn-232.13 counter `gomad-source-size/v1`, separating production, tests and generated code. Errors: a count without the counter version and exact revisions is a finding.
- **R6:** One fresh full cycle passes under the verb within 3,600 s, with verdict parity against the fn-232.13 closing cycle (bound to the commit it ran on). Wall-time-only misses are recorded and judged in the quiet-host window; correctness verdicts must match. Errors: a verdict difference blocks closure; a missing retained command exit or identity is a finding.

## Interim rule (2026-10-23, until this spec runs)

No new Python driver code beyond what the fn-232.13 freeze and its post-freeze windows strictly need. New checks go into Rust tests or `gomad` verbs.

## Decision Context

Owner request 2026-10-23: shrink the qualification machinery and move the driver into Rust. Python entered by accident (stage scripts promoted into the product tree in b31b463ca5) and the owner's rule since then is no new Python driver code. The parity lanes are kept out of the Rust move on purpose: they drive binaries from outside, which is what makes them a regression check rather than a self-check. Tasks come from `/flow-next:plan` once the starting point above is reached; the obligation inventory (acceptance item 1) is the first task because every deletion decision depends on it.
