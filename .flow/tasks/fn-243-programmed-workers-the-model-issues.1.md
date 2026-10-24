---
satisfies: [R1]
---
# fn-243-programmed-workers-the-model-issues.1 Program on Reset: protocol, Runtime, Realize, WorkerProgram envelope and the activity rows

## Description
Add `ResetResponse.program` (optional Any) and `PROPOSAL_KIND_READ = 3` to `umpire.model.v1` with wire goldens; `Realize` gains a program hook with a refusing default and `answer` gains a refusing default; the Runtime admits Read proposals (pending with no action, cleared by a correlated final Explained, as Bootstrap is). Define `umpire.temporal.v1.WorkerProgram` (guards, ordered rules of key Any to respond Anys), `ActivityTaskKey { scenario, attempt, activity_type }`, the hold instruction field, and the `umpire-scenario` header name; remove `ObserveActivity.worker_tasks`. The Temporal Realization derives the program from its one reply policy by enumerating scenarios and admitted attempts, stamps the scenario into the Start header instead of the input, proposes Read and Cancel (with a seeded hold), and drops its Answer implementation. Add `umpire program --model --world --out`. Coverage unit test: every scenario and admitted attempt has a row. Executors are not in this task.

Placement: after fn-228.6 and fn-244.2 land; one lane on the Driver.
## Acceptance
- [ ] Deterministic program on Reset for the admitted World; wire goldens pin `ResetResponse.program` and `PROPOSAL_KIND_READ`
- [ ] `umpire program` writes the program file; Runner and Driver never decode it (import graph)
- [ ] Scenario in the `header`; Read proposals; `worker_tasks` gone; Answer dropped from the Temporal Realization
- [ ] Coverage test: every scenario and admitted attempt has a row
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
