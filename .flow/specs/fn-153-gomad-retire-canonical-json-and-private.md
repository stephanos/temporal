# Gomad: retire canonical JSON and private atomic writes

## Conversation Evidence

> user (turn 4): "no need for  byte-for-byte compatibility requirements"
> user (turn 7): "we want lightweight and fast"
> user (turn 9): "yes write append-only log as new flow spec; (ie 4-8)"
> user (turn 10, selected): "Split as proposed"

Agent recommendations that turn 9 selects and turn 10 assigns to this spec, as listed in turn 8's answer:

- (7) Fold the per-package atomic-write copies into the shared host-filesystem replace helper.
- (8) Replace canonical JSON with strict standard-library decoding.

## Goal & Context

Gomad maintainers retire the generic canonical encoder and five private
whole-file publishers after fn-152 replaces Runner storage. The projected
surviving JSON inventory contains 44 production caller files. Reinventory the
landed predecessor before implementation.

The owner's compatibility waiver permits changing ordinary host JSON bytes,
formats and digest values. The old package also checks strings, duplicate keys
and numeric spellings. Those checks need semantic ownership rather than blanket
deletion. Identity inputs, capabilities, bounds, public APIs, transactions,
error classifications and resource lifetimes retain their existing contracts.

## Architecture & Data Models

**Ordinary encoding and input validation.** Domains use stdlib JSON directly
for persisted and hashed values. Preserve typed integer and decimal-string
bounds, complete identity projections, nil/empty distinctions, pointer presence
and semantic collection order. Check original in-memory strings through typed
domain validators before encoding when they guard an existing invariant.
A narrow pure strict decoder checks raw UTF-8 and duplicate object keys, then
uses DisallowUnknownFields and EOF decoding. It has no encoder, recursive
sorting, reencoding comparison or reflection-based string walker. Callers own
their byte/allocation bounds and JSONL framing.

Unknown fields fail in owned typed schemas. An intentionally projected target
record may carry other producer-owned fields; strictly decode its owned fields
without claiming that DisallowUnknownFields constrains a map. Validate embedded
I/O inventory against its actual producer contract, retaining its byte hash and
numeric/string invariants. Ordinary whitespace, key order, case aliases and
Unicode escape spellings may follow stdlib behavior when the old refusal was
only a spelling check. Required shape, identity and ambiguity checks remain.

**Wire isolation.** Existing World Encode entry points retain their closed typed
value domain. Encode typed values, decode that intermediate with UseNumber,
then use Encoder.SetEscapeHTML(false) and remove its single newline. Stdlib map
ordering preserves the existing nested wire ordering, including config-first
snapshot framing. Keep bounded preflight, semantic validation, exact wire
reencode/refusal checks, recording envelopes and transition JSONL. Add explicit
checks for terminal detail and nested replay request strings before encoding.
The disposable Go 1.27.1 Linux/arm64 probe matched 5,029 tested encodings across
76 serialized fields and identified seven invalid-UTF-8 divergences without
these checks. That finite probe supplies no native qualification evidence.

Live-capability host parsing retains producer JSON payload bytes, header layout
and raw payload authentication. Regenerate its source-derived producer identity
when validator inputs change, updating both producer and consumer together.
Simulation identity projections remain paired with the
target's stdlib formulas. Independent binary codecs, deterministic-I/O inventory
encoding and approved boundary-diff normalization keep their current owners.

**Shared file publication.** Extend the existing host-filesystem owner with one
staged-file handle. Staging fills a same-directory temporary file, applies the
requested mode, syncs and closes it. A path accessor allows exact-file
prevalidation; publication returns whether rename happened separately from its
error; cleanup errors join the primary failure. Existing byte-slice replacement
entry points delegate through the same owner. Callers retain directory guards,
permissions, locking, contextual errors and transaction order.

Qualification preserves its unique report name and returns that path after
publication even if directory durability fails. Source archives remain streamed
and checksum-validated before publication. Patch regeneration validates the
exact staged file before publication. The builder stages both launcher and
stamp before publishing either, preserving stamp-sync/failpoint followed by
launcher-sync/failpoint. Append, directory and immutable no-replace publication
remain separate operations.

## Edge Cases & Constraints

- Regenerate affected values from final source inputs. Retain each approval's
  reviewed semantic projection, exact module/source pins, owner, date, workload,
  platform and admitted facts. Retained platform discovery is not replaced by
  live discovery on the planning host.
  Prepare all retained requests with their new matching approval digests in a
  disposable staged root before generating/checking the complete batch. Publish
  validated outputs through the existing publication workflow; stale-approval
  refusal remains active throughout.
- Global execution SchemaVersion stays 1. Consume fn-152 storage/artifact lineage
  and current schema/hash/cache invalidation. No legacy decoder or universal
  lineage registry is added. Encoding changes do not authorize smaller identity
  projections, stale evidence acceptance or new capability admission.
- Increasing workload count changes repetitions, not codec or publication
  capability. Archives retain the existing download maximum and streamed memory
  bound. Decode/allocation and payload bounds remain domain-owned.

## Quick commands

```bash
go -C tools/gomad3 test -tags test_dep -count=1 ./internal/hostfs ./world
go -C tools/gomad3 test -tags test_dep -count=1 -run '^TestPackageArchitecture$' .
make -C tools/gomad3 validate
```

## Acceptance Criteria

