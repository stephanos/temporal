---
satisfies: [R7, R10, R11, R14, R15, R18]
---
# fn-112-make-the-standalone-activity-scala.8 Reorganize the standalone activity feature by subject and declaration kind

Touches: [model/temporal/standaloneactivity/**, model/temporal/taskqueue/**, model/gate/Roots.scala, model/README.md, .plans/UMPIRE_MODULES.md, model/ir/**, model/cases/**]

## Description
Move the now-deduplicated feature into the specified Model/Properties/Queries layout and finish vocabulary cleanup under the stable-name rules.

**Size:** M
**Files:** model/temporal/standaloneactivity/**; gate roots/pins; README and module map.

### Approach
- Create top-level, admission and compositions Model/Properties/Queries files; finish the shared temporal/taskqueue layout from task 12, omitting genuinely empty files. Delete System.scala and Claims.scala; no activity-local dispatchqueue folder remains.
- Group feature vocabulary under Product, Protocol and Admission objects and reuse the shared taskqueue vocabulary. Remove protocol/admission step prefixes and resolve completed/completes/completion collisions.
- Expose each machine's status sets as named defs on its vocabulary object (`Product.terminal`, `Product.paused`, `Product.running`, `Protocol.held`, the admission record's on `Admission`), moving the named defs task 6 kept, so no `in(…)` list is repeated at a use site and the shared-claim defs and the realization read those names (spec R10; `.plans/SEMANTIC_PROTOCOLS.md` "What fn-112 should do now", item 2).
- Put the three parameterized shared-claim defs of task 7 in the top-level `Properties.scala`, beside `admission/` and not inside it, since they are declared on the record and on both composition families.
- Keep the parent package in subfolders where it compiles without synthetic owner collisions. Apply task 2's one-per-former-owner DefinitionScope pins so actions, monitors, assumptions, channels and realizations keep exact IDs after moving under vocabulary objects; never add per-declaration ID strings.
- Remove worker.worker stutter and the workerStop alias; retain Delivery result text while resolving its dead Scala declaration.
## Acceptance
- [ ] The final directory/file matrix matches R11, includes fn107 models and has no System.scala, Claims.scala or empty placeholder file.
- [ ] Vocabulary objects and declaration names meet R10/R14 with no worker.worker stutter; each machine's status sets are named defs on its vocabulary object and no `in(…)` phase list is repeated at a use site.
- [ ] The three parameterized shared-claim defs live in the top-level `Properties.scala`, beside `admission/`.
- [ ] DefinitionScope pins reproduce every task-1 symbol ID after relocation; all outputs match R1's original baseline plus its exact authorized metadata delta.
- [ ] Feature source is trending toward the 1,600-line and 60-literal closing targets; both standalone-only and combined standalone-plus-shared-queue counts are recorded, so moving code alone does not count as simplification.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
