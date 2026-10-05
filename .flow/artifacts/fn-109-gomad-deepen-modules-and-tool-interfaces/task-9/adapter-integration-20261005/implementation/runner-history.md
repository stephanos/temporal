# Runner preimage recovery

`run-initial.cjs` retains the exact runner used by `public-bounds-red`, `adapter-focused-green` and `validate-after-integration`. Its observed SHA-256 is `a83b1f0b65a1b2134bf45f9301ede83545f7c52b4a0a0225a0dc925f92fbb36e`, matching each command's before and after input snapshots, six comparisons in total.

Recovery removed only the later `source-selection-final.go` input entry, the executable resolution and hash declarations, and the executable/tool hash fields added to the receipt. The recovered bytes matched the original recorded hash before creation and again after `apply_patch` wrote the new file.

The current `run.cjs` remains unchanged at `8223d7987c1a8ca98a2f37f3dd30bbc8f47ad58865ec78429c4b0bb2011e017e`. No runner or gate was executed during recovery. Raw receipts, streams and input snapshots remain unchanged. The initial runner's Go executable provenance remains in its before/after input maps; explicit top-level executable fields were introduced later. `run-initial.cjs` preserves the original filename references as historical bytes and should not be substituted for the current runner.

This recovery changes only the two new evidence files. The source/Go/cache lane remains released to the conductor.
