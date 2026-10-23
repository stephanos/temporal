# fn-232.13 Go versus Rust code size (pre-freeze, 2026-10-23)

Provisional: candidate is branch `fn232-retirement-v2` at fac2e3f4cb, not the
final frozen candidate. Rerun both commands on the frozen candidate for the
final ledger row.

```sh
python3 tools/gomad/scripts/source-size.py --baseline 1d7b37f00a --candidate fac2e3f4cb \
  --output /home/agent/wt/evidence/fn232/source-size-2026-10-23/source-size.json
python3 -I .flow/artifacts/fn232-retirement/source-size-groups.py \
  /home/agent/wt/evidence/fn232/source-size-2026-10-23/source-size.json
```

Baseline 1d7b37f00a is the frozen Go baseline (evidence/fn232/go-baseline-1d7b37f00a).
`source-size.json` SHA-256 6ea6796b75b85907c2e28b9fd179919eca7ee97a64012a601aef440da8fbc355
(7.4 MB, retained outside Git with complete per-file path/blob/SHA-256 inventories).
Code lines are the counter's physical lexical metric, not cloc. Groups come from
`source-size-groups.py` path roots; the native backend (fn-223) and the moved
WASI reference are shown separately so neither counts as the Rust port.

| group | language | category | base files | base bytes | base code | cand files | cand bytes | cand code |
| --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: |
| go-host (retired by fn-232) | Go | production | 217 | 2291895 | 51329 | 0 | 0 | 0 |
| go-host (retired by fn-232) | Go | tests | 215 | 2463933 | 53341 | 6 | 8243 | 336 |
| go-native-backend (fn-223) | Go | generated | 26 | 271981 | 4744 | 0 | 0 | 0 |
| go-native-backend (fn-223) | Go | production | 205 | 1835665 | 45511 | 0 | 0 | 0 |
| go-native-backend (fn-223) | Go | tests | 324 | 2177903 | 51195 | 0 | 0 | 0 |
| go-native-backend (fn-223) | Shell | production | 2 | 806 | 35 | 0 | 0 | 0 |
| go-native-backend (fn-223) | Shell | tests | 1 | 3049 | 79 | 0 | 0 | 0 |
| go-wasi-reference (moved to internal/model/wasi) | Go | production | 24 | 331197 | 9199 | 23 | 272444 | 7644 |
| go-wasi-reference (moved to internal/model/wasi) | Go | tests | 61 | 954818 | 21147 | 51 | 586471 | 13565 |
| gomad-other (docs, scripts, parity testdata) | Go | tests | 73 | 21729 | 350 | 94 | 307041 | 6498 |
| gomad-other (docs, scripts, parity testdata) | Make | production | 1 | 22115 | 227 | 1 | 7369 | 108 |
| gomad-other (docs, scripts, parity testdata) | Python | generated | 0 | 0 | 0 | 1 | 2206 | 30 |
| gomad-other (docs, scripts, parity testdata) | Python | production | 0 | 0 | 0 | 3 | 62330 | 1031 |
| gomad-other (docs, scripts, parity testdata) | Python | tests | 0 | 0 | 0 | 103 | 1463403 | 22350 |
| gomad-other (docs, scripts, parity testdata) | Shell | tests | 0 | 0 | 0 | 1 | 903 | 24 |
| gomad-other (docs, scripts, parity testdata) | TOML | production | 1 | 95 | 2 | 1 | 95 | 2 |
| retained-go-model-oracle-toolchain-support | Go | generated | 13 | 163309 | 3900 | 13 | 164656 | 3918 |
| retained-go-model-oracle-toolchain-support | Go | production | 146 | 1318638 | 33530 | 154 | 1203464 | 31166 |
| retained-go-model-oracle-toolchain-support | Go | tests | 167 | 1255577 | 30829 | 135 | 867828 | 21607 |
| retained-go-model-oracle-toolchain-support | Make | production | 1 | 8305 | 121 | 1 | 332 | 11 |
| retained-go-model-oracle-toolchain-support | Rust | production | 1 | 8984 | 248 | 2 | 10127 | 269 |
| rust-helper (pre-rewrite location) | Rust | generated | 1 | 6703 | 160 | 0 | 0 | 0 |
| rust-helper (pre-rewrite location) | Rust | production | 61 | 1363196 | 34879 | 0 | 0 | 0 |
| rust-helper (pre-rewrite location) | Rust | tests | 19 | 339856 | 8261 | 0 | 0 | 0 |
| rust-helper (pre-rewrite location) | TOML | production | 2 | 1164 | 36 | 0 | 0 | 0 |
| rust-host (gomad crate, driver, tools) | Go | production | 0 | 0 | 0 | 1 | 1744 | 65 |
| rust-host (gomad crate, driver, tools) | Python | production | 0 | 0 | 0 | 68 | 1414312 | 19715 |
| rust-host (gomad crate, driver, tools) | Python | tests | 0 | 0 | 0 | 71 | 835655 | 11588 |
| rust-host (gomad crate, driver, tools) | Rust | production | 0 | 0 | 0 | 210 | 3222469 | 84230 |
| rust-host (gomad crate, driver, tools) | Rust | tests | 0 | 0 | 0 | 61 | 896240 | 22701 |
| rust-host (gomad crate, driver, tools) | Shell | production | 0 | 0 | 0 | 1 | 725 | 14 |
| rust-host (gomad crate, driver, tools) | TOML | production | 0 | 0 | 0 | 3 | 1726 | 55 |
| rust-linked-engine | Go | tests | 0 | 0 | 0 | 1 | 2167 | 89 |
| rust-linked-engine | Rust | generated | 0 | 0 | 0 | 2 | 7732 | 180 |
| rust-linked-engine | Rust | production | 0 | 0 | 0 | 104 | 1960018 | 50316 |
| rust-linked-engine | Rust | tests | 0 | 0 | 0 | 44 | 675988 | 16578 |
| rust-linked-engine | TOML | production | 0 | 0 | 0 | 1 | 1583 | 58 |

