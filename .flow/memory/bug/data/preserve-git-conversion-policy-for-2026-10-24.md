---
title: Preserve Git conversion policy for content-addressed closure reports
date: "2026-10-24"
track: bug
category: data
module: tools/flow/validation-review.patch
tags: [git-attributes, artifact-integrity]
problem_type: data
symptoms: Committed report bytes differed from the recorded SHA-256 after Git conversion
root_cause: The close hashed working-tree bytes without protecting them through Git staging
resolution_type: fix
---

## Problem

The first fn-249 review found that a report hashed from working-tree bytes could change during its normal closure commit. Git's CRLF normalization and a clean filter both changed accepted JSON, leaving the committed blob inconsistent with its content-addressed filename and saved SHA-256.

## What Didn't Work

Hashing and atomically copying exact input bytes protected publication, while LF-only controls missed later Git conversion. Requiring a preliminary report commit would add a new workflow gate and still leave conversion policy unresolved.

## Solution

The installed close helper publishes two narrow `.gitattributes` rules that protect content-addressed report JSON and the companion attribute file from text, filter, encoding and ident conversion. It verifies effective attributes before lifecycle writes and returns both files in every successful new close manifest, including retry. A conflicting existing attribute file or higher-priority override refuses without rewriting user policy. `tools/flow/validation-review.patch` reproduces the installed fix.

## Prevention

`ValidationReview.test_git_conversion_preserves_exact_committed_bytes` checks CRLF input with `core.autocrlf=input`, `core.autocrlf=true` and a transforming clean filter, then compares committed bytes and SHA-256. The attribute conflict control verifies unchanged lifecycle bytes and preserved user policy.
