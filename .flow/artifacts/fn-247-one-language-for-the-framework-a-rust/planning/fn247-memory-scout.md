## Memory findings

| Track | Category | Entry | Why relevant |
|---|---|---|---|
| bug | security | `retain-backend-capture-handles-across-2026-10-23` | Directly covers backend process ownership, securely retained capture handles, bounded output, cleanup, descendants retaining descriptors, cancellation, and explicit owned-PID disappearance controls. |
| knowledge | decisions | `host-verified-model-provenance-without-2026-09-03` | Relevant to guest/hash provenance boundaries: model/compiler bindings and a deterministic checksum are verified against trusted host configuration; this is not a protocol signature or caller-supplied trust claim. |
| bug | integration | `correlate-pending-proposals-with-2026-10-23` | Confirms a precedent for reproducing temporal behavior against the committed Wasm guest and keeping operation correlation tied to exact typed request evidence. Not a build reproducibility recipe. |
| bug | build-errors | `track-schema-inputs-before-reusing-2026-09-07` | Adjacent lesson for artifact freshness: traced exact inputs must invalidate generated modules; schema mutation and warm-build controls provide evidence. This is Lake/Lean-specific, not a guest hash policy. |
| bug | build-errors | `isolate-scala-cli-author-builds-after-2026-10-16` | Adjacent freshness lesson: separate-library builds need isolated compiler workspaces after macro dependency changes; a successful package or rewritten jar alone does not prove fresh output. |

### Relevance

- The backend capture entry is the strongest match for process supervision. Preserve ownership of process/capture resources through cleanup, and ensure descendant-held descriptors cannot make joins outlive the deadline.
- Memory gives partial precedent for guest provenance and fresh artifacts, but no direct rule about hashing a committed guest or deterministic guest build outputs. Keep these as explicit plan requirements if fn247 needs them.
- R18 dependency-boundary search did not surface a directly relevant R18 entry. The tracked-schema and isolated-workspace entries are only analogous dependency/freshness lessons, not evidence for R18's specific boundary.
- The declined concept `.flow/memory/declined/generated-api-drift-verification.md` covers broad generated Lean API drift/CI verification. It appears unrelated to targeted process capture, guest hashes/build reproducibility, and R18 dependency boundaries; retain the decline and avoid turning this work into broad API drift verification.

### Query details

- Read cached `.flow/tmp/fn247-plan-memory-query.json`: original query `Rust process supervisor Python helpers guest lock R18`; rerank was `bm25`, with `rerank_reason: no_key` and `stage_line: memory: bm25 (jev-unavailable(no_key))`.
- Ran three targeted `flowctl memory search` queries without `--rerank`; default active-only status was retained. Queries were: (1) `backend process capture handles process group descendants cleanup cancellation pid ownership`; (2) `guest wasm committed hashes reproducible build artifact provenance`; (3) `R18 dependency boundaries generated API drift verification`.
- The guest/hash query returned host-verified provenance and committed-Wasm behavior evidence, but no direct reproducible guest-build/hash entry. The R18 query returned no R18-specific boundary memory. No judge retry was made.
