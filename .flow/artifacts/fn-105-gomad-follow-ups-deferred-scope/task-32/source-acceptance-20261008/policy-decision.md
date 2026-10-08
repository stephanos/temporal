# Retained D27 patch-policy decision

Patch-policy decision: declined the proposed `runtime/proc.go` overwrite of `LastGC` and `PauseEnd`. Although that hook would edit an already-allowed file rather than a prohibited collector file, scheduler-owned code would mutate collector-owned state and cross the collector prohibition in substance while merely masking the reported values and leaving the underlying host read. The fields therefore remain an explicit contract limitation. No collector, profiling, or assembly source was modified.

[Original worker summary](original-summary.utf8.json) and [evidence](original-evidence.utf8.json) retain the original bytes as lossless UTF-8 JSON strings. Decode each `value` as UTF-8; source-binding.json verifies the original byte counts and SHA256 values. This records the existing decision; it is neither new approval nor a fresh review receipt.
