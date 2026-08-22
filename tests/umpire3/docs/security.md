# Umpire3 security boundary

Treat Experiments, descriptors, Participant programs, mutation corpora, worker output, and Replay
bundles as hostile input. Strict decoders reject unknown fields and size/depth violations before
allocation. Typed values reject sensitive field names; payloads and endpoints are retained as
digests or redacted identities. Authentication is never serialized into a profile.

Replay bundles are versioned, size bounded, strict-decoded, and bound to the canonical Experiment
digest. They contain a redacted Result, required Deployment profile/capabilities, seed, bounds, and a secret-free
command; they never contain API keys. Replay still requires an operator-supplied endpoint, namespace,
isolated task queue, build attestation, and any credential through process environment.

Production authority is separate from local capability. Canary execution requires an immutable v3
approval bound to experiment, catalog, and profile digests; tenant and namespace isolation; action,
fault, and destructive-operation allowlists; count, rate, concurrency, duration, evidence, output,
and cleanup budgets; and persisted recovery intent. The controller owns a pinned Ed25519 approval
authority and verifies its signature before execution or resumed cleanup; an approver name and
self-computed digest are not authority. Worker processes receive only explicitly passed
environment, run with signed CPU and memory limits, and are terminated as a process group on
deadline, memory, CPU, or output exhaustion.

External qualification receipts bind the canonical candidate release, Experiment digest, exact
Result bytes, evidence digest, build, and configuration identity. Release promotion accepts only the
candidate's required external profiles, requires one shared Experiment across them, and embeds a
self-checking digest of every receipt. Each candidate gate pins a reviewed authority identity and
Ed25519 public key; the exact binding is signed by the corresponding external PKCS#8 private key and
the signature is retained and reverified by the qualified release. An absent authority, an unsigned
receipt, or a signature from another key fails closed. Receipts establish only the reviewed deployment
authority that produced their Result; they do not upgrade omitted or inconclusive evidence.

Do not accept arbitrary Lean source, descriptors, executables, or approvals from a canary caller.
Review every generated protobuf disposition and every capability expansion as a security change.
The generic Temporal Environment adapter has no fault authority. Restricted faults require both explicit Execution
opt-in and a Deployment-profile-owned realizer that reports positive scoped occurrence evidence; absence of a
realizer is unsupported, never a simulated success.