| group (all languages/categories) | base files | base bytes | base code | cand files | cand bytes | cand code |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| go-host (retired by fn-232) | 432 | 4755828 | 104670 | 6 | 8243 | 336 |
| go-native-backend (fn-223) | 558 | 4289404 | 101564 | 0 | 0 | 0 |
| go-wasi-reference (moved to internal/model/wasi) | 85 | 1286015 | 30346 | 74 | 858915 | 21209 |
| gomad-other (docs, scripts, parity testdata) | 75 | 43939 | 579 | 204 | 1843347 | 30043 |
| repository-other | 3423 | 40991241 | 925046 | 3427 | 40959001 | 924472 |
| retained-go-model-oracle-toolchain-support | 328 | 2754813 | 68628 | 305 | 2246407 | 56971 |
| rust-helper (pre-rewrite location) | 83 | 1710919 | 43336 | 0 | 0 | 0 |
| rust-host (gomad crate, driver, tools) | 0 | 0 | 0 | 415 | 6372871 | 138368 |
| rust-linked-engine | 0 | 0 | 0 | 152 | 2647488 | 67221 |

baseline 1d7b37f00a74a85399993f1edcad6fe742e778e3 repository files 9938 bytes 238487870 paths_sha256 605131da238369abf0b610b9c3785f122a0cb401f0fca5e346f7485bc4ba1357
candidate fac2e3f4cba62bd3ad28ef6d72effd9c4e0e8425 repository files 9825 bytes 271649866 paths_sha256 4ee9ef9b01bdb3596c2c04ea9010f99014b74a1e4479aae08323eea54a07b303
counter gomad-source-size/v1 e9e4a58cbf7e3ddefebee91ea682bc57262fe5925b9d3f2a375b27ab9c0919ae fac2e3f4cba62bd3ad28ef6d72effd9c4e0e8425
