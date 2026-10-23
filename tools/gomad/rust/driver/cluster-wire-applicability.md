# Six original cluster wire roots

The unchanged `cluster-provider` roster includes six former Go transport tests.
They test a framed helper boundary that current native cluster execution does not
expose. Passing unrelated native tests cannot receive these roots.

`cluster_wire_reference.py` executes exactly these original tests from commit
`1d695061b6382009d0659238c652a5a765b1d5f5`:

| Root | Original assertions retained by actual Go execution |
| --- | --- |
| `TestClusterTransportCodecCorpus` | Exactly one named fixture; strict fields and trailing EOF; source pin, schema, name and target a/incarnation 2; exact minimal result, old-prefix input, tagged bytes and invalid attribution; actual tag/attribution equality. |
| `TestClusterTransportCodecCorpusIsCurrent` | Original generator observation and source pin; indented JSON plus final newline equals the fixture; regeneration disabled. |
| `TestClusterTransportCodecLiteralStageGuards` | Current tag reaches unsupported payload rather than attribution refusal; malformed JSON is not attribution; all six literal mutations refuse. |
| `TestClusterTransportStageCorpus` | Exactly two named fixtures; strict fields and EOF; filename/name identity; source pin and exact target; current tag reaches unsupported payload, malformed JSON fails before attribution; actual typed errors equal both fixtures. |
| `TestClusterTransportStageCorpusIsCurrent` | Both original generated observations and source pin; both exact indented JSON/newline fixtures; regeneration disabled. |
| `TestClusterTransportStageLiteralGuards` | Both valid cases pass literal guards; both refuse each wrong-stage, wrong-target and stale-input mutation. |

There must be exactly six root and twelve mutation run/pass pairs, with package
start/pass, no skipped/failed/extra tests, and no regenerated fixture writes.
These tests construct the original in-memory `ClusterSession`/`clusterStream` and
call `appendClusterTag`/`frame`; their selected bodies do not launch a helper or a
guest. No native helper binary is substituted into this historical proof.

The producer reuses `terminal_controls_reference`'s admitted original archive,
stock Go installation and invocation-only glue. The existing complete Git/archive
verifier binds every original source blob before execution; the complete tree is
reopened afterward. Exact command, process, compiler, raw event and fixture hashes
are retained under `broad-harness/cluster-wire-reference`. The five original
terminal Go controls are a different reference and provide no credit here.

`terminal_controls_reference.module_cache` must name a genuine sealed dependency
cache. `original_wire_cache.py` defines the six modules actually needed by the
original WASI test package, reference command and cluster tests: JSON experiment,
testify, go-spew, go-difflib, x/sys and yaml.v3, at the original `go.mod` versions.
Their exact versions remain in `MODULES`. An offline `go list -deps -test -json
-tags=test_dep` over all three selections succeeded using only that six-module
cache and fresh metadata scratch; it did not compile or execute tests or guests.

Each extracted module must exactly match its genuine cached zip. Both the archive
and `go.mod` h1 checksums must match original `go.sum`; the latter also authenticates
generated module metadata for older go-spew/go-difflib archives without `go.mod`.
Only those module sources and download metadata are admitted. The freezer seals
every cache file, and both six-cluster-wire and five-terminal-wire producers check
the complete cache before and after their commands. Their child environments set
`GOMODCACHE` explicitly, use fresh private `GOCACHE`, disable proxy/checksum/VCS
network fallback and require readonly module resolution. No compiled cache is
included. The earlier development-only seventeen-event proof is unchanged and
does not establish this new input closure or current qualification.

`core_cluster_wire.py` separately reopens that proof and validates actual current
native lifecycle rows from the same authenticated helper plus host-scenario broad
processes. It requires all typed handle/namespace/seed admission refusals, both
unsupported callback refusals with zero launched stores, and stale restart refusal
that preserves the old owner and recovery inputs before allowing incarnation 2.
Pinned current `Request`, `NoRemote`, broker and test sources bind the typed owner
boundary. Those predicates do not emulate JSON parsing, newline tagging, stale
wire prefixes, exact Go error text or fixture regeneration in Rust.

Each root keeps its own complete assertion list. Former raw framing assertions
are marked retired at that boundary, backed by the actual original Go checks.
`original_pass=false` applies to current native equivalence; the historical Go
execution has its own explicit scope. A broad `PASS` means this combined,
architecture-specific applicability contract was received. Missing historical or
native execution, mutated native guards, incomplete assertions and generic PASS
facts refuse. None of these six roots tests fuel, output counters or terminal
timing, so no terminal-control credit is attached to this contract.

The provider stage produces the reference after its actual broad processes, then
the independent reader reopens all evidence. The reference can also be prepared
and checked explicitly after the broad build root exists:

```
python3 tools/gomad/rust/driver/cluster_wire_reference.py produce --config CONFIG
python3 tools/gomad/rust/driver/cluster_wire_reference.py check --config CONFIG
```

These commands execute Go and require the qualification owner's resource lease.
Source checks and synthetic Python controls are not execution evidence. No stage,
root, obligation or final qualification label is added or removed.
