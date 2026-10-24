# Legacy compatibility retired

Owner decision, 2026-10-24: backwards compatibility is not required for open Gomad work.
This policy supersedes conflicting legacy-preservation requirements in open specs, tasks,
batch manifests and proof inventories. Completed receipts remain historical evidence.

## Current acceptance

Implement one documented current Rust contract. Its producers, consumers, schemas, CLI,
identities and fixtures may change together. Matching legacy Go/Python serialization bytes,
JSON shapes, diagnostic text, argument ordering, CLI spelling or old artifact formats is
not an acceptance gate. Earlier schemas and lineages may be refused explicitly; migration
and compatibility readers are unnecessary.

Frozen retired-host executables and preparations are not prerequisites when their only
consumer was a compatibility comparison. Replace public-verb acceptance with current Rust
end-to-end behavior. Retain current release/diagnostic noninterference, independently
defined model oracles and matched performance experiments. Preserve shared evidence until
its remaining consumers are identified; this amendment itself deletes no files.

## Retained guarantees

Same-candidate, same-schema deterministic recording and replay remain byte-exact where
the current contract requires it. Authenticate the actual current source, inputs, commands,
outputs and raw bytes; keep stable reads, corruption refusal and independent receiving.
Schema changes may deliberately regenerate identities and goldens. A new lineage or schema
must distinguish unsupported artifacts; old and new bytes are not interchangeable.

Keep supported functional behavior, workload/test selections, bounds, resource limits,
isolation, process cleanup, failure classifications and truthful qualification. Current
command records must describe what actually ran. Transparent wrapper passthrough and current
Linux/Darwin behavior are separate requirements. The full-cycle attempt allowance, admission
floors and delivery order remain unchanged.

## Proof retirement

Keep original check/function/endpoint IDs and source hashes as discovery history. A row
whose sole guarantee is waived legacy compatibility may be marked `retired_compatibility`
with its exact ID and this decision as the reason. It needs no replacement implementation,
catching test or comparison run. Retirement is neither an executed pass nor qualification.

Mixed rows retain their functional, integrity and bound terms. Every retained guarantee
still needs a directly catching test or mutation and a valid control at its actual boundary.
Cutover requires zero unresolved retained guarantees and explicit dispositions for retired
rows. Historical inventory totals do not impose a retained implementation count. Old argv
tables and frozen-host goldens are historical inputs, not current acceptance authorities.
