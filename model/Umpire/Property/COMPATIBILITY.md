# Property compatibility boundary

Checked Property data has two supported semantic versions. Version 1 retains the original clause
vocabulary and its existing fingerprints. Version 2 admits guarded same-step cases and
trigger-frozen guarded bounded clauses. Admission rejects every other version with
`unsupported-property-version`, and guarded clauses under version 1 receive the same typed error.
The canonical semantic projection includes the version, requirements, capability meanings,
logical-time source, clause and nested identities, predicates, exceptions, group flags, patterns,
and resolved limits. Declaration and case order are canonicalized. Documentation and source
locations remain outside the Behavior Fingerprint while remaining present in checked metadata and
diagnostics. Explicit trace and enumeration order keeps its existing meaning in the owners that
define it.

The current consumers divide into three boundaries:

| Consumer | Property data consumed | Guarded-form decision |
| --- | --- | --- |
| `Umpire.Property.Check` and `Umpire.Property.Evaluation` | Complete authored and checked Property data | Admit versions 1 and 2, reject unknown versions, and evaluate version 2 through the authoritative checked predicate and temporal semantics. |
| `Umpire.Query.Language` and `Umpire.Planning.Engine` | Exact checked Properties; Query identity stores each Property ID and Behavior Fingerprint | Preserve the complete checked value for authoritative planning evaluation. Planning reports `property-evaluation-failure` when checked evaluation input cannot be produced. |
| `Umpire.Artifact.Planning` and `Umpire.Artifact.Codecs` | `PortableProperty` containing only Definition ID, Behavior Fingerprint, and requirement IDs | Preserve the guarded meaning by its new fingerprint. The v2 ExperimentSpec format is unchanged because it never contained a Property body. |
| `tools/umpire/internal/artifactv2` | The same three-field `Property` identity inside canonical ExperimentSpec v2 JSON | The retained generated-view reader validates the Experiment artifact version, exact fields, canonical order, digest syntax, checksums, and closure. It does not interpret Property clauses; unknown JSON fields and unsupported artifact majors fail at this narrow boundary. The retired runtime, evidence, result, and clone codecs have no Go compatibility surface. |
| `Umpire.Observation.Verdict` and `Umpire.Observation.Check` | Complete checked Properties after direct or translated Observation admission | Reject guarded same-step and guarded temporal clauses as `unsupported-property-clause`, retaining the responsible clause IDs. They do not flatten a guard or invoke a second evaluator. |
| `Umpire.Case.Compiler` | Already-lowered `ContractLowering` values | Reject an `unsupported` lowering with its source binding, source location, and construct. No checked-Property-to-`ContractLowering` producer exists, so the compiler is not a Property recognizer and no producer rejection can be tested from checked input yet. |
| `Umpire.Space`, `Umpire.Promotion`, and domain-specific consumers | Checked values or Property ID/fingerprint references | Carry and compare the exact fingerprint. They neither decode nor reinterpret Property clauses. |

There is therefore no generic Property JSON decoder, Property wire protocol, or universal Case
lowering to extend. A future consumer that needs the Property body must either use the checked
version 2 semantics or introduce a typed unsupported/version result before claiming success.
