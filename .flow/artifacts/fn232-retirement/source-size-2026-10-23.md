# fn-232.13 Go versus Rust code size (pre-freeze, 2026-10-23)

Provisional: candidate is branch `fn232-retirement-v2` at 2185b5d140, not the
final frozen candidate. Rerun both commands on the frozen candidate for the
final ledger row.

```sh
python3 tools/gomad/scripts/source-size.py --baseline 1d7b37f00a --candidate 2185b5d140 \
  --output /home/agent/wt/evidence/fn232/source-size-2026-10-23/source-size.json
python3 -I .flow/artifacts/fn232-retirement/source-size-groups.py \
  /home/agent/wt/evidence/fn232/source-size-2026-10-23/source-size.json
```

Baseline 1d7b37f00a is the frozen Go baseline (evidence/fn232/go-baseline-1d7b37f00a).
`source-size.json` SHA-256 c7ad6f4e927a7a345b1f95cabd04ac3b872adeb41d025065d52d134efdf83ace
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
| go-wasi-reference (moved to internal/model/wasi) | Go | tests | 61 | 954818 | 21147 | 51 | 586472 | 13565 |
| gomad-other (docs, scripts, parity testdata) | Go | tests | 73 | 21729 | 350 | 94 | 307041 | 6498 |
| gomad-other (docs, scripts, parity testdata) | Make | production | 1 | 22115 | 227 | 1 | 7369 | 108 |
| gomad-other (docs, scripts, parity testdata) | Python | generated | 0 | 0 | 0 | 1 | 2206 | 30 |
| gomad-other (docs, scripts, parity testdata) | Python | production | 0 | 0 | 0 | 3 | 62330 | 1031 |
| gomad-other (docs, scripts, parity testdata) | Python | tests | 0 | 0 | 0 | 103 | 1463403 | 22350 |
| gomad-other (docs, scripts, parity testdata) | Shell | tests | 0 | 0 | 0 | 1 | 903 | 24 |
| gomad-other (docs, scripts, parity testdata) | TOML | production | 1 | 95 | 2 | 1 | 95 | 2 |
| retained-go-model-oracle-toolchain-support | Go | generated | 13 | 163309 | 3900 | 13 | 164656 | 3918 |
| retained-go-model-oracle-toolchain-support | Go | production | 146 | 1318638 | 33530 | 154 | 1203386 | 31158 |
| retained-go-model-oracle-toolchain-support | Go | tests | 167 | 1255577 | 30829 | 135 | 867806 | 21607 |
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
| go-wasi-reference (moved to internal/model/wasi) | 85 | 1286015 | 30346 | 74 | 858916 | 21209 |
| gomad-other (docs, scripts, parity testdata) | 75 | 43939 | 579 | 204 | 1843347 | 30043 |
| repository-other | 3423 | 40991241 | 925046 | 3428 | 40962630 | 924536 |
| retained-go-model-oracle-toolchain-support | 328 | 2754813 | 68628 | 305 | 2246307 | 56963 |
| rust-helper (pre-rewrite location) | 83 | 1710919 | 43336 | 0 | 0 | 0 |
| rust-host (gomad crate, driver, tools) | 0 | 0 | 0 | 415 | 6372871 | 138368 |
| rust-linked-engine | 0 | 0 | 0 | 152 | 2647488 | 67221 |

baseline 1d7b37f00a74a85399993f1edcad6fe742e778e3 repository files 9938 bytes 238487870 paths_sha256 605131da238369abf0b610b9c3785f122a0cb401f0fca5e346f7485bc4ba1357
candidate 2185b5d14015736bdd85dfe43a58a8cf69bcc75d repository files 9827 bytes 271660908 paths_sha256 38025862637e772c9b947023394f2a42952a5fb7a53f3f5e8e9f360216dbfcda
counter gomad-source-size/v1 e9e4a58cbf7e3ddefebee91ea682bc57262fe5925b9d3f2a375b27ab9c0919ae 2185b5d14015736bdd85dfe43a58a8cf69bcc75d
