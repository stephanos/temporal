# Original integrated lint gate: exact-pack switch progress

Root executed the original gate on admission HEAD `533aa074337bc1e21e52c306ddb5a19e93fdfdbe` plus the frozen candidate after the worker returned with all tool handles terminal and reaped. This actual execution supersedes only the worker handover/evidence's pending-root-gate status; those files retain their earlier snapshot.

```sh
make --trace lint-code-gomad3 GOLANGCI_LINT_BASE_REV=951c5516e9e7b3066e7e069adda9565cfd68844c GOLANGCI_LINT_FIX=false GOLANGCI_LINT=/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 ERRORTYPE=/tmp/fn109-lint-tools.ZdNe1t50/errortype
```

Started `2026-10-05T14:37:53Z`, ended `2026-10-05T14:38:24Z`, elapsed `31.524960015` seconds. It listed the same 55 ordinary host packages with `disable_grpc_modules,,test_dep,`, frozen base and unchanged repository config. It exited2 at the top-level Make: golangci exit1, recursive Make exit2, module router error, final Make exit2. The full integrated errortype stage was not reached. Standalone compatibilitypack errortype separately passed; it does not make the unreached full stage green.

[Raw final output](root-integrated-final.log), [command](root-integrated-command.sh), [comparison](root-integrated-comparison.json) and [comparison implementation](root-compare-integrated.mjs) retain the execution. The final raw SHA-256 is `51d765a947eb3a52300ed85195a1d04e09c7009bdf5a7f829e8fc0d7aee3064f`. Its historical BASE is task41's immutable `root-integrated-final.log`, SHA `1bb1271a13e1ab87cf3ec0fadedd8ffb22e8064046c5ec5cf590678cdfac7544`.

| Analyzer | BASE | Final |
| --- | --- | --- |
| errcheck | 252 | 252 |
| exhaustive | 4 | 3 |
| forbidigo | 11 | 11 |
| gci | 1 | 1 |
| staticcheck | 56 | 56 |
| Total | 324 | 323 |

Root independently compared diagnostic headers and complete raw blocks. Only the `compatibilitypack/policy.go:196:4` exhaustive finding disappeared; no finding was added. All323 remaining blocks are byte-identical. The original input pins, all995 protected source/tool entries and both frozen candidates were checked before and after the gate. [Protection](root-integrated-protected.sha256), [candidates](root-integrated-candidate.sha256) and the corresponding check logs retain those results. The nine original gate inputs remain task41's `root-integrated-inputs.sha256`.

Root also independently verified all nine latest `-nil-selection` worker captures: actual raw test counts, log/manifest hashes,997 stable entries, exact current source hashes, revised literal BASE/final outputs,995 protected inputs, exact production reconstruction, unchanged scoped residual blocks and decoded Git diff bytes. [Verification command](root-verify-worker.sh) and [actual verification output](root-worker-verification.log) retain this check. The passing final test file is SHA `e3e498aa5750ec118480f68416170fbf002e2d5fa4cb879d59599f9e5a0266d5`; earlier first-final files/receipts are historical and superseded, not final-candidate proof.

Required lint remains red and formal implementation review is deferred, not SHIP. The scoped package's seven findings and integrated323 residuals remain with their original corrective/qualification owners. Source progress does not satisfy missing first-baseline, predecessor, preservation, full/default/functional/affected-consumer, formal, native Darwin or static both-source-set gates. Linux execution stays deferred/nonblocking under fn128. The independently traced external-pack policy mismatch remains separately scoped, with no plugin/cgo execution claim.
