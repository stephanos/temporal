---
title: Host-verified model provenance without protocol signatures
date: "2026-09-03"
track: knowledge
category: decisions
module: Umpire PortableTestPlan provenance
tags: [umpire, grpc, provenance]
applies_when: Host-verified model provenance without protocol signatures
---

Fn-52 model-compiled plans carry exact model/compiler bindings and a deterministic checksum, but no digital signature, key identifier, issuer, or trust anchor. The executor's host-configured verifier matches the checksum and bindings against trusted configuration; fn-29 additionally pins the expected checksum.

## Considered Options

- Detached Ed25519 signatures — rejected because this internal, pinned-plan interface does not justify a cryptographic key lifecycle.
- Caller-provided trust claims — rejected because callers cannot establish their own model authority.

## Consequences

- Model-bound authority depends on trusted host configuration.
- A future public or unpinned interface may require a separately versioned authentication mechanism.
