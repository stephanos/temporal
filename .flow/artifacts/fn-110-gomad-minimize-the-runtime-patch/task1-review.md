# fn-110.1 implementation review (raw codex bridge, gpt-5.6-sol at high, read-only, working-tree files)

Three rounds. Round 3's first launch returned no verdict (codex tried to start a review launcher that the read-only sandbox blocked); it was re-run with a direct-review instruction and is the terminal verdict.

## Round r1

Must-fix:

- [Runner drift handling](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_PATCH_SIZE.md:372) understates the equivalence impact. Qualification targets rebuild the Runner, and `make test` compiles the changed Runner and new tests. Matching results would still compare two changed inputs. Require the exact baseline Runner or rerun the before gates on identical Runner source. Also pin the referenced diff to `6782b55f…`; concurrent HEAD `38957053f…` makes the stated `git diff HEAD` digest unreproducible.

Should-fix:

- [The overlay delta explanation](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_PATCH_SIZE.md:160) omits generated digest changes. `gomad.go` grew 809 bytes, while the overlay grew 792 because `gomadchoicewire/wire_generated.go` shrank 17 bytes.
- [The process-simulation row](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_PATCH_SIZE.md:249) attributes 25 runs to the patched-Go command, but only 14 used it; 11 used host Go. Two runs also had a third timeout failure, so “9 pass, 2 fail” needs qualification.

All recomputed patch, per-file, overlay, digest, allowlist, identity, report, D12/D14, Linux, link, additions-only, and em-dash checks otherwise match the document.

VERDICT: NEEDS_WORK

## Round r2

- **Must-fix:** [identity.json](/Users/stephan/Workspace/temporal/gomad/tools/gomad3/.toolchain/fn-110/baseline/identity.json:31) swaps the core and Temporal `manifest_sha256_in_report` values. Core reports `6c60886a…`, Temporal reports `5a9ad084…`; lines 31 and 39 record the reverse. Correct them and refresh the identity digest in `task1-baseline.json`.

All document measurements recomputed exactly. Round-1 fixes, Runner drift handling, failure reporting, D12/D14 text, links, additions-only diff, and no-em-dash requirement pass.

VERDICT: NEEDS_WORK

## Round r3-attempt1-no-verdict

Flow-next review could not start because its launcher requires temporary-file creation, which the read-only sandbox forbids.

<promise>RETRY</promise>

## Round r3

No must-fix or should-fix findings. All independently recomputed figures, identity bindings, evidence, links, quoted dispositions, and additions-only prose checks matched.

VERDICT: SHIP
## Fixes applied

- Round 1: Runner-drift row rewritten to block cross-Runner comparisons and name the two valid routes; diffs pinned to 6782b55f4 and the baseline commit 38957053f recorded; overlay delta broken down; process-simulation row states 14 patched-go and 11 host-go runs and the two three-failure runs.
- Round 2: swapped core/Temporal manifest_sha256_in_report values in tools/gomad3/.toolchain/fn-110/baseline/identity.json corrected with a correction note; digest refreshed in task1-baseline.json.
