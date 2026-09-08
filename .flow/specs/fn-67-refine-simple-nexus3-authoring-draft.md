# Refine simple Nexus3 authoring draft

> HTML render lens: `.flow/artifacts/fn-67-refine-simple-nexus3-authoring-draft/spec.html` — open locally; regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Overview
Move the standalone Nexus authoring draft to Nexus3 and resolve its design review while keeping the feature-facing presentation simple. This is a design iteration, not implementation of the proposed syntax or a runtime compiler.

## Requirements
R1. Move the Nexus2 authoring draft to Nexus3/Nexus.md and preserve its teaching comments, updating them with changed meaning.
R2. Distinguish requests from observed progress, expose separate transition outcomes, and make terminal consistency model admission responsibility.
R3. Scope model progress to the selected operation and specify unsupported runtime lowering explicitly, without converting model steps to time.
R4. Derive IDs from feature namespace, declaration kind, and name, with optional per-declaration compatibility overrides; keep Case bindings, correlation, and rejection in Integration.md.
R5. Validate draft consistency and record applicable checks; preserve unrelated work and do not commit.

## Design
Keep the readable proposed model in Nexus.md and proposed Case mapping and identity convention in Integration.md. These design artifacts use Lean-like syntax but are not compiled modules. No parallel identity registry: IDs survive builds, comment edits, and reordering; renames change IDs unless individually overridden. Existing Nexus2 modules remain available for later reuse; do not adopt their different setup/cancellation semantics implicitly. Case integration for the broader cancellation draft remains explicitly unimplemented; the checked success slice is delivered. Unsupported cases cannot produce success by attaching a Known Gap.


## Current delivery reconciliation

The design-only scope and R1–R5 remain intact. The existing completed task records the original draft iteration; it is not evidence that all broader syntax or cancellation runtime behavior shipped.

- R1: the draft moved from `Nexus2/Nexus.lean` to `Nexus3/Nexus.md` in commit `5b53d5725`; the retained draft still explains vocabulary, ordinary Lean syntax, transitions, Properties, Behaviors and Queries with teaching comments. Existing Nexus2 modules remain separately available.
- R2/R3: current `Nexus.md` and `Integration.md` separate commands from correlated observed progress, admit distinct terminal outcomes, require terminal closure at model admission, scope counting to the selected operation and reject unsupported lowering. Fn-68 implements the success slice; fn-78 implements generic scoped counting. Neither establishes the deferred cancellation draft's Target, adapter or Case.
- R4 remains partially uncovered: name/kind/namespace-derived identities and rename/reorder/comment behavior are specified, but the broader draft no longer specifies its optional per-declaration compatibility override. Fn-68 explicitly omits overrides from the success demonstration. Reconcile those scopes in the retained design before claiming fn-67 complete; do not add an identity registry or change the executable demonstration merely to satisfy historical draft wording.
- R5: read-only current consistency checks pass for relocation, teaching sections, no registry, four action bindings, named Property references, operation-scoped bounds, terminal admission and unsupported rejection. Evidence: `/tmp/fn67-current-draft-consistency.json`; R2/R3 source mapping: `/tmp/fn67-r2-r3-reconciliation.md`. These are draft checks, not compilation or runtime qualification. Historical task evidence remains preserved.

The remaining documentation pass must also distinguish delivered generic counting from unsupported cancellation-specific forms where `Nexus.md` still calls counting proposed, and qualify references to absent cancellation support against the separate historical Nexus2-derived `Cancellation.lean`. It must not resume fn-79. The historical blanket statement that Case integration is unimplemented applies to the broader draft; fn-68's checked success Producer is now delivered.

## Remaining delivery

One documentation reconciliation follows the historical completed draft iteration. Model authors are the affected readers; executable APIs, runtime behavior and operator workflows do not change.

The broader draft retains an optional explicit compatibility ID override for an individual declaration. Ordinary namespace/kind/name derivation remains the default. Overrides do not cascade to owned declarations; a renamed child's identity needs its own explicit override if compatibility is desired. An override preserves identity only: it cannot suppress a changed Behavior Fingerprint, freeze source-bound provenance or bypass malformed, duplicate, conflicting or wrong-kind identity admission. The delivered success syntax still accepts no author identity or version inputs. Describe the distinction without introducing executable override syntax, aliases or a registry.

Generic operation-scoped monitoring and checked lowering are delivered. The scheduled-only cancellation draft still lacks its qualified Target, correlated adapter, operation capability and Case; the separate historical already-started cancellation Target does not satisfy that contract. Preserve command-versus-confirmation, alternative terminal outcomes, terminal admission, same-operation counting, trigger-step semantics and whole-Case unsupported rejection.

### Boundaries

No runtime or executable declaration changes, cancellation implementation, fixture regeneration, new checking framework, API drift gate or CI expansion. Fn-79 remains deferred until explicit user resumption. Existing context cancellation and bounded cleanup remain required.

### Validation and early proof point

The follow-up task first checks that both identity policies can be stated together without changing the success demonstration. If they conflict, revisit the wording and scope distinction rather than implementing overrides or deleting R4. Record positive and negative draft-consistency checks, local-link and whitespace checks, and preservation of existing teaching material. Any Lean edit is confined to the stale module documentation and is checked through the existing focused build; no runtime qualification is implied.

Quick commands: `git diff --check`; `cd model && mise exec -- lake build Temporal.Feature.Nexus3.Tests` when its module documentation is edited. Exact task-scoped draft checks are recorded with the implementation evidence.

### Requirement coverage

| Req | Task coverage | Remaining proof |
| --- | --- | --- |
| R1 | Historical task 1; follow-up task 2 | Preserve relocated draft and teaching comments. |
| R2 | Historical task 1; follow-up task 2 | Preserve commands, outcomes and terminal admission. |
| R3 | Historical task 1; follow-up task 2 | Reconcile delivered generic counting with unsupported cancellation lowering. |
| R4 | Follow-up task 2 | Restore optional broader-draft compatibility overrides without changing the success syntax. |
| R5 | Historical task 1; follow-up task 2 | Record current consistency, preservation and applicable checks; no commits. |

No unresolved design questions or new cross-spec dependencies are required. Fn-68 and fn-78 supply delivered reference behavior; fn-79 is a scope boundary, not a prerequisite.
