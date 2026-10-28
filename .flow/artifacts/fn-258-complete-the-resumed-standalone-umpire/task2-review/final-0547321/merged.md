## Issue 1
- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: .flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/standalone/adaptation-allowlist.json:749
- **R-IDs**: [R3]
- **Title**: Name the portability category in the required-test setup adaptation
- **Problem**: The gate.rs allowlist reason describes supervised fetch and setup failure but does not explicitly name the portability category enumerated by task acceptance.
- **Suggestion**: Prefix the reason with the framework-only cold-checkout portability rationale. Metadata-only, no source or approved blob change. Source: contracts finding1.

## Issue 2
- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: .flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/standalone/source-adaptation.patch:500
- **R-IDs**: [R3]
- **Title**: Make the required-test provisioning side effect explicit in the CLI guide
- **Problem**: dev/xtask/README.md says validate forces required TLC but does not explicitly describe tests-stage fetch-tlc.sh network/build writes or exit propagation. Root README already promises provisioning checksum-pinned tools as needed; this is a non-blocking narrower documentation omission.
- **Suggestion**: Add that sentence during task5 final documentation synchronization. Source: integration finding1; reviewer marked R3 met and verdict SHIP.

Declined correctness finding1/contracts finding2: the prelude intentionally belongs to the tests stage receipt; script stderr identifies setup errors and exit status is preserved. Duplicate timing labels follow the existing execute pattern and break no consumer. Record the label in the adaptation rationale instead of changing source after full proof.
Declined correctness finding2: direct update/test-run provisioning is unchanged and outside this gate correction; required direct-stage and scheduled tests are covered. The controls preserve the existing contract.

This is a format-repaired merge of the actual three SHIP draws. Correctness and integration used inline enum fields rejected by the parser; raw reviews remain unchanged. All three reviewers mark task2 R2/R3 met; task4/5 remain deferred.

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>
