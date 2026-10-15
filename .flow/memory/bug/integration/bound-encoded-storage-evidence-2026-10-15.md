---
title: Bound encoded storage evidence separately from recovery images
date: "2026-10-15"
track: bug
category: integration
module: tools/gomad/internal/backends/wasm/wasi/helper_model.go
tags: [wasm, storage, evidence, bounds]
problem_type: integration
symptoms: Valid large enumerations or escaped metadata fail helper evidence capacity
root_cause: Binary image limit did not account for JSON expansion and enumeration budgets
resolution_type: fix
related_to: [bug/integration/share-offered-output-accounting-across-2026-10-15]
---

## Problem
Go capped helper storage-state evidence at twice the recovery-image limit. Crash enumerations retain multiple filesystem states under their own byte budgets, and escaped paths expand snapshot and operation JSON independently of binary image size. Valid executions therefore failed as capacity before Go could decode their evidence or return the image.

## What Didn't Work
An image-derived cap alone rejected a 45,065,293-byte enumeration with a 135,937-byte image. With enumeration disabled, fifty empty files with 4000-byte control-character paths produced 3,634,997 bytes of JSON metadata and an 823,189-byte image, exceeding the previous 2,129,920-byte state cap.

## Solution
storageStateLimit accounts for the two snapshot trees and operation history, JSON escaping, each volume's enumeration byte budget, seen identities, cursor and framing. Saturating arithmetic preserves valid huge caller budgets without wrapping. The separate recovery-image cap remains unchanged. CrashEnumerationLimits also rejects unknown nested fields before guest execution.

## Prevention
Test real Go-owner and helper-owner execution with enumeration evidence larger than the base allowance, escaped metadata without enumeration, image recovery, huge budgets with tiny actual results, and exact-limit versus one-byte-over-limit stream admission. TestStorageEvidenceLargeEnumerationOwnerParity, TestStorageEvidenceEscapedMetadataOwnerParity and TestStorageEvidenceStreamAdmissionKeepsIndependentBounds cover these cases. Nested serde structs need their own deny_unknown_fields annotation.
