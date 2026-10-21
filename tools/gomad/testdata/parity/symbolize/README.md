# Pinned Go symbolization fixtures

`guest.wasm` is the actual retained module from a fresh Go 1.27.1 cooperative
`go-run` execution of `guest.go.txt`, copied to `main.go` in module
`gomad.fixture/symbolize`. `choices.bin` is that execution's unmodified trace.
The authenticated baseline inspector observed these select-poll sites, in order:
5383, 5384, 5384, 5385, 5385, 5386, 5387, 5386. All eight selects had two ready
alternatives. Exact replay succeeded through the same baseline installation.

| Physical function | Recorded site | Direct selectgo return PCs | Expected result |
| --- | ---: | --- | --- |
| `main.positive` | 5383 | 352780292 | `main.choose`, line 4; outer `main.positive`, line 23 |
| `main.conflicting` | 5384 | 352845828, 352845837 | Physical fallback: `choose` and `chooseOther` disagree |
| `main.sameChain` | 5385 | 352911364, 352911373 | `main.choose`, line 4; outer `main.sameChain`, no location |
| `main.known` | 5386 | 352976898 | `main.known`, line 40 |
| `main.unknown` | 5387 | 353042436 | Physical fallback: a real function-value `call_indirect` prevents completeness |

All known locations use `gomad.fixture/symbolize/main.go`. The call PC is the
concrete stored return PC minus one. `sameChain` has separate inline tree rows
for the same callee; its outer source locations are lines 34 and 35. This must
not erase the agreed inner frame/location or invent an outer line.

`expected.json` pins complete public symbols. `metadata.json` retains the actual
header/moduledata bytes, function records, PC streams, inline rows, decoded
chains, complete wrapper instruction bodies and all transfers. Its direct-call
patterns are matched in the actual retained code, not guessed from a nearest
constant. In particular, Go's SP is local 1; stack updates use `local.tee 1`
and `global.set 0`. Every syntactic direct `runtime.selectgo` call in each
wrapper is included. The indirect fixture is unmodified compiler output, not a
controlled mutation. Its pointer currently targets `known`, which explains the
last recorded 5386; a direct select in `unknown` still records 5387.

`compiler-inlining.txt` proves actual compiler inlining of `choose` and
`chooseOther` into their retained noinline wrappers. `compiler-assembly.txt`
supplies the corresponding emitted return-address relocations. Diagnostics use
the same source, pinned compiler, overlay and optimization flags as preparation,
adding only `-gcflags=-m=2` or `-gcflags=-S`. Their complete module bytes differ
because build flags/build IDs are embedded. Every fixture wrapper body matches
the retained guest byte-for-byte in both builds; every function body in the
`-S` build matches. These comparisons are explicit in `metadata.json`.
Only trailing spaces/tabs are removed from the tracked assembly transcript;
the original compiler stderr and its digest are retained in `proof.json`.

`guest-stripped.wasm` is a real rebuild of the same source and overlay with
`-ldflags=-s`. Inspection confirms its name section is absent while Go pclntab
magic remains in active data. Its `no_symbols` result relies on the required
name cross-check, not an assertion that stripping removes pclntab.
`no-metadata.wasm` separately contains only a valid eight-byte WASM header; it
is synthetic metadata-absence input and is never described as a stripped Go
guest.

`proof.json` records exact commands/environment, compiler executable/compile/
link digests, module/source/trace digests, baseline authentication, actual
record hash, raw inspector trace facts and numeric command exits. All recorded
commands exited zero. Fixture presence is mandatory; tests must fail if a module
is missing. This proves these fixture executions and metadata, not general Go
guest qualification or arbitrary indirect-call analysis.

The raw baseline artifact, provenance, overlay, diagnostic modules and producer
scripts are retained under
`/home/agent/wt/evidence/fn229-symbolize/fixture-proof/`. The generation order is
`produce.py`, `compile.py`, `metadata.py`, `publish.py`; each command driver is
launched with `env -u BASH_ENV -u SANDBOX_START_DIR /bin/bash`. The baseline
bundle and archived source paths are recorded in `proof.json`. Generation
requires those authenticated existing inputs and the full pinned compiler
distribution; it does not download tools or use a Go symbolization subprocess
in production.
