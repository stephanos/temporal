# Task 32 allocation and conversion diagnosis

Six constructor families pass BASE and fail the frozen candidate for both dirty and clean callbacks. They are `new([1]func())`, `new(func())`, `new(map[int]func())`, `new([]func())`, an empty map literal, and `&[1]func(){}`. Dirty execution calls the callback once and clean execution calls it zero times. BASE retains `record.Check -> canonicaljson.Dirty -> time.Now` and has no clean findings on both supported Load source sets. The candidate produces `unresolved callback` for each counterpart. This preserves fail-closed rejection but loses concrete provenance and rejects proven clean code.

`new(helper.Box)` is green on both versions. The proposed struct regression is not proven by its nearest reproduction. Its source still has a nil-field sharing hazard, but global field-assignment resolution supplies `F` when the original receiver lacks a field entry. This fallback explains why the simple test succeeds without proving shared storage was retained.

## Exact proof

Root is `/Users/stephan/Workspace/skunkworks/gomad/temporal`, branch gomad, BASE and HEAD `f931c9879e3017b346562667b9f6fbcc4db458ec`. Candidate effects.go SHA256 is `8a077fd8a0faea457d43930bc6defc98b8f0d524333ac1a8015a29a97934cdc2`; standard.go is `efad9411c3876b452d4d2f4905a4ba1cb92b43a0428ea2f039c94b3e5d9fd88d`; new test working source is `4663ca0c0035d8f8a5d9097b52e47d5a5638309a03e49ebe9d1f874f77f6a0b6`.

`task32-allocation-alias-scout-probe.py candidate` and `baseline` retain all three substituted source files and exact overlay manifests under `task32-allocation-alias-scout-proof/{candidate,baseline}`. Baseline effects.go and standard.go were independently fetched with `git show BASE:path` and compared byte-for-byte with the previously saved baseline sources. Both versions use the same appended fixture code and causal helper in the new test. The existing test file itself was not changed.

Each command has this argv, with its archived overlay path and exact test expression recorded in its JSON receipt:

```text
/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go test -count=1 -tags test_dep -v -overlay <bound overlay.json> ./internal/gomadtool/architecture -run ^TestAllocationAliasScout$
```

Cwd is the nested tools/gomad3 module. Environment sets GOWORK=off, GOTOOLCHAIN=local, GOPROXY=off, GOFLAGS empty, and the stock toolchain bin first on PATH. GOMADSEED and GOMAD3_CHILD_SEED are absent. Every fixture first invokes literal stock `go test -count=1 -tags test_dep -v ./record`, then runs Load for linux/amd64 and darwin/arm64. Package edges are empty throughout. Each main receipt contains 24 runtime PASS observations, twelve count1 and twelve count0 observations, and 24 Load observations per platform. Overall baseline exits1 because three inherited fixture assertions fail; candidate exits1 with those three plus twelve introduced failing assertions.

| Fixture family | BASE dirty / clean | Candidate dirty / clean | Classification |
| --- | --- | --- | --- |
| new struct | pass / pass | pass / pass | Nearest hypothesis unproven |
| new array | pass / pass | fail / fail | Introduced |
| new function slot | pass / pass | fail / fail | Introduced |
| new map slot, replaced through `*p` | pass / pass | fail / fail | Introduced |
| new slice slot, replaced through `*p` | pass / pass | fail / fail | Introduced |
| empty map literal | pass / pass | fail / fail | Introduced |
| address of zero array literal | pass / pass | fail / fail | Introduced |
| address of zero array variable | fail / fail | fail / fail | Inherited |
| slice literal containing nil | pass / pass | pass / pass | Preserved |
| address of empty struct literal | pass / pass | pass / pass | Preserved |
| address of zero struct variable | pass / pass | pass / pass | Preserved |
| address of zero function variable | pass / fail | pass / fail | Inherited clean unresolved |

`task32-allocation-alias-scout-empty-probe.py` separately binds five constructor-content controls under `task32-allocation-alias-scout-empty-proof`. Its argv differs only in the overlay and `^TestAllocationAliasScoutEmpty$` expression. Both stages exit1. Stock runtime counts are zero for make map, empty map literal, make zero-length slice, empty slice literal, and zero-length array literal with concrete Leaf.String methods. Both versions nevertheless report Leaf.String -> time.Now on both Load sets. These false positives are inherited. Candidate make-map's typed element cell is a phantom map value; replacing it with an empty storage cell would avoid that tested element callback, but map-key callbacks still use the declared-type fallback in standard.go. The probes establish no general empty-map formatting fix.

