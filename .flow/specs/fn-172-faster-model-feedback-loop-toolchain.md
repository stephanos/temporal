# Faster model feedback loop: toolchain fixes independent of reachable exploration

## Execution status (2026-10-21 backlog audit)

Deferred by the owner. The Rust decision has been made and the active workspace is `model/`; the former Scala tree is frozen under `model/archived/` and is not expected to build (AGENTS.md and MILESTONES.md). The earlier “until the Rust verdict” note is historical, not an automatic release of this spec. Existing task states and old SHIP reviews describe historical checkpoints, not current implementation readiness.

Resume only after an explicit owner decision and a fresh scope/dependency review against the Rust architecture. Scala paths and commands below describe the archived layout; do not run them as current gates or mechanically port the plan. Preserve unfinished requirements and completion evidence; successor Rust work does not discharge this spec without an explicit requirement mapping. Taskless proposals remain deferred captures, not executable plans.


## Conversation Evidence

> user: "is the reachabile design of umpire also responsbile for the super long/slow feedback cycles?"
> user: "yes add flow next todos for `Toolchain fixes that don't need this design`"

## Goal & Context

A change to the framework samples took 19 minutes under `make model-check-changed`, measured on 2026-10-13 (record `.flow/tmp/checks/20261013-092021/record.md`). Most of that time is fixed Scala toolchain overhead and Go-side interpretation cost. Neither depends on the full-catalog design that `model/design/reachable-exploration.md` proposes to replace. This spec takes the fixes that need no design change, so the loop gets faster now and stays fast after reachable exploration lands.

Measured on 2026-10-13 on a warm machine:

| Step | Time |
| --- | --- |
| `scala-cli package` of framework and samples | 32.6 s |
| `scala-cli compile --print-class-path` of the same inputs | 12.4 s |
| One lift (Kettle, 564 states) | 40.7 s |
| One lift (Voting, 19,683 states) | 30.5 s |
| Go check of Bank's lifted IR | 48.6 s |
| Go check of Voting's lifted IR | 52.0 s |
| `make model-check-changed`: samples stage | 317 s |
| `make model-check-changed`: lifter suites | 560 s |

Lift time does not track model size: Kettle's lift takes longer than Voting's. So lifting is fixed overhead. Every scala-cli call passes `--server=false`, the same inputs are compiled twice, and each sample gets its own lifter JVM. A CPU profile of Bank's Go check, in reachable-exploration.md §10, shows `check.Check` interpreting the Model twice: the replay binding at `tools/umpire/check/checking.go:370-375`. It also shows garbage collection taking about 58% of CPU.

## Acceptance Criteria

- **R1:** Find out why `--server=false` is used and record it. Remove it wherever a warm scala-cli/Bloop server is safe: `model/framework/check/Samples.scala:136,143,153`, `model/framework/check/ChangeCheck.scala:260`, and the Makefile model targets (`Makefile:766-880`, 19 uses). Where it must stay, keep it and leave a comment giving the reason.
- **R2:** Compile each set of inputs once per run. Today the samples check (`Samples.scala:136-147`) and the Makefile's model jar (`Makefile:766-767`) package the inputs and then compile them again only to print the classpath.
- **R3:** Lift all samples in one lifter run, using the existing multi-file mode (`lift --ir`, `model/README.md:315`) or several roots per run, instead of one JVM per sample (`Samples.scala:150-165`). Progress-claim roots stay as they are.
- **R4:** Package the lifter once, and reuse the samples jar and classpath when their inputs haven't changed.
- **R5:** Build the Go sample checker once, not through `go run`, and check the samples in parallel.
- **R6:** `check.Check` interprets each Model once: remove or share the replay binding (`checking.go:370-375`), and reduce the allocation behind the 58% GC share. Receipts stay byte-identical.

## Edge Cases & Constraints

- Every requirement keeps today's answers. The six samples still give the same Query and progress receipts, and `make model-check-changed` gives the same verdicts apart from the macOS-only suite failures already seen (`flock` missing, `/private/var` paths, the step-ledger jar receipt).
- Each task records before and after wall times for the stage it touches, measured the same way as the table above, and cites them in its done summary.
- No change to IR bytes, Behavior Fingerprints, Definition IDs or Case identities.

## Requirement coverage

| Req | Task |
| --- | --- |
| R1 | .1 |
| R2 | .2 |
| R3 | .3 |
| R4 | .4 |
| R5 | .5 |
| R6 | .6 |

## Out of scope

- Reachable-only tables and the semantic changes in `model/design/reachable-exploration.md`.
- IR shape changes in `model/design/ir-simplification.md`.

## Decision Context

2026-10-18, deferred (owner): all specs other than fn-210 (the Rust prototype of the framework in `temporal/model2`) are deferred until fn-210's write-up recommends whether to extend the Rust prototype or stay on Scala. Flow has no deferred status; this line is the record. MILESTONES.md was reset to the prototype alone and no longer lists this spec. Nothing in this spec changes; resume or close it after that decision.
