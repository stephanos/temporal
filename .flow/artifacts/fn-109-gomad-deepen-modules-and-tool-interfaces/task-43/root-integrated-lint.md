# Root integrated gate and preservation check

Root executed the original integrated gate after the worker returned with all
Go/cache handles terminal and reaped. The frozen candidate has admission HEAD
6cd5df49bdaafc43a0585bc3a251caa7081817ac plus the six reviewed Go files. This
actual result supersedes only the worker handover's pending-root-gate status.

```sh
make --trace lint-code-gomad3 GOLANGCI_LINT_BASE_REV=951c5516e9e7b3066e7e069adda9565cfd68844c GOLANGCI_LINT_FIX=false GOLANGCI_LINT=/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 ERRORTYPE=/tmp/fn109-lint-tools.ZdNe1t50/errortype
```

The command ran 2026-10-05T15:26:38Z through 2026-10-05T15:27:53Z, 75 elapsed
seconds at integer-second resolution. The root wrapper exited0 after verifying
the expected recorded Make exit2 and stable source/tool hashes. The actual gate
still failed. It listed 55 host packages with disable_grpc_modules,,test_dep,
and the unchanged config/base/fix=false settings. Golangci exited1, recursive
Make exited2, and the module router/top-level Make failed. The later full
integrated errortype stage was unreached; the worker's standalone affected-package
errortype pass is separate evidence.

[Raw output](root-integrated.log), [metadata](root-integrated.json),
[command](root-integrated.sh) and [comparison](root-integrated-comparison.json)
retain the execution. Root compared task42's immutable actual323 baseline with
the current323 diagnostics. No finding was added or removed. All319 unaffected
blocks are byte-identical. Four schema diagnostics moved by one line because
the owning comment gained one line; their headers normalize only that exact
shift and all other block bytes remain identical. The comparison also checks
the complete diagnostic-header multiset, so a duplicate cannot conceal an
omission.

| Analyzer | Task42 baseline | Current |
| --- | --- | --- |
| errcheck | 252 | 252 |
| exhaustive | 3 | 3 |
| forbidigo | 11 | 11 |
| gci | 1 | 1 |
| staticcheck | 56 | 56 |
| Total | 323 | 323 |

Root's before/after manifests match all994 selected inputs, comprising993 tracked
source/configuration files and the new untracked test. Go/golangci/errortype
executable hashes also match after the gate. These are named selected-input
checks, not a full repository/toolchain closure qualification.

Root separately ran [the worker verifier](root-verify-worker.mjs), whose
[actual output](root-worker-verification.json) checks1219 tracked scoped source
files against current bytes, four executable hashes, all six candidates,
1214 protected files,51 generated/pin files and13 command captures. It recomputes
raw JSON test event counts and skips, checks statuses/times and verifies that
the five RED test files match the passing candidate bytes. The independent
source review supplies exact production and existing-test reconstruction.

R21's admission behavior is corrected at this source checkpoint. Original
matched-first-baseline/predecessor/preservation/full/default/functional,
affected-consumer/formal/native-Darwin/static-both-source-set acceptance remains
open wherever unproved under task11/21. Native Linux remains nonblocking and
unverified under fn128. Required lint is red; formal implementation review is
deferred. No plugin/cgo execution, host escape or native qualification is claimed.
