# Revised engineering boundaries

The owner's three approvals remain those recorded in `bbf0835fbf`. This pass
adds no requirement or owner amendment. All seven original task IDs and R1-R7
retain their identity. Native .1 is verified done at closure `186d362f72`;
all new support tasks are todo, not implementation claims.

| Checkpoint | Owner | Consumes |
| --- | --- | --- |
| Native inputs | .2 | Existing .1 and .7 order retained |
| Independent source inputs and exact-payload fairness (I) | .8 | .2, .7 |
| Optional handler facts (F) | .9 | .7 |
| Nominal bounded optional values (O) | .10 | .2, .7 |
| Phase/rejection/closure-law source admission (P) | .11 | .8 |
| Request transport and same-invocation capture (T) | .12 | Completed parent prerequisites |
| Retry/refinement consumer | .3 | .2, .11 |
| Deadline/cancellation consumer | .4 | .3, .9, .10 |
| Transactional live admission and SDK proof | .5 | .4, .12 |
| Documentation and final evidence | .6 | .5 |

Splits separate substantive native/source or support/consumer frontiers, not
trivial sequential setup. Native inputs include compatibility-preserving value
capability and checked catalog propagation; independent source inputs own typed
payload identity, replay and exact concrete fairness. Neither is discharged by
the other's tests. Optional facts and nominal options adapt existing typed/SSA
machinery; neither introduces a collector, arbitrary ADT evaluator or reflection.
Phase/rejection admission preserves both existing closure laws before retries.

The earlier range note's automatic native-admission proposal was not backed by
an existing API. Machine has no generic World/state validator, World::new and
suite constructors are infallible, and checker invariants run after starts are
seeded. The final briefs instead share one range declaration with try_new and
contains. Explicit fixture/live option admission rejects invalid configuration;
source binding/decoding refuses invalid typed values. Deliberately bad native
starts/updates refute authored bounds invariants. No blanket native Refusal,
fallible World constructor or generic validity framework is promised. This
corrects an engineering proposal without weakening R6's Unsupported-plus-freeze.

Existing native fairness compares exact action values. A variant-wide
existential source operator would let a stale payload discharge another
payload's fairness. I therefore owns finite concrete specialization and a real
starvation control; P/.4 consume it, including rejection and retry cases.

The revised parent still awaits formal re-review on its original session and
counter. Overlapping source/guest/codegen/cache ownership remains serial despite
dependency-level parallel candidates. All canonical writes used Flow CLI;
drafts and read-only investigation reports remain ignored under .flow/tmp.
