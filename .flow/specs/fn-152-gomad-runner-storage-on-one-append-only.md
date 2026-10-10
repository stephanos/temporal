# Gomad runner storage on one append-only log

## Conversation Evidence

> user (turn 4): "no need for  byte-for-byte compatibility requirements"
> user (turn 4): "and if there is a library we could use, we could vendor it and only use what we need from it"
> user (turn 7): "what's pebble and can we use more of it? is sqlite the best pick? we want lightweight and fast"
> user (turn 8): "summarize actionable recommendations now"
> user (turn 9): "yes write append-only log as new flow spec; (ie 4-8)"
> user (turn 10, selected): "Split as proposed"

Agent recommendations 4–8 that turn 9 selects, as listed in turn 8's answer:

- (4) Replace the campaign journals, resume, recovery and merge with one append-only log: length-prefixed records with a checksum, fsync per commit, replay on open, truncate a torn tail; shard merge is log concatenation. Artifacts stay as files.
- (5) Unify the choice and simulation exploration journals.
- (6) Move the corpus onto the same log with an in-memory hash index.
- (7) and (8), atomic-write consolidation and canonical-JSON retirement, were split into the sibling spec at the user's choice.

## Goal & Context

<!-- Goal & Context: 55% [paraphrase], 45% [inferred] -->

Gomad's Runner persists campaigns, exploration rounds and the guided corpus
through a hand-built directory-tree database: per-strategy journals, sha-indexed
segments, staging and partial files, a resume archive, a recovery pass, a shard
merger, and private atomic-write helpers. That machinery combines crash recovery
with stable on-disk formats and bytes across versions.

The owner has dropped byte-for-byte and format compatibility. Old campaigns,
corpora and artifacts no longer need to resume or replay. That makes a much
simpler storage model possible. The owner wants the result lightweight and fast,
and evaluated SQLite, bbolt and Pebble before choosing an in-tree append-only
log.

This spec replaces the Runner's campaign and corpus persistence with one
append-only log and unifies the two exploration journals. A sibling spec,
"Retire canonical JSON and private atomic writes", follows it and finishes the
host-file cleanup. It runs alongside
fn-109, which restructures the same Runner code; fn-109's preservation clauses
for the replaced storage are superseded by this spec.

## Architecture & Data Models

<!-- Architecture & Data Models: 40% [paraphrase], 60% [inferred] -->

**Log.** One host-side module owns append-only logs. A frame contains a protected
length/kind header, a typed payload and its checksum. The reader validates header
integrity and checked, owner-specific length bounds before allocation. It validates
the complete typed transaction before applying it. Earlier damage, complete invalid
headers, unknown kinds and fully framed invalid JSON/state are corruption.
A short final frame or final payload checksum failure at a trustworthy frame
boundary is a torn tail. Coherent rewrites and checksum collisions are outside
this damage classifier.

Append writes and syncs one transaction before reporting success. Any partial
write or uncertain sync outcome poisons the writer until reopen. A complete frame
whose call failed may survive; replay decides its outcome. The caller advances
committed state only after append succeeds.

One writer holds the existing nonblocking host lock. Writer-open or recovery
repairs and syncs a recognized torn tail before further append. Read-only replay
captures EOF on one opened file, reports the validated prefix and observed tail,
and never truncates or acquires the writer lock. A concurrent truncation that
invalidates that snapshot produces a changed-snapshot error. A reader does not
claim that every visible complete frame received a durability acknowledgment.

The owner pins a private 0700 store directory and opens its 0600 regular,
single-link log and lock files relative to that root. Existing symlinks,
nonregular files, aliases, wrong modes and detected identity changes fail before
append or tail repair. The lock is bound to the validated store/log identity;
writes and truncation use verified descriptors, not reopened unchecked paths.
Read-only replay applies the same containment and identity guards without
creation, chmod, locking or repair. Creation sets modes only on newly owned
objects. These guards retain the supported Darwin/Linux filesystem contract,
not protection against arbitrary mutation by an equally privileged process or
new rules for intentionally shared artifact target links.

