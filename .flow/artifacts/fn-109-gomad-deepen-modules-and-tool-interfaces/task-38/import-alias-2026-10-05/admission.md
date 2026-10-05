# Capability-policy import alias correction

Task38 extends its prior compatibility alias correction to policy.go and policy_test.go in target/internal/capabilitypolicy. Only their two import declarations may change. The existing package identifier, selectors and all other source bytes remain unchanged. Existing controls, pins and historical receipts are protected.

BASE: 3b15e3cab90c8d72ac5b7d298ebdb80537906145. Actual unfiltered target plus leaf lint reports exactly two goimports findings. BASE policy tests pass: four top-level, 16 including subtests; target digest/golden/projection controls pass: ten top-level, 23 including subtests; neither command has failures or skips.

The actual root integrated gate loads 55 ordinary host packages and reports 327 findings with the original comparison revision 951c5516e9e7b3066e7e069adda9565cfd68844c and FIX=false. Make exits 2 after golangci-lint exits 1; its errortype stage is not reached. Final integrated counts must be measured, not inferred.

The configured Codex plan review resumed its existing session using an explicit gpt-6.1-sol/high override and returned SHIP. The only prior FYI concerns overlapping source owners; serial admission remains mandatory. The duplicate cleanup advisory is already retained in the parent Decision Context. The receipt attests the backend model; host subagent model execution metadata is not exposed.

Original five acceptance bullets remain in force for their historical correction. The additive sixth criterion requires matched controls, architecture/purity/edge checks, scoped lint, errortype, formatting, check-only validation, fresh source bindings, the real integrated gate and independent source-progress review. No task, dependency or original completion gate is removed. Native Linux remains transferred to fn128 and nonblocking. Original source-owned full/native Darwin/formal requirements remain open.

The complete baseline integrated output is losslessly JSON-string encoded in baseline-integrated-gate.json to preserve diagnostic whitespace. Decode with jq -jr '.' when comparing bytes. Baseline scoped logs and source bindings are separate files. Root owns Flow state and commits; one source/cache writer runs at a time.

stage: plan-review - ran (Codex backend SHIP)
stage: plan-sync - skipped(config: disabled; no task completed)
Tracker sync: n/a (bridge inactive)
