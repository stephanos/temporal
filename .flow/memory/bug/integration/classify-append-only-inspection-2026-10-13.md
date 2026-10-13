---
title: Classify append-only inspection failures by error provenance
date: "2026-10-13"
track: bug
category: integration
module: tools/gomad3/runner
tags: [inspect, append-only, corruption]
problem_type: integration
symptoms: Semantic log corruption escaped structured invalid-store JSON diagnostics
root_cause: Post-replay validation mixed embedded log semantics with external reference resolution
resolution_type: fix
---

## Problem
Read-only inspection recognized syntactically valid append-only stores, but some semantic failures discovered after replay escaped as ordinary errors. That broke the structured invalid-store contract for JSON callers even though the bad state was entirely encoded in the inspected log.

## What Didn't Work
Classifying frame and replay failures alone was insufficient. Post-replay validators mixed semantic checks on embedded records with resolution of borrowed artifacts, so wrapping the entire call would also have mislabeled external filesystem failures as log corruption.

## Solution
The runner now marks semantic failures attributable to merged-log envelope and embedded source bytes with a dedicated corruption marker, while validating embedded source state separately before resolving external borrowed evidence. Focused tests mutate checksummed records and require an empty validated prefix in the structured invalid-store result.

## Prevention
For append-only inspection, enumerate errors by provenance: framed bytes, post-replay semantic state, and external references. Test each log-owned semantic branch through the public inspection result, and keep a separate negative assertion that external evidence failures do not become structured log corruption.
