---
satisfies: [R1, R2, R3, R4, R5, R6, R7]
---
# fn-14-milestone-a-pilot-baseline-and-lean.6 Execute and retain the Milestone A pilot decision

## Description
Run the frozen protocol once, retain the full v1 evidence bundle and narrative report, and reconcile roadmap status for R1-R7.

**Size:** M
**Files:** `docs/research/umpire-milestone-a-pilot-evidence/v1/**`, `docs/research/umpire-milestone-a-pilot.md`, `model/README.md`, `.plans/UMPIRE4_COMPONENTS.md`
**Touches:** [docs/research/umpire-milestone-a-pilot-evidence/v1/**, docs/research/umpire-milestone-a-pilot.md, model/README.md, .plans/UMPIRE4_COMPONENTS.md]

### Approach

- Execute the provider-free baseline twice and exactly three logical live Agentworkflow trial slots sequentially from the frozen source/config inputs, retaining every attempt and the single permitted infrastructure-only retry per slot. This task is the sole owner of live provider trials.
- Publish strict Agentworkflow exports, raw normalized evidence, hashes, exact metrics, gate results, and the recomputed decision receipt without hand-editing measurements or agent candidates.
- Generate the narrative report from the verified receipt and link rather than duplicate the threshold table in public docs/roadmap.
- Record whichever closed outcome the evidence yields; do not force a GO. Update Milestone A and pilot sequencing truthfully, including the next action authorized by that outcome.
- Run focused checks, strict bundle verification, regression projection checks, and diff/comment preservation checks.

### Investigation targets

**Required:**
- `docs/research/umpire-milestone-a-pilot.md` — frozen protocol and report target.
- `model/README.md:68-158` — user-facing model commands and no-runtime boundary.
- `.plans/UMPIRE4_COMPONENTS.md:569-714` — milestone status, evidence, and pilot sequence.
- `tools/umpire/pilot/decision/receipt.go` — landed strict outcome authority.

### Quick commands

`make umpire-pilot-run && make umpire-pilot-verify EVIDENCE=docs/research/umpire-milestone-a-pilot-evidence/v1 && make umpire-check-regression`

## Acceptance
- [ ] The retained v1 bundle contains exactly the frozen manifests, twelve mutation results, coverage/timing records, three logical trial slots plus every retained attempt/retry, strict Agentworkflow exports, canonical patches where available, rubric evidence, and one recomputable decision; any conclusive outcome requires all three slots to be valid, while exhausted infrastructure evidence recomputes to `INCONCLUSIVE`.
- [ ] Every digest validates and two provider-free runs agree on normalized non-duration inputs.
- [ ] No earlier task ran a live provider trial; all three retained trials start from the same frozen source/config/prompt/model and no candidate was applied or manually changed.
- [ ] The report states exact metrics, misses, decision, and authorized next action without claiming runtime execution or conformance.
- [ ] Roadmap and model docs link the evidence and decision without duplicating or weakening thresholds.
- [ ] Focused tests, pilot verification, regression checks, and `git diff --check` pass with existing comments preserved.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
