# Retained lifecycle checkpoint

`original/` contains all 15 activity expectation files reconstructed from Git commit
`af0ebafc571d33651fdf2c008d4415ad72467a0d`, verified against
`.flow/artifacts/fn-227-parameterized-models-declared-worlds/original-artifacts.sha256`.
`manifest.json` retains their original paths and hashes; `verify.py` checks those hashes.
The source remains in Git at that pin; this tree contains no executable second Model.
The old lift refusal is intentionally retained, even though current source lifting admits activity.

`checkpoint.rs` parses the independent old states and rows. Its comparison World selects
maximum_attempts=1, atomic cancellation, one-tick schedule-to-start and start-to-close,
and absent schedule-to-close and heartbeat timeouts. Before projecting each start,
source and destination to phase, it requires attempt=1, cancel_requested=false,
schedule-to-start Remaining(1) exactly in Scheduled, start-to-close Remaining(1)
exactly in Started, and absent global and heartbeat budgets. Terminal states have
all four budgets absent; exhausted zero is cleared, never retained.

The action projection maps Poll(1) to Poll and both RespondFailed retryable assignments
to RespondFailed. Tick maps to ScheduleToStart only from Scheduled with exactly the
singleton ScheduleToStart expiry fact, or StartToClose only from Started with exactly
the singleton StartToClose fact. Each Tick must be Accepted and end in TimedOut;
only then is its validated expiry fact erased. Every other edge requires empty facts.
Outcomes, destinations and choice=0 remain exact; unexpected rejections cannot collapse
into NotFound. Canonical sets deduplicate payload assignments with the same projected
source/action/outcome/facts/destination, comparing semantic relationships rather than
catalog evaluation counts or BFS IDs.

The only erased edge is Worker::Heartbeat from Started when it is Accepted, has no
facts and leaves the full Model unchanged. No general self-loop erasure applies.
Current Poll while Started and terminal worker actions remain disabled. RespondCanceled
has no emitted edge in this one-attempt atomic World: no pending request can exist and
no stale attempt is in its catalog. An emitted acknowledgement fails the projection.

The test compares exact starts, all six reachable states, and all eleven projected edges,
including ScheduleToStart and StartToClose to TimedOut and four terminal Cancel
Rejected(NotFound) loops. The old report independently retains starts=1, reached=6,
emitted=11, evaluated=36, refused=25 and depth=2. A mutation changes the old
ScheduleToStart destination and fails this same graph comparison. These retained counts
describe the original graph; the extended Model additionally emits the explicitly erased
Heartbeat stutter and evaluates payload assignments before projection.
