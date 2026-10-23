# Namespace conformance corpus

These test-only vectors bind the Go namespace reference. Each JSON file contains
an initial namespace image, WASI operation inputs, expected replies and a final
state digest. `recorded-stock-guest.json` extracts filesystem calls unchanged
from the retained stock WASM guest artifact named and hashed in its `origin`.
Its expected replies come from the current Go reference.

Run the drift check from `tools/gomad`:

```sh
go test -tags test_dep -count=1 -run '^TestNamespaceConformanceCorpus$' ./internal/model/wasi
```

After inspecting an intended semantic change, append `-args -update-conformance`
to regenerate the vectors. Live differential testing uses the non-default
conformance helper via `make -C internal/backends/wasm test-conformance`; it also
runs 128 operations for each of 100 fixed seeds and reports the first mismatch.
The helper is a test executable and never an installed runtime.

`repeat_data` replaces the input's `data_base64` with that many copies of byte
`x` before execution. This keeps the write-buffer overflow fixture compact.
The driver wire protocol always receives ordinary WASI input fields.

The digest format and session protocol are defined in
[`rust/wasmhost/protocol.md`](../../../../../rust/wasmhost/protocol.md) under “Conformance driver”. These vectors supply
source conformance controls; they do not qualify a guest, runtime integration,
replay, platform or performance threshold.
