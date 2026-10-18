---
title: Use file snapshots for reviewable external insta goldens
date: "2026-10-18"
track: bug
category: integration
module: model2/framework/samples/tests/kettle.rs
tags: [insta, goldens, review]
problem_type: integration
symptoms: Snapshot comparisons passed but cargo insta review could not accept changes
root_cause: Dynamic external references were passed through the internal inline snapshot adapter
resolution_type: fix
---

## Problem
The samples runner compared external text files by passing their contents to insta's internal InlineValue adapter. The comparison passed, but cargo insta review could not accept changes because the assertion had no inline string literal to patch.

## What Didn't Work
Overwriting expected text files when INSTA_UPDATE=always bypassed insta's pending-snapshot and acceptance workflow.

## Solution
The runner uses supported binary snapshots with text payloads and metadata. Required report.txt and counterexample paths are symlinks to their .snap.txt payloads, so readers and cargo insta review use one source of expected content.

## Prevention
A deliberate payload mismatch generated a pending binary snapshot. cargo insta accept accepted that snapshot, updated the required text alias, and restored a passing test. Alias, metadata and payload omissions each fail explicitly, as does orphan metadata.
