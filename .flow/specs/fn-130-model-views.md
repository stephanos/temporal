# Model views

## Goal

Help engineers understand a Model and review changes to it. Every Model gets a small, fixed set of rendered views. Each view is checked in as a `.d2` source plus an `.svg`, regenerated and gated like `model/ir` and `model/cases`, so a pull request that changes behaviour also shows the change in a picture and in a reviewable text diff. Source: `.plans/MODEL_VISUALIZATION.md`.

**Deferred** by the owner on 2026-10-05, before task planning. When revived, start after fn-126 closes, so views render the final layout, IDs and names.

## Decisions taken from the research

- **Renderer.** D2 used as a Go library, with the ELK layout engine (or dagre), version pinned. Deterministic content-hashed IDs, `OmitVersion`, and a fixed theme and font. TALA is open source since D2 0.9.0, but its layout is randomized by design, so it is used at most for the signature view.
- **No DSL declaration.** Views follow convention and are a pure function of the IR, read through the Go reader and lint tables. They are never IR data. Authors declare nothing.
- **Review artifact.** The `.d2` text diff is the reviewable delta. The SVG is a visual aid and is marked `linguist-generated`.
- **Interactivity.** SVG `<title>` tooltips only in checked-in SVGs. GitHub strips scripts. An HTML viewer is optional and later.

## Requirements

- **R1 Views:**
  1. Signature per IR file: who can do what to which entity.
  2. Phase diagram per machine. Edges are action classes. Hover shows facts, `because`, guards and the source line; a phase's hover shows its disabled pairs with their lint kind and reason.
  3. Refinement: System phase to Product phase.
  4. Composition and syncs.
  5. Derived-design diff against the inferred base.
  6. Query witness path as a sequence diagram (Mermaid text).
  7. A generated overview README per IR file.
- **R2 Generator.** A Go package `tools/umpire/render` and a command `umpire-render --check|--update`. It is run by `umpire-gen-model` and checked by the model gate. Artifacts go under `model/views/<irfile>/`.
- **R3 Projection.** Machines without a phase field (the close policy) get a usable projection. Either the lifter emits the `Rules(_.phase)` projection as an optional IR field, or the view uses a two-field label. Decide and record.
- **R4 Stability.** The same IR renders byte-identical artifacts. Font-subset diff noise is measured and contained (fixed font file, or stripped `@font-face`).
- **R5 Boundaries.** `render` imports only the reader and lint; the ownership test permits D2 there only. The gate's added time stays within a few seconds.

## Open questions

- Whether tooltips fire in GitHub's blob and rich-diff views (unverified).
- Doc comments on hover need a `doc` field in the IR (deferred idea).
- An optional HTML viewer (hover panels, source links, a witness player) on GitHub Pages.

## Outline

- **Spike (1–2 days):** render `activityProduct`'s phase view twice, then check byte stability, font noise and PR rendering.
- **Core views and gate (4–6 days).**
- **Derived views (3–4 days).**
- **Optional HTML viewer (3–5 days).**
