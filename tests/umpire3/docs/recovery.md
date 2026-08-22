# Canary incident recovery

The controller persists recovery intent before starting a production worker. The record binds the
approval and experiment digests to tenant, namespace, and redacted resource identities. Cleanup runs
under an independent timeout even after cancellation, violation, worker termination, or controller
failure.

On restart, load the record by approval ID and call `ResumeCleanup` with the same signed approval,
pinned approval authority, profile definition, and least-authority worker environment. Resumption
rejects an approval, profile, tenant, namespace, experiment, or resource identity that differs from
the persisted record. Do not delete or edit a pending record.
Escalate digest mismatch, missing isolation metadata, or repeated cleanup failure to the owning
deployment team; preserve the audit sequence and worker failure class without copying credentials or
payloads into the incident.

Recovery is complete only after the worker reports cleanup complete and the controller removes the
record. Until then, the experiment is not conforming and the namespace must remain quarantined.
