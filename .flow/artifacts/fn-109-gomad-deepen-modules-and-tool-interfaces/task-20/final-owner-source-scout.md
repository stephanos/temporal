# Final lifecycle and handle guidance delta

Read-only `/root/final_owner_guidance_scout` extends the existing task-20 scout
only for finalized source candidates 16–18. Requested thinking-scout routing:
gpt-6.1-sol / high; actual execution metadata unobservable. Dispatch judge called
once, unavailable (no_key). No edits, tests, builds, generation, Flow/Git writes,
bridges or task-19 moving-source inspection. Native acceptance remains open.

## Minimal guidance changes

ARCHITECTURE.md process arbitration (currently around line 133): describe the
private progress owner applying complete typed transitions under the arbiter
mutex, validating participant incarnation, coordinator response identity,
epochs and aggregate arrival credits before consumption/reservation/wakes/
settlement. Coordinator response and model request identities are separate;
aggregate credits identify no individual operation. Model transport owns
pending/abandoned correlation and late-response discard, domain handlers model
commitment, coordinator/process owners completion and crash/reap lifetime.
Sources: simulation_progress.go:106,119,286; simulation_time.go:150;
simulation_model.go:30,192; simulation_unix.go:395.

ARCHITECTURE.md network discussion (around lines 75,115): Listener and Conn
select a private standalone, in-process simulation or process implementation at
creation. Conn direction locks cover complete Read/Write operations. Both
simulation mechanisms use the shared network model; process owners carry remote
identities and typed commands. Preserve backend-specific empty I/O, error
precedence, queued accepts, 64-KiB chunks and transcripts rather than implying
uniform behavior. Sources: gomadio/network.go:43,61,147;
simulation_handles.go:14; process_network.go:21; task-17 handover/source-audit.

ARCHITECTURE.md filesystem discussion (around line 653): three execution paths
use two Handle/Mapping representations. LOCAL holds filesystem/node/generation;
PROCESS holds typed-command identities and copied-byte cache state. Creation
selects the representation; facades delegate; local indexes hold concrete owners.
Path operations select their route independently. Sources: gomadfs/handles.go:7,28;
fs.go:84,210,352,421; local_handles.go:13; process_volume.go:16.

LOCAL identical mapping regions share a buffer and one byte charge; surviving
aliases retain the charge after its owner closes (local_handles.go:394,449).
PROCESS writable Map returns ENOTSUP before closed/access/bounds checks;
read-only bytes are copied/cached, nonnil cache returns without revalidation,
nil cache refetches (process_volume.go:177,194). Process ReadDir normalizes nil
entries to an empty slice (:169).

README.md deterministic-I/O mapping prose (around line 534) currently states
shared memory without LOCAL qualification: qualify it and add PROCESS limitations.
README test-simulation counts (around line 1275) are stale: replace counts with
seeded root toolchain and selected Runner transport coverage, including handle
cases, strict-delay watchdog exclusion and separate forward regression. Current
selection includes thirteen network and four filesystem process entrypoints
(Makefile:137–149), not evidence of native execution.

Keep SPEC's normative SIMULATION.BACKENDS/NETWORK/STORAGE/REPLAY unchanged:
three execution paths do not turn the public two-Backend seam into three
Backends. CLI needs no 16–18-specific change. TUTORIAL:350 retains its two-Backend
Fidelity explanation and can point mapping details to README.

ARCHITECTURE binary protocol guidance (around line 675): timewire.json generates
host/runtime layout codecs; descriptor I/O, native timers/quiescence and host
process arbitration stay handwritten. Reuse the existing task-13 guidance.
Keep D12 open, Darwin D14 resolved, collector prohibition and native gaps.
No new performance claim or final task-19 checker/public names are supported.
