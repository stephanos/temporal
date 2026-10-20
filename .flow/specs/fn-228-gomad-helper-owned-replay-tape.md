# Gomad: helper-owned replay tape validation

## Goal & Context

Exact replay validates the choice tape in two places. The Runner decodes the retained choice trace, projects it into a replay plan, checks the plan digest, serializes the plan into JSON tape chunks, and after exit decodes the helper's choice bytes a second time to compare digests and counts. The Rust helper independently decodes the chunks and performs the per-decision comparison. The Go reference session keeps a third per-decision comparison that runs only under split-owned differential mode and the conformance corpora.

This spec makes the helper the single owner of replay-time tape validation. The Runner hands it the retained trace bytes with the expected identities; the helper projects, validates, replays and reports. The Runner keeps one codec for inspection, identity and publication, and the Go reference session stays as the differential oracle. Rigor is business-critical for evidence integrity: a replay that reports a match must be exact, and nothing here may weaken that.

## Architecture & Data Models

The seam is unchanged: Go Runner, Rust helper per execution, JSON control frames plus binary payload frames on the existing pipe. What moves is the direction of the tape. Today Go sends projected decisions; after this change Go sends the retained trace and expectations, and the helper derives the decisions.

The helper gains a replay-plan projection equivalent to the Runner's: filter decision records, carry each select result's readiness onto the poll decisions of the same select, encode and hash the plan. The helper model configuration carries mode, limit, execution identity, expected trace digest, expected tape digest and expected decision count. The helper terminal gains the complete trace digest, record count and branching count.

A new conformance corpus holds the Go and Rust projections equal on retained trace fixtures, regenerated like the other corpora: two passes, byte-identical.

## API Contracts

Helper replay request, replacing tape chunks: `{mode, limit, execution_identity, trace_sha256, tape_sha256, decisions, trace_frames}` where `trace_frames` is a count of bounded binary frames that follow, each at most the choice limit. Helper preflight reply on failure: `{error: "replay_preflight", check: "trace_sha256" | "trace_decode" | "tape_sha256" | "decisions"}`. Helper exit terminal adds `{trace_sha256, records, branching_records}` beside the existing fields. The divergence terminal shape is unchanged.

## Edge Cases & Constraints

Record contract, choice trace wire format, replay plan encoding and every identity are unchanged; an artifact published before this change replays after it with the same result. Trace frames over the limit are refused before allocation. A helper and Runner from different lineages refuse each other through a request schema version rather than misreading frames.

## Acceptance Criteria

- **R1:** When the Runner replays an artifact with a choice profile, it passes the retained trace bytes, trace digest, execution identity, expected tape digest and expected decision count to the helper and does not project a replay plan on the execution path. Errors: unreadable trace payload → replay preflight error, status unchanged from today; trace digest mismatch against the manifest → same.
- **R2:** Before the guest starts, the helper verifies the trace digest, projects the replay plan, and verifies the projected tape digest and decision count against the supplied expectations. Errors: any check fails → helper refuses to start, replies with the preflight error naming the failed check, and the Runner surfaces it as a replay preflight error with the same status code as today.
- **R3:** During exact and prefix replay the helper validates kind, site, enabled alternatives, alternative-set digest and selected rank at every decision, and reports tape exhaustion and unconsumed tape, exactly as today. Errors: first mismatch → divergence terminal with ordinal, reason, expected and observed decision; no new error surface.
- **R4:** On exit the helper terminal carries the complete trace digest, record count and branching count, and the Runner compares those three against the manifest without decoding the trace bytes a second time. Errors: any mismatch → the existing `choice_profile.trace.*` divergence strings, unchanged.
- **R5:** When a storage crash interrupts a replay, the helper reports the tape cursor and the Runner derives the unconsumed-tape divergence and the interrupted replay plan from that report with the same outcomes as today. Errors: cursor beyond tape length → invalid helper terminal, as today.
- **R6:** The Runner still decodes the choice trace for inspection, identity derivation, publication and minimization evidence. No error surface beyond the existing decoders.
- **R7:** For every artifact in the existing replay test corpora, replay produces the same match, divergence ordinal and divergence reason as before this change. No error surface beyond the corpora.
- **R8:** A conformance corpus of retained trace fixtures holds the helper projection and the Runner projection byte-equal with equal tape digests, and a second regeneration pass is byte-identical. Errors: a projection difference fails the corpus test; a non-identical second pass fails regeneration.
- **R9:** The Go reference session and the split-owned differential mode remain and keep routing runtime ops to it. No error surface beyond today.
- **R10:** Trace frames replace JSON tape chunks and are bounded by the choice limit. Errors: a frame over the limit → refused before allocation with an invalid-configuration error; a frame count that does not match the declared count → same.
- **R11:** The helper replay request carries a schema version. Errors: a version the helper does not implement → explicit refusal naming both versions, never a silent misread.

## Boundaries

- No change to the choice trace wire format, replay plan encoding, record contract or any identity.
- No removal of the Go reference session or the split-owned mode.
- No change to inspection, publication or minimization codecs.
- Passing the helper a path into the artifact store instead of frames is out of scope; the helper stays sealed from the store.

## Decision Context

The per-decision comparison already lives in the helper on the default path; this spec removes the surrounding duplication rather than the comparison. Frames were chosen over a store path because the choice limit already bounds the copy and the helper never touches the artifact store today. Rejected deleting the Go reference session as overkill: it is the differential oracle for the transcript corpus, and removing it would leave the helper checking itself.

## Quick commands

```bash
cd tools/gomad && make validate && go test ./internal/runner/... -run 'Replay' && (cd internal/backends/wasm/wasmhost && cargo test replay_plan)
```
