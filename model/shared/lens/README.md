# Shared Temporal entity lens

`EntityLens<B: EntityBinding>` owns the existing Model dispatch transaction together with entity admission,
immutable receipt, pending bootstrap/instruction, freeze and Reset. `prepare` constructs an opaque
prospective change; the existing guest publishes a response before `commit` installs that change.
The first admitted Model is prepared separately and replaces the whole old runtime at commit.
Failed response publication cannot install a receipt, configuration or pending request.

The binding supplies typed bootstrap decoding/correlation, independent Model construction,
actual evidence validation, supported-domain checks and committed proposal context. It retains
all feature phase, World, scenario, worker-response and finality decisions. `Read` and `Entity`
retain their registry-facing declarations; their old registry paths re-export the shared types.
`support` owns the validated RPC envelope and transport-envelope constructors. Baseline routing
owns no catalog gap reconciliation or mutation policy.

The activity migration adapter remains in `realization/src/live.rs`; its probe baseline policy
lives under `feature/activity/probes.rs`. Moving these adapters does not reclassify feature
semantics as shared code. This source checkpoint does not implement the later feature registry
migration, compatible-frontier task, independent protocol refinement or worker-policy enumeration.

`entity_lens_equivalence_*` compares the frozen pre-extraction facade and the new lens on the
same actual messages, including pending/read and failed-publication cases. Probe controls compare
all current contextual baseline bytes against the frozen pre-extraction constructor. Boundary
controls refuse feature imports and representative phase/World/scenario leakage. The probe
inventory refuses any method set that differs from the registered baseline fillers; catalog keeps
probe coverage reconciliation. `probe_fillers::registered` routes an action baseline through the
same registry `Fill` used by proposals and workers; the activity adapter adopts it when its
feature declaration moves to the registry. The full fresh-guest/live gate remains later work.