**Campaign state.** Campaign lifecycle, per-execution outcomes, failure policy
progress, seed ordinals and resume state are derived by replaying the campaign's
log into memory. One outcome transaction binds the selection ordinal,
classification, retention/novelty effects and failure-policy/controller progress.
Resume restores frozen selection and skips committed ordinals. Uncommitted
physical attempts may repeat; the final logical committed outcome set must match
the uninterrupted scripted reference, including parallel and policy-sensitive runs.

The same log retains an admitted-ordinal frontier bounded by configured
parallelism and terminal-result receipts for that frontier. Reservations commit
before launch. Actual completion evidence and owned payload references commit
before entering the ordering buffer; receipts do not increment outcome counters
or release active slots. Ordinary ordered outcome commits atomically reserve
the next permitted work, preserving incremental refill rather than fixed batches.

A stopping ordered receipt freezes admissions and selects every already
admitted outstanding ordinal as one stop group. First-failure requests the
existing cancellation; budget exhaustion lets active attempts finish. The
stopping outcome and every actual ordered drain outcome commit together after
all terminal receipts exist. Replay restores the frontier and receipts first,
derives any pending stop from the durable ordered receipt and committed policy,
then retries only admitted attempts without a durable receipt. Buffered real
successes/failures cannot become synthetic cancellations. Committed cancellations
restore their ordinals and counters; a committed stop admits no new seeds.
Stop-group acknowledgment waits for bounded active drain. Checked frame and
pending-payload bounds derive from existing component envelopes and parallelism,
not a smaller new aggregate cap or the final retained-success budget.

**Exploration rounds.** Choice exploration and simulation exploration commit
their rounds through one shared round-journal implementation, parameterized by
strategy, instead of two near-duplicate journals. One round transaction binds its
ordered executions and engine/frontier/controller transition. No second campaign
execution append follows it. Round/candidate/ordinal identity distinguishes work
even when candidates share one numeric base seed.

**Shard merge.** Merging shards combines their logs into one campaign log after
checking they belong to the same campaign plan. Full validation precedes output
creation. Source-scoped concatenation keeps each shard's initialization and
terminal records separate. The aggregate retains immutable source provenance
and borrowed artifact references; it never owns or sweeps shard payload roots.
The existing seed-only shard protocol, partial-merge option and ordinal/capacity
checks remain.

**Corpus.** The guided corpus is a log too. Its hash-keyed index lives in memory
and is rebuilt on open. Admission and eviction membership form one transaction.
A duplicate live hash is a no-op; committed eviction permits replay-verified
readmission. Streaming replay retains only the bounded live index. The existing
1,024-entry and 1 GiB live-payload budgets remain. Historical log bytes are
reported separately and grow until ordinary host storage failure; this spec
adds no history cap, compactor or hidden charge against the live-payload budget.

**Artifacts.** Executables, traces and transcripts stay content-addressed files.
An artifact file is published before the log record that references it.
Recovery first validates references required by the final committed live state,
then removes unreachable files only within an exclusively owned payload root.
Committed eviction retires corpus references. Prepared inputs and diagnostic
sidecars participate in reachability, including diagnostics without a retained
success artifact. Durable pending-result receipt references are live too. An
outcome transaction retires its receipt references and either adopts their files
as final evidence or makes them reclaimable. Payload files never become a second
authority for admission, receipts, policy or counters. Shared target pools and
borrowed shard roots retain their own ownership; a campaign cannot sweep them.

**Encoding.** New log records use the standard library JSON encoder with
strict decoding (unknown fields and trailing data rejected).

**Storage lineage.** Campaign, corpus and aggregate log envelopes, artifact
storage envelopes and portable plans declare the current storage format. Old,
missing or unknown lineage fails before mutation. Artifact lineage is checked
before decoding its current execution manifest. Global execution record and
runtime choice/World/I/O/simulation wire contracts remain unchanged.