- **R1:** Every whole-file atomic replacement in Gomad uses the shared host-filesystem owner; no private publisher remains. Streaming, staged validation, requested modes, caller guards, unique report paths and the builder's ordered two-file transaction retain their contracts. Errors: cancellation, fill/write/chmod/sync/close, validation, rename, directory durability and cleanup failures preserve primary error identity and publication disposition; failure before rename leaves the prior destination, and failure after rename reports publication without claiming durability.
- **R2:** The generic canonical-JSON package is removed. Persisted and hashed values use stdlib encoding with complete current-build identities, and affected committed digests/approvals/generated values are regenerated from final inputs. Owned typed decoding rejects unknown fields and trailing data; projected producer records retain their explicit field ownership. Errors: malformed/raw-invalid-UTF-8/duplicate input, invalid original domain strings, numeric/shape/bound violations, hash tampering and stale approvals retain refusal; ordinary spelling-only checks retire. World wire bytes, live-capability payload bytes/header layout, paired simulation formulas, bounded parsing, public API/error/stream contracts and capability scope remain unchanged; regenerated producer identity binds both source sets.

## Early proof point

Task fn-153-gomad-retire-canonical-json-and-private.5 proves the domain-local stdlib approach against real
Encode/Decode paths, config-first bounds, every transition variant, exact
recording bytes and invalid-string controls. If it fails, prove a typed World
projection before package deletion; relocating the recursive encoder or
changing the wire contract does not satisfy R2.

## Boundaries

- Campaign, exploration and corpus persistence belong to fn-152. Surviving
  exploration and corpus semantic identities are included here.
- No older-version compatibility layer, migration, new command, configuration
  surface, third-party library or newly admitted capability.
- Host/runtime wire redesign, global execution schema changes, append-log
  commits, directory/no-replace publication and direct-truncation provenance
  writes are outside this cleanup.
- Deferred native qualification, CI dispatch, PR creation and push remain
  outside current authority.

## Delivery and verification

Fn-152 remains the sole spec prerequisite. Fn-155 remains first for
implementation. Serialize overlapping fn-109/fn-110/fn-155 source owners and
shared Go, lint and generator gates; independent consumer owners may use
disjoint isolated worktrees after their dependencies land.

Each refactor owner captures a frozen semantic/identity/publication baseline.
Ordinary-byte goldens become semantic round trips, same-build repeatability and
identity-input/tamper controls. Wire byte fixtures remain exact. Generated
outputs follow the final source changes; architecture purity/effect fixtures
retain their positive and negative controls when the generic package retires.
Measure production/test code removed and replacement code added separately;
the capture's line-saving estimate is not acceptance evidence.

Retain focused negative controls, ordinary source coverage, fast and nested
lint, generated validation, architecture/public-signature checks, both static
source sets, preservation and independent source review. Run one full supported
host gate on the frozen integrated batch using the milestone workflow. New
fn-153 checks retain their own acceptance; the historical native transfer does
not automatically transfer them. Missing native proof stays disclosed with its
owner, and portable coverage does not establish a full test-host pass.

## Decision Context

- The existing stdlib boundary-normalization pattern and the finite World probe
  support a domain-only wire codec without a second generic serializer.
- A staged shared primitive handles the five surviving publishers while
  leaving transactions, validation and directory ownership with their callers.
- A module-wide rename sweep would conflate replacing files with immutable
  directory publication and target filesystem operations; those owners remain.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Every whole-file atomic replacement in Gomad uses the shared host-filesystem owner; no private publisher remains. Streaming, staged validation, requested modes, caller guards, unique report paths and the builder's ordered two-file transaction retain their contracts. Errors: cancellation, fill/write/chmod/sync/close, validation, rename, directory durability and cleanup failures preserve primary error identity and publication disposition; failure before rename leaves the prior destination, and failure after rename reports publication without claiming durability. | fn-153-gomad-retire-canonical-json-and-private.13, fn-153-gomad-retire-canonical-json-and-private.14, fn-153-gomad-retire-canonical-json-and-private.18, fn-153-gomad-retire-canonical-json-and-private.2, fn-153-gomad-retire-canonical-json-and-private.7 | — |
| R2 | The generic canonical-JSON package is removed. Persisted and hashed values use stdlib encoding with complete current-build identities, and affected committed digests/approvals/generated values are regenerated from final inputs. Owned typed decoding rejects unknown fields and trailing data; projected producer records retain their explicit field ownership. Errors: malformed/raw-invalid-UTF-8/duplicate input, invalid original domain strings, numeric/shape/bound violations, hash tampering and stale approvals retain refusal; ordinary spelling-only checks retire. World wire bytes, live-capability payload bytes/header layout, paired simulation formulas, bounded parsing, public API/error/stream contracts and capability scope remain unchanged; regenerated producer identity binds both source sets. | fn-153-gomad-retire-canonical-json-and-private.1, fn-153-gomad-retire-canonical-json-and-private.10, fn-153-gomad-retire-canonical-json-and-private.11, fn-153-gomad-retire-canonical-json-and-private.12, fn-153-gomad-retire-canonical-json-and-private.15, fn-153-gomad-retire-canonical-json-and-private.16, fn-153-gomad-retire-canonical-json-and-private.17, fn-153-gomad-retire-canonical-json-and-private.18, fn-153-gomad-retire-canonical-json-and-private.3, fn-153-gomad-retire-canonical-json-and-private.4, fn-153-gomad-retire-canonical-json-and-private.5, fn-153-gomad-retire-canonical-json-and-private.6, fn-153-gomad-retire-canonical-json-and-private.7, fn-153-gomad-retire-canonical-json-and-private.8, fn-153-gomad-retire-canonical-json-and-private.9 | — |
