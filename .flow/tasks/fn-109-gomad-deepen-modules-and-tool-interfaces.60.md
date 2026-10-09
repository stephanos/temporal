---
satisfies: [R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.60 Check six child-fixture output results without changing process outcomes

## Description
Bounded R18/R19 execution-fixture correction supporting fn-112.10 source acceptance, admitted at cd883200c6791527acf263b151e7cf54d7ccd1e8. Root owns Flow, gates, integration, independent review and commits. Follow MILESTONES.md; keep native fn149/fn128 deferred and unverified.

**Touches:** [tools/gomad3/runner/internal/execution/process_test.go, tools/gomad3/runner/internal/execution/process_fixture_output_test.go]

Check exactly the six remaining child stdout writes: target stdout, post-choice marker, read-only tape marker, reordered runnable output, choice-select sequence and choice-prefix RNG sequence. Use the retained Astra design in source-unblocking-20261007/lint-next-batches.md and existing source. Preserve all argument evaluation, receive order, logical work, lock/unlock order, healthy bytes, parent-observed stderr/EOF, and helper exits. Checking a failure must not add an exit status, skip remaining work or invent a new fatal diagnostic. The target-stdout helper must still write target stderr and exit7; marker helpers retain their existing subsequent work and exit0; leaf sequence functions may return a checked error to their existing caller without changing its exit. No production output policy change, new stdout abstraction, watchdog ready-marker change, deliberate busy-loop change, suppression, platform bypass or dependency addition.

Before editing retain independent healthy/error expectations and actual original diagnostics. Add only focused controls that exercise real child-output failures through a lawful existing descriptor/fixture boundary; never count source inspection or a mutation as real failed-writer execution. Keep genuine unavailable error paths explicit. Freeze source/tool hashes and every baseline/candidate command outcome. Source-equivalent original reconstruction must prove all non-admitted bytes unchanged. Original-base integrated lint is a retained red gate, not a predicted subtraction; compare full residual diagnostic blocks after line mapping. Parent-wide first-baseline/fixed-identity/R18/R19 acceptance remains unchanged.

**Quick:** pinned stock Go1.27.1, all tests with -tags test_dep -count=1; focused affected child-helper controls and new output controls, ordinary ./runner/internal/execution once for the final candidate, affected configured lint, check-only make lint-code-fast with FIX=false, vet and format, original-base lint against951c5516e9e7b3066e7e069adda9565cfd68844c. All Go/build/test/lint/vet/generator commands require explicit root serialized-lane grant. Bind results and inherited failures/skips without treating them as native passes or automatically repeating unchanged failures. Task-unique handover and raw receipts live under .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-60/; these lifecycle evidence paths are exempt from product Touches. No worker Flow completion/review verdict, local commit until root grant, merge/push/PR/CI.

## Acceptance
- [ ] Exactly six admitted output results checked with all healthy bytes, work, receive and defer order, EOF/stderr and helper status preserved; every other source byte reconstructed unchanged.
- [ ] Independently frozen healthy and failure controls use genuine lawful failure inputs where reachable; source-only or unsupported cases and inherited ordinary-package failures stay explicit.
- [ ] Actual source-bound baseline/candidate lint removes exactly six errcheck findings, introduces zero findings, and preserves all other full diagnostic blocks after line mapping; no policy or platform bypass.
- [ ] Focused tests, affected ordinary coverage, vet, format and check-only canonical fast lint retain their actual outcomes; original-base lint/errortype and parent acceptance remain open wherever red or unexecuted.
- [ ] Fresh independent source-progress review, separate integrated checkpoint and task-unique evidence retain all remaining source and deferred native requirements; Done only when every requirement still owned here is verified.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