**Limits.** Preserve semantic execution, artifact, frontier and partial-attempt
capacities. Retire storage-only segment grouping/count meanings explicitly in
detached reports, retaining caller fields where needed. Preserve the old
per-execution payload bound; derive whole-round framing bounds from its allowed
components rather than turning the former segment byte limit into a new round
cap. No segment-limit CLI flag exists in the surveyed dispatcher.

## API Contracts

<!-- API Contracts: 100% [inferred] -->

- The log module offers: open (replaying records to a caller-supplied
  consumer), append-and-commit of one record, and close. It reports torn-tail
  truncation and corruption as distinct outcomes.
- Runner's public operations (run, resume, merge, explore, inspect) keep their
  CLI commands and flags. Their on-disk layout and record formats may change
  freely.
- `inspect` (human and JSON output) reports campaign and corpus state read from
  the logs. Existing `inspect PATH` accepts a corpus directory through the same
  kind-based dispatcher, with no new command or flag. Healthy and observed-tail
  reports use stdout/status 0. Recognized corruption reports invalid store state
  and its last validated offset/count on stderr/status 2; `--json` encodes that
  diagnosis on stderr without also printing a plaintext duplicate. Report-write
  failure remains status 3. Generic invalid-path precedence and artifact/plan
  inspection remain unchanged.

## Edge Cases & Constraints

<!-- Edge Cases & Constraints: 30% [paraphrase], 70% [inferred] -->

- Crash during append: the torn record is dropped on next open; everything
  committed before it survives.
- Crash between artifact publication and its log record: the artifact is an
  orphan and recovery removes it.
- Two writers on the same campaign or corpus: the second fails with a clear
  "already in use" error rather than waiting or corrupting state.
- Merging shards from different campaign plans, or the same shard twice, is
  rejected.
- A campaign, corpus or artifact written by an older Gomad is rejected with an
  error naming the unsupported format; there is no migration path.
- No new third-party dependency is introduced for storage; the log is in-tree.
- Crash-safety testing keeps the ability to inject failures at chosen write
  points, now at the log's append and sync boundaries.

## Acceptance Criteria

