---
title: Preserve arithmetic evaluation roots across empty domains and unused values
date: "2026-10-23"
track: bug
category: integration
module: model/umpire/lift/src/safety.rs
tags: [rust, apalache, arithmetic, initialization]
problem_type: integration
symptoms: Later empty starts or discarded condition values hid evaluated arithmetic faults
root_cause: Safety validity followed surviving values instead of all evaluated source obligations
resolution_type: fix
related_to: [bug/integration/match-complete-macro-invocations-when-2026-10-19, bug/integration/preserve-rust-expression-and-inference-2026-10-20, bug/integration/preserve-scalar-bound-identities-when-2026-10-22, bug/integration/prove-result-shape-before-propagating-2026-10-21]
---

## Problem
The first Apalache safety Init conjoined every start domain. An overflowing outer endpoint followed by an empty inner domain therefore had no reachable error state, although native enumeration faults before reaching the inner domain. The audit also found an evaluated condition discarded when an empty State and empty branch bodies never consumed its named definition.

## What Didn't Work
A separate initialization-error disjunction repeated state constraints quadratically and let an unreachable FALSE branch omit state assignments that Apalache still requires syntactically. Forcing every lazy definition would instead evaluate untaken branches.

## Solution
Carry a named cumulative validity predicate in field declaration order, and assign each field from its authored domain while valid or from its typed default after an error. Earlier empty domains still prevent later evaluation. Share the complete step validity predicate once. The shared front records each evaluated condition as a guarded checks root, so the existing emitter checks obligations independently of value consumers.

## Prevention
Keep native and actual pinned-tool controls for faulty-first versus empty-first starts and evaluated versus untaken empty-State conditions. Pin length-zero initialization faults, and compare emitted size at 64 and 128 fields alongside sequential-return growth. Treat tool Deadlock/NotRun separately from a successful bounded safety answer.
