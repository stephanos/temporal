I’ll read the task spec first, then review the diff and focused tests for contract and consistency issues.
The main composition path and differential fixtures are present. I’m checking whether the tests cover the promised bare classed action references and whether synchronization validates input domains, beyond matching their keys.
Bare `when:` references currently require an exact catalog key, so `when: reply` cannot select `reply-ok` and `reply-error` as the spec requires. I also found that synchronization compares class spellings without checking domain types. The lint command failed because the read-only sandbox prevents it from creating a temporary file; I’m trying focused checks against the existing Lean artifacts.
The focused elaboration check reproduced both bare-reference failures: `reply` and `job.poke` are rejected as unknown actions despite having classed catalog entries. I’m finishing checks for mismatched input domains and key collisions that could overwrite rows during the walk.
1. **Severity:** P2  
   **Confidence:** 75  
   **Classification:** introduced  
   **File:Line:** `model/Umpire/Command/Syntax.lean:475`  
   **R-IDs:** [R3]  
   **Problem:** Bare classed `when:` references still require an exact catalog entry. Against the supplied fixture, both `when: reply` and `when: job.poke` fail with “unknown Model action,” because only their class-specific keys exist. This contradicts task .2’s explicit bare-reference requirement and prevents the planned all-classes Property. The tests cover classed references and classless bare references, missing this case.  
   **Suggestion:** Resolve bare action names to all corresponding class keys and enumerate their Property groups. Add tests for both synchronized and member-qualified bare classed actions.

2. **Severity:** P2  
   **Confidence:** 75  
   **Classification:** introduced  
   **File:Line:** `model/Umpire/Command/Compose.lean:173`  
   **R-IDs:** [R3]  
   **Problem:** The input compatibility check compares rendered class suffixes, not input domains: `classed.find? (·.2.2 != firstClasses)`. A focused elaboration test synchronized `Job.Reply` with a distinct `Mirror.Reply`, both declaring `ok` and `error`; the composition succeeded and generated an action taking only `Job.Reply`. Thus unrelated domains silently synchronize when their constructors have matching spellings, despite the promised domain-mismatch rejection.  
   **Suggestion:** Validate the ordered input-domain types using the registered action metadata before constructing candidates. Add a mismatch fixture whose distinct domains share constructor spellings.

3. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/Umpire/Command/Compose.lean:287`  
   **R-IDs:** [R3, R4]  
   **Problem:** `found.insert (stateKey source, candidate.key)` assumes composed keys are unique without validating that assumption. Both `["a_b", "c"]` and `["a", "b_c"]` produce `a_b_c`; underscores are legal in member values. A focused walk reproduced overwritten rows: two self-looping members produced transitions from the first state to the second. Similarly, a sync named `job_poke` can duplicate the unsynchronized `job.poke` class keys. The resulting catalogs contain duplicate keys and rows no longer faithfully represent member behavior.  
   **Suggestion:** Reject composed state/action key collisions with located diagnostics before indexing rows, and use structural state identities internally. Pin both collision cases.

Targeted in-memory Lean checks reproduced these findings. The focused Lake build did not complete; `lint-model` failed because the read-only sandbox prevented temporary-file creation.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Worker entity belongs to task .1. |
| R2 | deferred | Uniqueness lint belongs to task .1. |
| R3 | partial | Command and unions exist; bare references, domain checks, and key collisions need fixes. |
| R4 | partial | BFS and fingerprint tests exist; state-key collisions overwrite rows. |
| R5 | deferred | Agreement proofs and checks belong to task .3. |
| R6 | deferred | Machine derivation belongs to tasks .4/.5. |
| R7 | deferred | Shipped claims belong to tasks .3/.5/.7. |
| R8 | deferred | Shipped composition counts belong to tasks .3/.5. |
| R9 | deferred | Regression verification belongs to later gates; fixture paths are unchanged here. |
| R10 | deferred | Existing Query regression verification belongs to tasks .4/.5. |
| R11 | met | Both set resolution paths reject compositions; fixture pins the Query path. |
| R12 | deferred | Authoring documentation belongs to task .6. |
| R13 | deferred | Governance documentation belongs to task .6. |
| R14 | partial | Fixture Queries appear in the differential, including composed `find`; shipped Queries are deferred. |
| R15 | partial | Member fields are lowered; field-addressed enumeration is explicitly deferred to task .7. |

Unaddressed R-IDs: []

Classification counts: 3 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":3,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict>