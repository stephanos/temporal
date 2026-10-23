# Gomad: shrink the qualification machinery and move the cycle driver into Rust

## Problem

The fn-232 Rust host is about 62k lines of production Rust, against 51k for the Go host it replaced. Around it sits a qualification pipeline that has no Go counterpart: `tools/gomad/rust/driver/` is about 31k lines of Python (19.7k production, 11.6k tests). It holds the 96-stage scheduler, stage adapters, receivers, readers, custody checks and `required-final.json` obligations. There are also about 22k lines of Python parity tests and a 24.8k-line `measurement/` module that repeats much of the same checking.

Nobody decided to write this in Python. It started as throwaway stage scripts in `.flow/artifacts/wasm-qualification-driver/` and was moved into the product tree during fn-232.13 (commit b31b463ca5, 2026-10-22). It then grew one obligation at a time. Reviews found obligations that only re-prove other obligations, and readers that check receipts instead of behavior.

## Goal

Qualification stays at least as strong as today, with much less code:
1. **Every check catches a real failure or goes.** For each obligation, reader and receipt check, name the concrete failure it detects and show a test or mutation that it catches. Delete what re-proves another check or only checks bookkeeping. Waive no stage, bound, assertion or the one-hour limit.
2. **The cycle driver becomes a Rust `gomad` verb.** The scheduler, stage dispatch and stage readers move into the Rust workspace, as one verb that runs the cycle and validates its results. The Python driver is deleted. Stage commands, bounds, resource policy and the one-hour limit stay the same.
3. **The parity harness is reviewed separately.** The black-box Python parity lanes drive both binaries from outside, which is a reason to keep them out of Rust. With the Go host retired and its goldens frozen, decide lane by lane: keep it as a golden regression, fold it into Rust tests, or retire it with its Go baseline.
4. **Measurement is checked for duplication.** Remove `measurement/` logic that duplicates qualify/result checking.

## Constraints

- **Starting point.** This begins only after the fn-232.13 final cycle and the post-freeze evidence windows (fn-233 rerun, fn-188.5 stress, fn-221 sweep, fn-185.2 profiles) have finished on the frozen candidate. It must not change the code those windows qualify.
- **Ordering with fn-240.** fn-240 (crate split) also changes identities. Run this spec before fn-240's extractions, or interleave with them on purpose, so only one requalification follows.
- **Requalification.** The result must pass one fresh full cycle under the new Rust verb, with the same stage set and bounds, and with a side-by-side comparison against the frozen-candidate cycle's verdicts.
- **Testing.** Golden and e2e tests come before unit tests, and a unit test must name its invariant.

## Acceptance

- [ ] An obligation inventory exists. Each entry lists the failure it detects and either its catching test or mutation, or its deletion reason. No obligation remains without a named failure.
- [ ] The Rust cycle verb replaces `tools/gomad/rust/driver/`. The Python driver and its tests are deleted, and Make, CI and QUALIFICATION.md point at the verb.
- [ ] Each parity lane has a recorded disposition (keep, fold or retire), and that disposition has been carried out.
- [ ] Duplicated measurement checking is removed.
- [ ] Before/after line counts are recorded with the fn-232.13 size counter.
- [ ] One fresh full cycle passes under the verb within 3,600 s, with verdict parity against the frozen-candidate cycle.

## Interim rule (2026-10-23, until this spec runs)

No new Python driver code beyond what the fn-232.13 freeze and its post-freeze windows strictly need. New checks go into Rust tests or `gomad` verbs.
