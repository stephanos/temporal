# Public installation corpus

`installation-lane.py` runs fresh Go and Rust public commands twice. It checks
install profile/default/resource semantics, compiler pins, manifest publication,
linked-helper admission, discovery precedence, and the actual WASM refusals.
The unselected `doctor --json` report is compared separately from the selected
WASM dispatcher's unsupported `doctor` refusal. `analyze` remains unsupported.

Every command retains stdout, stderr, numeric exit and SHA-256 receipts under a
new `--work` directory. Discovery authenticates the returned manifest digest and
manifest bytes. The Rust golden preserves every implementation/compiler/runtime
and configuration digest, original configuration, namespace and bounds. Only
explicit manifest/cache/compiler/runtime path operands are mapped to named
roots; no digest, identity, status or outcome is normalized. Unknown manifest
or admission fields fail the lane.

The numeric child lane checks twelve public cases against fixed fuel and memory
values: decimal, hexadecimal, binary, octal and underscore spellings preserve
the same bounds; signs, invalid octal, malformed underscores and overflow refuse.
Both lineages retain the literal argument vectors and parsed values. A duplicated
member inside the admitted guest configuration must refuse before discovery.

The configuration table checks Go field aliases, zero defaults, null values and
fixed entropy-array length against both public installers. Accepted original
configuration bytes and their hashes remain literal. Temporal descriptor aliases
use authenticated descriptor bytes with recomputed length and SHA-256; duplicate
aliases and inserted delimiters must refuse.

Discovery retains the frozen Go `gomad-choice-trace/v3` and Rust
`gomad-choice-trace/v4` capability names literally. Each lineage's expected
format is checked separately; the remaining capabilities compare directly,
with Rust's authorized public `plan` receiver included in its golden.

The Temporal installation case captures the same checkout's SQLite schema with
both public binaries, compares the complete guest configuration, build tags and
default limits, and independently checks descriptor and payload hashes. Changed
payloads, changed descriptor bytes and schema symlinks must refuse publication.

Seed preparation, scoped choice preparation and recovery use the adjacent
`plan-check.py` and `recover-check.py` public-command owners. The Go reference
runs real exploration and recovery on its own authenticated campaign; the Rust
receiver retains canonical prepared records and receipts with zero seed-plan
executions, then repairs torn seed and round journals idempotently.

The detached-input case installs using private compiler/runtime copies, changes
the compiler, then removes both copies without changing the manifest. Discovery
must return exactly the same authenticated description each time. The running
binary owns helper execution: omitted `--helper` uses its linked implementation,
identical executable bytes are admitted explicitly, and the legacy Go helper is
refused without being selected for execution.

Run from the repository root with authenticated frozen inputs:

```sh
GOMAD_GOLDEN_REGENERATE=1 tools/gomad/scripts/behavioral-parity \
  --corpus tools/gomad/testdata/parity --verb install \
  --installation-binary /absolute/current/gomad \
  --installation-go-binary /absolute/frozen/gomad \
  --installation-go-helper /absolute/frozen/gomad-wasmhost \
  --installation-go-compiler /absolute/pinned/go \
  --installation-go-source /absolute/frozen/source \
  --installation-preparation-root /absolute/stable/preparation \
  --output-root /absolute/new/installation-run
```

Regeneration is opt-in. Both independently executed views must agree, and the
second regeneration must reproduce the first file byte-for-byte. Run again
without the flag in a different fresh output root to check the reviewed corpus.
`--grammar-only` on the child script probes grammar and default doctor without
invoking installation preparation. Missing cases and missing goldens fail.
