# Refine simple Nexus3 authoring draft

## Overview
Move the standalone Nexus authoring draft to Nexus3 and resolve its design review while keeping the feature-facing presentation simple. This is a design iteration, not implementation of the proposed syntax or a runtime compiler.

## Requirements
R1. Move the Nexus2 authoring draft to Nexus3/Nexus.md and preserve its teaching comments, updating them with changed meaning.
R2. Distinguish requests from observed progress, expose separate transition outcomes, and make terminal consistency model admission responsibility.
R3. Scope model progress to the selected operation and specify unsupported runtime lowering explicitly, without converting model steps to time.
R4. Derive IDs from feature namespace, declaration kind, and name, with optional per-declaration compatibility overrides; keep Case bindings, correlation, and rejection in Integration.md.
R5. Validate draft consistency and record applicable checks; preserve unrelated work and do not commit.

## Design
Keep the readable proposed model in Nexus.md and proposed Case mapping and identity convention in Integration.md. These design artifacts use Lean-like syntax but are not compiled modules. No parallel identity registry: IDs survive builds, comment edits, and reordering; renames change IDs unless individually overridden. Existing Nexus2 modules remain available for later reuse; do not adopt their different setup/cancellation semantics implicitly. Case integration remains explicitly unimplemented; unsupported cases cannot produce success by attaching a Known Gap.
