# Oracle ABI v1

The generated protobuf messages in `umpire/oracle/v1/oracle.proto` are the wire contract.
Method enum numbers are stable dispatch IDs. `call` always returns a `Response` envelope.
Explore replaces the current exploration only on success; Rows needs a successful Explore.
Each rows string is one complete Spell-based native golden line, including its newline.

The wasm32-wasip1 guest exports application functions `alloc(u32) -> u32`,
`free(u32, u32)` and `call(u32, u32, u32) -> u64`, plus memory and toolchain lifecycle exports.
The packed result is `(length << 32) | pointer`. `alloc(0)` returns zero;
`free(0, 0)` succeeds. Other allocations are nonzero, initialized, guest-owned buffers.
The guest bounds individual buffers to 1 MiB and total live storage to 16 MiB.
Allocation exhaustion returns zero; invalid frees trap without dereferencing the supplied pointer.
`call` accepts only an exact live allocation range, with `(0, 0)` representing an empty request.
It retains the request and returns a separately owned response allocation, or zero on exhaustion.
Unknown methods, malformed requests and invalid checker queries return typed Refusal envelopes.
The host copies response bytes before freeing either allocation, then frees both exactly once.
Access to one instance is serialized. An unexpected trap invalidates the host instance.

Rust generates into Cargo's OUT_DIR with prost-build 0.14 and mise's protoc 29.5.
Go uses the repository's pinned protoc-gen-go v1.36.6 and protobuf runtime v1.36.11.
