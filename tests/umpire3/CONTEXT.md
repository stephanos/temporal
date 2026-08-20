# Umpire3 domain language

Lean and the generated semantic catalog are authoritative for resources, actions, properties,
capabilities, observations, evidence, faults, modules, and targets. The terms below name the
domain flow shared by models, authoring, execution, qualification, and operations.

| Term | Canonical meaning | Avoid |
| --- | --- | --- |
| Scenario | Sparse author intent before compilation. | Calling a compiled experiment a scenario. |
| Experiment | A validated, versioned, executable semantic artifact. | Calling arbitrary test configuration an experiment. |
| Regression | A test that compiles a Scenario and executes every resulting Experiment. | Using it as the name of the Scenario data structure. |
| Execution | One Experiment run against one Environment. | Generic “runtime” or “runner” when this operation is meant. |
| Environment | An adapter that realizes actions, observes evidence, and cleans up. | Using “environment” for a deployment profile. |
| Environment identity | The non-secret build, configuration, isolation, authority, evidence, and capability facts reported by the prepared Environment. | `EnvironmentProfile` or another competing Profile type. |
| Deployment profile | The validated maximum authority, capabilities, isolation, and attestation allowed for a deployment kind. | `Config`, `Definition`, and partial `Profile` values with overlapping meanings. |
| Participant program | Concrete Temporal commands used to realize an Experiment. | Generic “plan” outside participant internals. |
| Replay bundle | A redacted, digest-bound Experiment and Result plus reproduction metadata. | Generic `artifact.Record`. |
| Result | The evidence and semantic claim from one Execution. | Unqualified `Result` outside a module; use `execution.Result`, `canary.Result`, and similarly specific names. |
| Qualification receipt | The signed-off result of checking a candidate release against external execution evidence. | Generic report or artifact. |

Architecture documentation uses **module**, **interface**, **implementation**, **seam**, **adapter**,
**depth**, **leverage**, and **locality** consistently. “Temporal API” means the actual Temporal
protocol surface.
