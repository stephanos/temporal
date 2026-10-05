# Canonical JSON switch scope amendment

The first admitted expressionless-switch recipe passed its literal characterization but failed the actual unfiltered package lint. On 2026-10-05 at 13:13:15 UTC the pinned golangci-lint command exited 1 with exactly one staticcheck QF1002 finding. The worker stopped with no live command; root amended the scope before resuming source writes.

The rejected production SHA-256 was `a515dfb5ecc15a6d5348ddf5ec6aa251d3451f2594c6cc6a4622c7accdb271d9`; characterization SHA-256 was `274b979156bd6f4ff47c6d5daa454dee51a670d65c3ba694ebed87e647bff747`. The raw lint output, retained as `rejected-expressionless-lint.log`, has SHA-256 `6024d4dbba92e061797fd49be0dc57f62c56f8db918b4d26079434c2b61681a6`. Its exact command was:

```bash
/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 run --config=../../.github/.golangci.yml --build-tags=test_dep --timeout=10m --fix=false ./internal/canonicaljson
```

It ran in `tools/gomad3` with the stock Go 1.27.1 Linux/arm64 PATH and GOENV=off, GOWORK=off, GOTOOLCHAIN=local, GOPROXY=off, GOSUMDB=off, empty GOFLAGS and GOMAXPROCS=2, with all three inherited Gomad seed variables unset. The worker receipt records before/after source stability. No native Darwin or qualified native Linux acceptance is implied.

Read-only research inspected the cached primary analyzer source `honnef.co/go/tools@v0.8.0-rc.1/quickfix/qf1002/qf1002.go` and exhaustive v0.12.0. QF1002 detects the expressionless equality chain and recommends a tagged switch. Reversing equality operands could avoid that pattern but offers no reader benefit and was rejected. Adding a default would hide future enum additions and was also rejected.

The revised recipe preserves the original tagged switch, its seven selectors and branch bodies, invalid-value guard, one Kind evaluation, visited keys, encoder ordering and trailing nil. One grouped terminal case explicitly lists the 20 omitted known reflect.Kind members and returns the nil result they already reach. This expresses their no-string-traversal behavior and leaves future enum members detectable. Root must receive actual plan-review SHIP before the worker changes source. The new source proof must reconstruct unchanged production BASE plus that single arm; earlier candidate checks remain historical and are rerun against the revised candidate.

BASE fixture development also found the pinned encoder's invalid UnsupportedValueError.Value and pointer-to-interface self-cycle stack overflow. Those failures do not authorize production semantic changes. The final fixtures retain literal typed error expectations and use a recursive struct-pointer cycle, without claiming all possible encoder cycles are safe. The existing visited-slice length omission remains explicitly characterized and unfixed.

The original admission receipt, BASE logs and review receipt are historical and unchanged. Original first-baseline, preservation, predecessor, full/default/functional/affected-consumer/formal/native Darwin obligations stay open wherever unproved; fn-128 still owns native Linux obligations.
