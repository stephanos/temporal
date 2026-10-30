# TLC producer v2 schema (source-only)

Exact fields and collector invariants are in `fn25412-tlc-producer-schema.json`. Producer implementation is not runtime validated. Authorities use JSON version 1; event protocol uses version 2. Paths and untrusted text are lower-case byte hex; hashes are SHA-256. Only fixture records carry authority references. Explicit parent assertion completion is required in addition to matching process terminal evidence.