- **R1:** An in-tree append-only log module provides open-with-replay, append-and-commit, and close; a committed record survives a crash, and a record whose commit had not returned may be lost but never half-applied. Errors: a recognized torn final frame or final payload checksum failure is reported and truncated by exclusive writer recovery; read-only replay reports it without mutation; damaged earlier records, invalid complete headers or typed payloads fail as corruption; a second writer gets an "in use" error; a changed reader snapshot fails without repair; invalid filesystem metadata, containment or identity fails without log mutation or repair.
- **R2:** Campaign lifecycle, execution outcomes, seed progress and resume state are stored only as records in the campaign's log; the previous journal, segment, staging and resume-archive files are gone. Errors: no error surface beyond R1 and R8.
- **R3:** A campaign interrupted at any commit boundary (including via injected failure at the log's append and sync points) resumes and finishes with the same set of executed seeds and classified outcomes as an uninterrupted run, with no seed executed twice after its outcome was committed. The set is logical committed ordinal/candidate outcomes; uncommitted physical attempts may repeat. Durable admissions and actual terminal receipts preserve outstanding/buffered work; normal refill remains incremental, and stopping outcome plus ordered drain commit atomically. Committed cancellations restore their ordinals and counters; stopped replay admits no new work. Errors: no error surface beyond R1.
- **R4:** Choice exploration and simulation exploration commit rounds through one shared round-journal implementation; no strategy keeps its own journal type. Errors: no error surface beyond R1.
- **R5:** Shard merge produces one campaign log from shard logs of the same campaign plan, and inspecting it reports the same totals as the shards combined. Errors: shards from different plans, or a duplicated shard → rejected before any output is written; invalid ordinals, partial coverage, bounds or required evidence also fail preflight before output creation.
- **R6:** The guided corpus is stored as a log with an in-memory live hash index rebuilt on open; admitting an already-present live hash is a no-op, and committed eviction permits readmission. Errors: no error surface beyond R1 and the existing live count/payload bounds.
- **R7:** Artifacts remain content-addressed files, each published before the log record that references it; recovery deletes owned files that the final committed live state no longer references, including live durable pending-result receipts until retirement. Errors: a missing or corrupt live referenced artifact, prepared input, pending-result payload or diagnostic sidecar is corruption and prevents orphan deletion; shared pools and borrowed shard payloads are never swept by the caller.
- **R8:** Opening a campaign, corpus or artifact written in a previous Gomad format fails with an error naming the unsupported format; no migration or legacy decoder remains. Errors: this criterion is itself the error case; missing, malformed or unknown new lineage also fails before mutation, including portable plans whose storage contract changed.
- **R9:** `inspect`, in human and JSON form, reports campaign and corpus state read from the logs. Errors: observed torn tail or corruption is surfaced with R1's distinctions and the stream/status contract above; report delivery failure returns status 3, and invalid paths retain prior precedence.
- **R10:** Storage adds no third-party dependency to the Gomad module. Errors: no error surface.

## Early proof point

Task fn-152-gomad-runner-storage-on-one-append-only.1 proves acknowledged-commit
survival, bounded framing, corruption/tail classification and read-only versus
exclusive repair through injected append/sync failures. If it fails, reconsider
the framing or transaction seam before integrating its dependent storage owners.

## Boundaries

- Fake network and fake filesystem in the runtime overlay are out of scope; research found no library worth adopting there. [paraphrase]
- SQLite, bbolt and Pebble are not adopted for Runner storage. [paraphrase]
- No migration of existing campaigns, corpora or artifacts. [paraphrase]
- Artifact payload storage (executables, traces, transcripts) stays as files; only their bookkeeping moves to the log. [paraphrase]
- Runner CLI commands and flags are unchanged; only on-disk formats change. [inferred]
- Network partition semantics and the fn-109 module refactors are separate work. [inferred]
- Retiring canonical JSON and consolidating private atomic-write helpers belong to the sibling spec. [paraphrase]
- Compaction, new corpus history limits, exploration shards and runtime wire-codec changes are outside this storage rewrite.

## Decision Context

<!-- Decision Context: 60% [paraphrase], 40% [inferred] -->

### Motivation

The owner removed byte-for-byte compatibility, which turns most of the Runner's
storage code into removable weight. They asked for something lightweight and
fast. [paraphrase] At capture the work was split in two: this spec owns the
log, and the sibling spec "Retire canonical JSON and private atomic writes"
runs after it so it does not migrate code this spec deletes. [paraphrase]
Alternatives were weighed against that priority:

- **Append-only log over SQLite:** SQLite (modernc) adds a large machine-translated
  library, several megabytes of binary and slow compiles; its strengths
  (ad-hoc queries, SQL merges) are not needed for one writer appending small
  records. [paraphrase]
- **Over bbolt:** bbolt is small and fast, but campaign state is append-mostly
  and rebuilt in memory; keyed in-place updates are unnecessary. bbolt remains
  the fallback if the corpus later needs keyed lookups too large for memory.
  [paraphrase]
- **Over Pebble:** about 166k lines with heavy dependencies and background
  goroutines, built for write volumes Gomad never reaches. [paraphrase]
- **Expected size:** roughly 3–4k fewer production lines and 2–3k fewer test
  lines, mostly in campaign and corpus persistence. This is an estimate, not a
  criterion. [inferred]

The source survey confirms R3 through frozen selection, ordered completions,
completed-ordinal filtering and policy restoration. Direct corpus dispatch in
R9 is a planning integration decision using the existing Inspector, rather than
a historical feature or a verbatim user request.

The repository consumer inventory found qualification pruning, diagnostic
sidecars, manual CLI fixtures and real kill/resume readiness probes. Their
migration belongs to the consumer task. This inventory establishes nothing
about code outside the repository.

Ten cohesive owners keep campaign-store conversion separate from Runner
transactions, and consumer migration separate from the four current contract
documents and final evidence matrix. Each of those combinations would produce
an L-sized task. Deletion and fixture edits count toward implementation scope;
the conductor must split again if the actual surviving validator surface grows.

Plan review identified filesystem guard and stopped-but-undrained recovery
gaps. Guarded writable opens retain the current root/file identity protections.
The separate bounded admission/receipt owner precedes Runner integration.
Atomic final stop grouping avoids an independently committed partial drain
cursor; durable receipts retain real buffered outcomes and live payloads across
pre-group crashes. Its costs are one receipt commit per completed attempt and
delayed stop acknowledgment while active attempts drain. Ordinary outcome/refill
reservations share a commit and preserve the current incremental scheduler.

## Quick commands

After implementation, run the portable storage smoke checks with the pinned Go:

```sh
go -C tools/gomad3 test -tags test_dep -count=1 ./runner/internal/logstore ./runner/internal/corpus
```

## Delivery and verification

fn-155 remains first for implementation. Its missing supported-platform proof
does not create a new dependency edge for this planning pass. The prerequisite
Runner decomposition has landed as reviewed source progress; its original
source acceptance stays open with its owner. fn-153 follows this spec through
its existing dependency. Serialize overlapping Runner/identity edits and all
shared Go, lint and generation gates.

Each task retains focused crash controls, semantic equivalence evidence and
applicable source/lint/architecture/static/generated checks. Compare scripted
uninterrupted execution with interruption at every enumerated append/sync point,
including uncertain complete frames, parallel attempts, both round strategies,
failure policies, cancellation, watchdogs and repeated deadline-bound resume.
Seed controls cover admission-before-launch, each terminal receipt, ordinary
outcome/refill, stop selection and group append/sync. Include buffered real
success/failure before a lower-ordinal stop, natural budget drain, committed
cancellations and missing/corrupt pending payloads. A host cancellation or
deadline retains its original error precedence, not a fabricated successful
stop group. Validate a complete transaction before applying any replay member.
Unchanged CLI/error/classification and identity-input semantics remain pinned;
old storage bytes and formats do not.

The final owner retains the current candidate's complete source matrix,
independent review, ordinary tests, fast and nested lint, generated validation
and affected both-source-set static checks. New storage and built-CLI crash
tests remain this spec's acceptance; they are not automatically transferred by
the older native manifest. Inherited deferred native qualification remains
unverified under its existing owners. Portable tests supply no full native
host pass. This plan authorizes no native revival, CI, PR or push.

## Parked unknowns

- Whether the corpus outgrows an in-memory index for realistic campaigns; resolved by measuring corpus size on the largest existing qualification campaign.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | An in-tree append-only log module provides open-with-replay, append-and-commit, and close; a committed record survives a crash, and a record whose commit had not returned may be lost but never half-applied. Errors: a recognized torn final frame or final payload checksum failure is reported and truncated by exclusive writer recovery; read-only replay reports it without mutation; damaged earlier records, invalid complete headers or typed payloads fail as corruption; a second writer gets an "in use" error; a changed reader snapshot fails without repair; invalid filesystem metadata, containment or identity fails without log mutation or repair. | fn-152-gomad-runner-storage-on-one-append-only.1, fn-152-gomad-runner-storage-on-one-append-only.3, fn-152-gomad-runner-storage-on-one-append-only.5, fn-152-gomad-runner-storage-on-one-append-only.6, fn-152-gomad-runner-storage-on-one-append-only.8, fn-152-gomad-runner-storage-on-one-append-only.9 | — |
| R2 | Campaign lifecycle, execution outcomes, seed progress and resume state are stored only as records in the campaign's log; the previous journal, segment, staging and resume-archive files are gone. Errors: no error surface beyond R1 and R8. | fn-152-gomad-runner-storage-on-one-append-only.3, fn-152-gomad-runner-storage-on-one-append-only.4, fn-152-gomad-runner-storage-on-one-append-only.7, fn-152-gomad-runner-storage-on-one-append-only.8, fn-152-gomad-runner-storage-on-one-append-only.9, fn-152-gomad-runner-storage-on-one-append-only.10 | — |
| R3 | A campaign interrupted at any commit boundary (including via injected failure at the log's append and sync points) resumes and finishes with the same set of executed seeds and classified outcomes as an uninterrupted run, with no seed executed twice after its outcome was committed. The set is logical committed ordinal/candidate outcomes; uncommitted physical attempts may repeat. Durable admissions and actual terminal receipts preserve outstanding/buffered work; normal refill remains incremental, and stopping outcome plus ordered drain commit atomically. Committed cancellations restore their ordinals and counters; stopped replay admits no new work. Errors: no error surface beyond R1. | fn-152-gomad-runner-storage-on-one-append-only.4, fn-152-gomad-runner-storage-on-one-append-only.5, fn-152-gomad-runner-storage-on-one-append-only.9, fn-152-gomad-runner-storage-on-one-append-only.10 | — |
| R4 | Choice exploration and simulation exploration commit rounds through one shared round-journal implementation; no strategy keeps its own journal type. Errors: no error surface beyond R1. | fn-152-gomad-runner-storage-on-one-append-only.5, fn-152-gomad-runner-storage-on-one-append-only.9 | — |
| R5 | Shard merge produces one campaign log from shard logs of the same campaign plan, and inspecting it reports the same totals as the shards combined. Errors: shards from different plans, or a duplicated shard → rejected before any output is written; invalid ordinals, partial coverage, bounds or required evidence also fail preflight before output creation. | fn-152-gomad-runner-storage-on-one-append-only.7, fn-152-gomad-runner-storage-on-one-append-only.9 | — |
| R6 | The guided corpus is stored as a log with an in-memory live hash index rebuilt on open; admitting an already-present live hash is a no-op, and committed eviction permits readmission. Errors: no error surface beyond R1 and the existing live count/payload bounds. | fn-152-gomad-runner-storage-on-one-append-only.6, fn-152-gomad-runner-storage-on-one-append-only.9 | — |
| R7 | Artifacts remain content-addressed files, each published before the log record that references it; recovery deletes owned files that the final committed live state no longer references, including live durable pending-result receipts until retirement. Errors: a missing or corrupt live referenced artifact, prepared input, pending-result payload or diagnostic sidecar is corruption and prevents orphan deletion; shared pools and borrowed shard payloads are never swept by the caller. | fn-152-gomad-runner-storage-on-one-append-only.2, fn-152-gomad-runner-storage-on-one-append-only.3, fn-152-gomad-runner-storage-on-one-append-only.4, fn-152-gomad-runner-storage-on-one-append-only.5, fn-152-gomad-runner-storage-on-one-append-only.6, fn-152-gomad-runner-storage-on-one-append-only.7, fn-152-gomad-runner-storage-on-one-append-only.9, fn-152-gomad-runner-storage-on-one-append-only.10 | — |
| R8 | Opening a campaign, corpus or artifact written in a previous Gomad format fails with an error naming the unsupported format; no migration or legacy decoder remains. Errors: this criterion is itself the error case; missing, malformed or unknown new lineage also fails before mutation, including portable plans whose storage contract changed. | fn-152-gomad-runner-storage-on-one-append-only.2, fn-152-gomad-runner-storage-on-one-append-only.3, fn-152-gomad-runner-storage-on-one-append-only.6, fn-152-gomad-runner-storage-on-one-append-only.7, fn-152-gomad-runner-storage-on-one-append-only.8, fn-152-gomad-runner-storage-on-one-append-only.9 | — |
| R9 | `inspect`, in human and JSON form, reports campaign and corpus state read from the logs. Errors: observed torn tail or corruption is surfaced with R1's distinctions and the stream/status contract above; report delivery failure returns status 3, and invalid paths retain prior precedence. | fn-152-gomad-runner-storage-on-one-append-only.8, fn-152-gomad-runner-storage-on-one-append-only.9 | — |
| R10 | Storage adds no third-party dependency to the Gomad module. Errors: no error surface. | fn-152-gomad-runner-storage-on-one-append-only.1, fn-152-gomad-runner-storage-on-one-append-only.2, fn-152-gomad-runner-storage-on-one-append-only.9 | — |
