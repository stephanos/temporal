# World, network and selected boot receiving

These functions accept immutable buffers independently reopened after B04 command/artifact custody. They never open producer-selected paths or accept a producer PASS field. B15 owns stage routing; missing routes remain errors.

- `world(raw,encoding,broker,sha256)` authenticates raw bytes and calls the engine's authoritative encoded World admission. The returned read-only projection has `summary()`, `requests()`, `events()` and `transitions()`. Every fact comes from canonical replay; no second binary parser is involved.
- `measured(raw,encoding,requests,sha256)` checks the fixed three-request owner or measured 20,000/200,000-request workload. It validates all terminal owners and every register/ready/quiesce body, counts, time, IDs and payload accounting. Caller memory/measurement bounds remain separate.
- `network(raw,limit,shared,local,sha256,record,log_sha256,world_sha256)` admits the current network wire and compares its actual record and stream identities. Shared network limits and local World limits retain independent authority.
- `boot_rows(stdout)` reads the complete thirteen-variant selected-model harness, checks exact current authored input, immutable caller ownership, every selected incarnation and byte/stall binding, no-store admission, and the admitted shared identity.

`corpus::Corpus::admit(kind,raw,frozen,pin)` authenticates the complete independently frozen current file inventory. `receive(stdout)` compares all World, clock or native network corpus responses and states; World rows also admit the actual raw/compressed roundtrip. Broker callers use `receive_broker(stdout,raw_node_streams,frozen_node_hashes)`, which requires every node stream, compares actual bytes and invokes authoritative network admission. Output-only broker receiving fails closed.

The existing engine `validate_world`, `validate_encoded_world` and `validate_network_evidence` wrappers retain their refusal behavior. The new projections expose no mutable engine model.

Legacy-only reconstructed Go JSON byte equality and historical pin substitutions are candidates for exact-ID retirement under the current compatibility policy. They supply no execution proof.
