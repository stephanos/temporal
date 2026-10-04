# Architecture fitness source investigation

Read-only scout requested as Codex thinking scout `gpt-6.1-sol` at high, fresh context. No production changes, tests, builds or qualification resulted from this investigation. Task 19 remains unstarted and follows tasks 16–18. This artifact preserves the scout's findings and the conductor's source-scope clarification; it is not an acceptance receipt.

## Checker design

Actual stored/generic callback bindings and concrete JSON receiver/method-set boundaries are retained in [callback-source-scout.md](callback-source-scout.md). Consult that source-grounded evidence when implementing transitive callback analysis; this design's names alone are not purity proofs.

Exact source-tree and nested-module discovery boundaries, optional generated roots and stale-exclusion controls are retained in [inventory-source-scout.md](inventory-source-scout.md). Consult that inventory rather than broadly exempting hidden directories, all testdata or all nested modules.

Confirmed public-signature defects and their selected corrective scope are retained in [public-signature-source-scout.md](public-signature-source-scout.md). Consult it when implementing the export walker and repair the exposed internal identities; builtin-only defined public types and ordinary private storage remain positive controls.

Use Go-owned typed analysis (`go/parser`, `go/ast`, `go/types`, `go/build` and go-list metadata), with no new dependency. The module already uses x/mod, not x/tools.

- Discover host production packages with `go list -e -json -tags test_dep ./...` separately for darwin/arm64 and linux/amd64, controlled GOWORK and readonly module mode. Reject included package-listing errors. Inventory source directories too: ./... omits hidden directories, testdata and nested modules. Exclusions name a path category and reason, match actual mandatory fixture/overlay modules, and have synthetic matcher controls. Optional build trees need not exist in a clean checkout. The root's test-only harness is an explicit owner and rejects new unowned production files.
- Retain current owner and module edges. The current architecture_test.go's selected roots and running-host file set miss undiscovered owners and alternate-platform edges; its import/name allowlist and export-name walker do not establish transitive purity or signature visibility.
- Pure roots include World core/mailbox (not effectful process/descriptor transport), record, campaign controller rather than its effectful journal siblings, exploration/choice frontiers, capability policy/evaluation, and task 16's final lifecycle event owner rather than the whole execution package. Include all production functions, methods, initializers and closures in named roots. Follow typed object references through local/dependency helpers, function values, method calls, dot imports and generic instances; detect go statements.
- Report platform, root, call chain, source position and effect. Permit local allocation/mutation, hashing, locks, model time and bounded waiter notifications. Reject host filesystem/environment/network/process/clock/timer/entropy effects and new goroutines. Do not exempt whole stdlib packages: fmt.Errorf differs from fmt.Print; bytes.Buffer.Write differs from os.File.Write. fmt/JSON may dispatch user Formatter/String/Error/MarshalJSON callbacks. Model exact symbols, receivers and callback-sensitive arguments with source-version guards and positive/negative controls. Resolve repository callback bindings such as SeedController.Next and exploration clone/size operations; unexplained effect-bearing bindings fail closed.
- Walk complete public types with cycle guards: parameters/results, methods, exported fields, aliases and RHS, arrays/maps/channels/pointers, generic arguments and constraints, interfaces/unions, embeddings and promoted public members. Keep encapsulated private fields private unless exposure through embedding creates public surface. Resolve every internal segment using exact parent-directory boundaries and intended consumer import paths. A module-root internal type and runner/internal type have different visibility; a substring test is insufficient. Retain the existing external Runner compile-positive.
- Negative fixtures run the actual production checker and assert specific failures: ownerless roots, forbidden owner/module edges, direct and indirect host effects (including dependency and stdlib helpers), goroutine starts, callback-mediated effects, inaccessible function/method/field/interface/alias/generic/embedded signatures, stale exclusions and alternate-platform-only failures. Positive controls include harmless formatting/JSON to an in-memory buffer, resolved pure callbacks, local mutations, unrelated effectful siblings, legal internal visibility and valid platform-specific files. Use small local fixture modules/replacements, no downloads.

## Confirmed record timezone effect

The scout directly inspected the nested module's selected Go 1.27.1 GOROOT: `/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64`. The Unix source applies to both qualified source sets. This is source-reachability evidence, not a native execution result.

`record/validation.go:27`, `:146` and `:149` parse RFC3339Nano strings and discard the returned Time. For accepted numeric offsets (including +00:00, -00:00 and +01:00), the reachable stdlib chain is:

1. time/format.go:1026 passes Local to both fast RFC3339 and fallback parsing.
2. time/format_rfc3339.go:146 performs local.lookup for a numeric offset; the Z branch skips this lookup.
3. time/zoneinfo.go:149 reaches Location.get at :91 and lazy initLocal.
4. time/zoneinfo_unix.go:28 reads TZ through syscall.Getenv. Unset TZ tries /etc/localtime; named/absolute zones can read other files. TZ=UTC still reads the environment.
5. time/zoneinfo_read.go:531 and :575 load through helpers reaching syscall.Open/Read in time/sys_unix.go:27. Successful zone loading calls runtimeNow for the cache timestamp at zoneinfo_read.go:323.

Replace these three validation calls in task 19 with `time.ParseInLocation(time.RFC3339Nano, value, time.UTC)`. At format.go:1041, this uses the same fast and fallback grammar with UTC rather than Local. Both numeric paths construct the same instant before selecting location metadata; UTC lookup does not initialize Local and FixedZone uses in-memory state. Parsing syntax/range errors remain unchanged. Original timestamp strings remain untouched because validation discards Time.

Preservation tests must compare old/proposed acceptance, Time.Equal on valid instants, complete ParseError fields/text, and all three field/error-precedence paths. Include Z, both zero-offset spellings, positive/negative offsets, fractions, permissive fallback comma fractions/single-digit hours/offset hour 24 or minute 60, invalid dates/offsets and trailing input. Verify finalized manifests, canonical bytes, record hashes, failure signatures and decode round trips for identical inputs. Checker controls reject time.Parse and accept explicit UTC ParseInLocation with an argument-sensitive rule, not a package/function-wide exemption.

## Scope reconciliation

The conductor extended task 19's Files/Touches and Approach through flowctl to include record validation and its preservation tests. This repairs a confirmed effect in an already-required pure root instead of teaching the new checker to ignore it. R8 acceptance criteria, single D4 ownership, delivery order, native gates and no-commit constraint are unchanged. The writing-for-agents skill influenced the clarification by placing the evidence pointer beside the precise implementation and completion obligations. No record production source was edited during task 16's ownership.