`python3 .flow/tmp/task32-allocation-alias-scout-audit.py` exits0. It independently validates all four log digests, tool and script digests, archived source/overlay bindings, actual runtime counts, fixture status counts, both metadata counts, exact Git baseline bytes, env, argv, cwd, and current source stability. The empty harness derives its runner from the main harness; the audit additionally pins both harness digests. No product source was edited. Sessions60578,58469,85924,90437 and the audit command are terminal. No delegate was spawned. The inherited routing result remains Tier session, jev-unavailable(no_key); no judge or bridge was invoked.

## Cause and pertinent allocation inventory

effects.go:750 copies the abstractValue header so concrete destination types and methods can differ from source types. Existing fields maps and element nodes can stay shared, but nil storage cannot. `assign` allocates a missing fields map or replaces nil elements on the receiving header. For dereference assignment without `$pointee`, it overwrites that receiving header. BASE returned the original source header from conversion, so those operations affected the caller. The candidate returns a distinct header, so the first mutation can be attached to the converted view alone.

`new(T)` at effects.go:782 returns only valueOf(pointer). It has neither a pointee slot nor an element cell. It is the producer for all four proven new regressions. `new(struct)` has the same fields hazard but the global field-binding fallback masks the nearest case.

`make` establishes element nodes for slices and maps in this candidate. Those close the already reviewed make-origin first-insertion cases. A positive-length slice has real zero-valued elements; a fresh map has no entries, so its declared element type is not an existing value. Channels have no modeled mutable callback storage here and no additional channel-conversion claim was tested.

CompositeLit initializes a fields map for structs and aggregates actual explicitly present array/slice/map values into elements. An empty map has nil elements. A zero-initialized array literal can have positive length and nil abstract elements. A slice literal with an explicit nil entry has a nonnil node, which explains its passing conversion pair. Struct literals already own a fields map, explaining passing empty-struct pointers.

Unary address-of creates a pointer header with `$pointee` referring to the source value, but snapshots its elements pointer. That existing mechanism preserves dereference slots. It does not establish a shared indexed cell when the source array's elements are nil. This causes the introduced address-of-array-literal regression. Zero variable initialization through ValueSpec/valueOf is an analogous source-only producer; zero-array address indexing was already unsupported in BASE and remains inherited.

Other valueOf paths represent unresolved values, parameters, fields, tuples or generic concrete types. Nil elements there can mean unknown content, so they cannot globally be reinterpreted as known empty. Parentheses, slices and identifiers forward existing values. append/copy and the slices summaries consume existing storage nodes; they are mutators or summaries in this path, not additional proven constructor regressions. Header copying in assertions/type switches and the wider ordinary value-copy approximation are outside this bounded investigation.

## Smallest complete repair recommendation

Keep the typed concrete destination header. Use the existing fields and elements graph to establish shared storage at the proven fresh allocation/address producers before a reference conversion copies the header. A new storage struct, graph-key change, memo change or fixed-point redesign is not causally required by these proofs.

For freshly allocated maps, establish a shared bottom element node with no invented contents. For new pointer allocations, establish an existing `$pointee` cell so `*p` replacement mutates the shared slot. For positive-length fresh arrays used through pointers, establish a shared known-zero element node before creating/copying the pointer view. Initializing only a fields map and elements node cannot repair the pointer-to-function/map/slice replacement cases without a shared pointee slot.

Limit this initialization to fresh allocation or address storage. Ordinary scalar, struct and array value conversions must not gain newly shared mutable Go value storage as a consequence of the repair. Nested reference fields may remain shared according to Go semantics. Known fresh nil function/interface values may receive the corresponding known-zero flag; unknown imported/field/parameter content must retain fail-closed unknown flags. Do not clear unknown globally. Recursive zero types should stop at nil reference values instead of recursively allocating their referents.

Use existing nodes recognized by cloneValue, compactFields, bindingGraphKey and normalization. Keep source and destination headers distinct and do not assign artificial shared origins that normalize them back into one typed header. The exact proposed repair remains source-only reasoning, with no patched-overlay or implementation verification in this scout. Root should require the six introduced dirty/clean pairs, the existing conversion/make/nil/unknown controls, meaningful value-copy preservation controls, recursive and graph tests, ordinary package/consumer/validation/static gates, and a fresh frozen review before admitting it. Inherited forms and empty-content callbacks remain explicit limits unless root admits their constructor controls. Stock linux/arm64 and supported-platform Load are developmental evidence; both native patched-runtime acceptance gates remain open.
