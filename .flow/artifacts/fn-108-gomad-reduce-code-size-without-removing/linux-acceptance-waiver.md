# fn-108 Linux acceptance waiver

## Decision

On 2026-10-04 the owner requested: "close fn-108 without linux check".

The outstanding linux/amd64 execution requirement in fn-108 R9 and task 8 is
waived for this spec only. Close the spec under this amended acceptance scope.

## Evidence and limits

- Tasks 1–7 retain their accepted darwin/arm64 evidence in [final.md](final.md).
- No Linux check was performed or passed as part of this closure; Linux remains
  unverified for the fn-108 candidate.
- The original final report, commands, baseline dispositions, and source-bound
  results are unchanged. Its incomplete R9 statement describes the pre-waiver scope.
- D12 and all other specs' native-platform requirements remain unchanged.
- This waiver makes no new platform-support, equivalence, or replay guarantee.
