# fn-247 savings amendment spec scout

The owner authorized the bounded savings amendment with "yes apply savings". This report uses only the five parent-supplied cached fresh CLI snapshots. No Flow calls, canonical changes, source edits, builds, tests or probes ran.

## Current state and dependencies

The cached parent is open, ready=true, plan-review=ship, reviewed at 2026-10-24T19:02:53.838178Z. Its native whole-spec dependency list is empty. The show snapshot was updated at 2026-10-24T20:16:34.292610Z.

| Existing task | Status | Native task prerequisites |
| --- | --- | --- |
| .1 extraction/baseline | done, flow-state | none |
| .2 identity/closure xtask | todo, committed | .1, .4 |
| .3 final wiring/docs/parity | todo, committed | .2, .7 |
| .4 Rust supervisor | todo, committed | .1 |
| .5 guest orchestration/freshness | todo, committed | .2 |
| .6 Apalache installer | todo, committed | .5 |
| .7 retired names/original verifier | todo, committed | .6 |

Current serial order is .1 -> .4 -> .2 -> .5 -> .6 -> .7 -> .3. The direct .1 -> .2 and .2 -> .3 edges remain explicit even though the chain also reaches them.

Task .1's completed extraction, evidence and acceptance stay untouched. Its retained guest baseline is `eaf6efd637e0e33b1a9f1e02b24cd8595c1f73706b5695fcde86151e884515f0`; its implementation evidence cites `4615afe62d8795f12c04b0cbc8e5522b23580678`. Its summary already assigns full gates and timing to the final task, so the savings amendment does not require rewriting that historical record.

## Legal insertion and requirement changes

Append exactly one M task. Let the parent allocate its actual ID through the CLI and verify that allocation before naming it in canonical text. Its native dependency is existing .1. Add the allocated task as another prerequisite of .4, retaining .4's existing .1 dependency and every other existing edge. The new serial order is .1 -> allocated savings task -> .4 -> .2 -> .5 -> .6 -> .7 -> .3. Existing IDs and statuses remain intact.

Append R5 for the approved validation savings. It covers the new `model-validate` entrypoint, preservation of required check/control coverage, reuse of shared work within one validation invocation, honest timing/failure evidence, and unchanged public `model-check`/`model-time` behavior. Concrete implementation and focused controls belong to the new task. The final .3 consumes that delivered target and supplies its integrated evidence. Assign R5 to the new task and .3 in task frontmatter and the parent coverage table.

Retain R1, R2 and R3 meanings. Retain R4's documentation, ownership, dependency, byte-equivalence, platform and overrun obligations, with one explicit owner-approved amendment to its measurement command. Replace the requirement for measured `make model-time` in final validation with measured `make model-validate` evidence under R5. The independently available public `make model-time` target keeps its behavior. Mirror R4's amended wording exactly in the coverage row.

Task .3 currently lists three complete invocations (`make model-check`, TLC-required model-check and Apalache-required model-check) plus `make model-time`. Replace that final repetition with one `make model-validate` invocation that discharges the approved integrated validation and timing responsibilities. Update .3's approach, Quick commands and acceptance together. Preserve its no-Python scan, exact baseline comparison, frozen originals, guest pin and platform-reporting obligations. The savings target's acceptance must prove equivalent required work rather than infer success from cached receipts or dropped checks.

The parent currently says the final task runs the complete default/required-backend gates and timing once. Amend that sentence, R4, the coverage table and any final-validation references consistently. .4 continues to run its focused process tests/Clippy and cold lifecycle proof. Only add its newly allocated prerequisite and corresponding ordering prose.

## Bounded downstream follow-up

Potential command-reference consumers are fn-246, fn-237.4, fn-237.5 and fn-243.5. This scout has no fresh bodies for those consumers and makes no claim about their exact current command text. The parent must verify `model-validate` exists and passes its specified coverage contract before updating their pending validation references. Limit any subsequent authorized follow-up to those four consumers, retain their semantic/live/evidence obligations, and leave completed tasks unchanged. No other-spec write belongs to this amendment.

No process redesign, dependency/TOML admission, archive investigation, guest policy change or downstream dependency rewrite is needed. The existing fn-247 execution ancestry gate and fn-246's whole-spec dependency on fn-247 remain in force.
