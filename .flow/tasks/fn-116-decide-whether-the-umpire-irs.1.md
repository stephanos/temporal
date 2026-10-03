---
satisfies: [R1, R2, R3, R4, R5]
---
# fn-116-decide-whether-the-umpire-irs.1 Prototype CEL expressions against both Models and write the report

## Description
Run the CEL spike end to end and write the report. Implements R1-R5.

**Size:** M
**Files:** an isolated prototype Go module and a small JVM harness under git-ignored `.flow/tmp/fn116-spike/`; the report `.plans/UMPIRE_CEL_SPIKE.md`
**Touches:** [.plans/UMPIRE_CEL_SPIKE.md]

### Approach
- The owner allows no worktrees and reserves commits, so "a branch that is not merged" is an isolated prototype module under `.flow/tmp/fn116-spike/` that depends on the server module through a `replace` directive. Nothing in the main tree changes except the report.
- Translate every expression of `model/ir/nexus-caller.json` and the standalone activity IR to CEL syntax trees in Go, evaluate with `cel-go`, and compare with the reader's own evaluator on every state and class, using the reader's public interface and the fn-115 goldens as the baseline.
- Evaluate the same translated step functions with `cel-java` and compare with `cel-go`.
- Take the measurements listed under API Contracts; state any not taken with the reason.

## Acceptance
- [ ] Every expression of both Models translates to CEL or is listed with its Scala position and blocking construct; `cel-go` results equal the current evaluator's on every state and class, with each difference listed.
- [ ] `cel-java` results equal `cel-go`'s, or each disagreement is listed.
- [ ] `.plans/UMPIRE_CEL_SPIKE.md` gives the measurements, the adoption cost beyond the prototype and a recommendation with reasons; no production path, schema, lifter or checked-in IR changed.


## Done summary
Built an isolated CEL prototype under `.flow/tmp/fn116-spike/` (an `Expr`-to-CEL translator, a copy of the reader that evaluates through the reader, cel-go or both, and a cel-java harness) and wrote `.plans/UMPIRE_CEL_SPIKE.md`. Every expression of the six Models translates and type-checks. cel-go matched the reader's evaluator on every call over every state and class (0 mismatches; non-vacuous by two sabotage runs and the channels fixture), and 43 of 44 reader golden entries are byte-equal, the exception being an unordered channel send the prototype does not translate. cel-java matched cel-go on all 31,025 exported calls when given the same descriptors.

Recommendation: keep the IR's own expressions. Adoption deletes about 490-580 lines of Go evaluator and validator but needs about 960 lines of Go and 1,000 of Scala, grows the IR 1.8-3.7x with positions, and makes table derivation up to 3.8-6.3x slower; 206 of 208 Property and monitor roots would need generated descriptors, so a subset adoption does not escape the cost. Not measured: nexus-close's Queries and goldens under CEL, the lowerer goldens, realization operands.

Independent review (Claude Fable, fresh context): NEEDS_WORK in round 1 (presentation of size and timing, unmeasured subset), SHIP in round 2. `go.mod`/`go.sum` and all production paths are unchanged. Handover: .flow/tmp/fn116-1-summary.md; evidence: .flow/tmp/fn116-1-evidence.json; reviews: .flow/tmp/fn116-1-review/. No agent commits.
## Evidence
- Commits:
- Tests: cd .flow/tmp/fn116-spike && ./run.sh test -tags test_dep -p 1 -count=1 ./tools/umpire/model -run '^TestSpikeTranslate$' -v  # exit 0, logs/translate.log, SPIKE_MODELS=<each> ./run.sh test ... -run '^TestSpikeCompare$'  # exit 0 for nexus-caller, activity, activity-system, activity-race (logs/compare-first-run-terminated.log: those four completed before the run was stopped during nexus-close), nexus-control (logs/compare-nexus-control.log), SPIKE_MODELS=nexus-close SPIKE_CHECK=0 SPIKE_EXPORT=0 ./run.sh test ... -run '^TestSpikeCompare$'  # exit 0, logs/compare-nexus-close-tables.log, UMPIRE_CEL_LOGIC=strict | UMPIRE_CEL_CALLS=host SPIKE_SUFFIX=... -run '^TestSpikeCompare$'  # exit 0, logs/compare-strict.log, logs/compare-host.log, UMPIRE_EVAL=cel SPIKE_GOLDEN=<input> ./run.sh test ... -run '^TestSpikeGoldens$'  # exit 0 for 10 inputs, exit 1 for lifts/expected/channels (1 entry differs), timeout for ir/nexus-close; logs/goldens-cel.log, logs/goldens-variants.log, out/goldens/, UMPIRE_EVAL=cel SPIKE_SABOTAGE=1 SPIKE_GOLDEN=ir/nexus-caller ... -run '^TestSpikeGoldens$'  # exit 1 as intended (negative control), logs/goldens-sabotage.log, UMPIRE_EVAL=expr SPIKE_GOLDEN=ir/nexus-caller ... -run '^TestSpikeGoldens$'  # exit 0, logs/goldens-expr-baseline.log, SPIKE_MODELS=... SPIKE_FUNCTIONS=... -run '^TestSpikeUncovered$'  # exit 0, logs/uncovered.log, -run '^TestSpikeEdges$'  # exit 0, logs/edges.log, out/edges.json, -run '^TestSpikeFallible$'  # exit 0, logs/fallible.log, out/fallible.json, -run '^TestSpikeSizes$'  # exit 0, logs/sizes.log, out/sizes.json, -run '^TestSpikeTiming$'  # exit 0, logs/timing.log, out/timing.json; profile out/profile/cpu.out, ./run-java.sh ../out/<model>/java standard|planner [own-descriptors] && diff expected-cel-go.txt actual-cel-java-*.txt  # 0 diff lines for all exports in both runtimes; 1890 with own-descriptors; logs/cel-java.log, deps/: go list -m all before/after go get cel.dev/cel-go@v0.32.0 on a copy of go.mod/go.sum  # out/deps-before.txt, out/deps-after.txt, out/deps-gomod.diff, out/deps-new-pkgs.txt, git status --short go.mod go.sum  # empty: unchanged, SPIKE_MODELS=... -run '^TestSpikeSubset$'  # exit 0, logs/subset.log, out/subset.json, UMPIRE_EVAL=cel SPIKE_SABOTAGE=steps SPIKE_GOLDEN=ir/nexus-caller|ir/activity.json ... -run '^TestSpikeGoldens$'  # exit 1 as intended (2 and 3 entries differ), logs/goldens-sabotage-steps.log, UMPIRE_EVAL=expr SPIKE_GOLDEN=ir/nexus-close ... -run '^TestSpikeGoldens$' -timeout 5m  # exit 0 in 44.2s, 0 different, logs/goldens-expr-nexus-close.log, review round 1 revisions applied to .plans/UMPIRE_CEL_SPIKE.md, Independent review round 2 SHIP (claude-fable-5-1); .flow/tmp/fn116-1-review/round2-review.md
- PRs: