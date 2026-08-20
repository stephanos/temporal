# Umpire3 security boundary

Treat Experiments, descriptors, Participant programs, campaign corpora, worker output, and Replay
bundles as hostile input. Strict decoders reject unknown fields and size/depth violations before
allocation. Typed values reject sensitive field names; payloads and endpoints are retained as
digests or redacted identities. Authentication is never serialized into a profile.

Replay bundles are versioned, size bounded, strict-decoded, and bound to the canonical Experiment
digest. They contain a redacted Result, required Deployment profile/capabilities, seed, bounds, and a secret-free
command; they never contain API keys. Replay still requires an operator-supplied endpoint, namespace,
isolated task queue, build attestation, and any credential through process environment.

Production authority is separate from local capability. Canary execution requires an immutable
approval bound to experiment, catalog, and profile digests; tenant and namespace isolation; action,
fault, and destructive-operation allowlists; count, rate, concurrency, duration, evidence, output,
and cleanup budgets; and persisted recovery intent. Worker processes receive only explicitly passed
environment and are terminated as a process group on deadline or output exhaustion.

Do not accept arbitrary Lean source, descriptors, executables, or approvals from a canary caller.
Review every generated protobuf disposition and every capability expansion as a security change.
The generic Temporal Environment adapter has no fault authority. Restricted faults require both explicit Execution
opt-in and a Deployment-profile-owned realizer that reports positive scoped occurrence evidence; absence of a
realizer is unsupported, never a simulated success.
