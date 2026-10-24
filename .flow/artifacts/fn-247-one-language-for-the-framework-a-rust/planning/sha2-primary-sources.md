# sha2 0.11 primary-source facts for fn247.2

Checked 2026-10-24. The admitted `sha2 = "0.11"` requirement currently has stable 0.11.1 available. Its one-shot and incremental APIs support in-process hashing directly. This report records evidence for the existing admission; it does not change dependency policy or choose the final resolution. No Cargo download/build/install, source edit or Flow command ran.

## Exact releases

The [official crates.io index](https://raw.githubusercontent.com/rust-lang/crates.io-index/master/sh/a2/sha2) supplies these records, fetched directly with curl. Both are non-yanked.

| Version | Published UTC | MSRV | Registry checksum |
| --- | --- | --- | --- |
| 0.11.1 | 2026-10-12T18:58:22Z | 1.88 | `47d7069beb7d6ac7b9acd1039986e73443f24234f41074da099d6f994ac9ad19` |
| 0.11.0 | 2026-03-25T15:05:27Z | 1.85 | `446ba717509524cb3f22f17ecc096f10f4822d76ab5c0b9822c5f9c284e825f4` |

Both satisfy the stated R18 release-within-12-months basis. [Tagged 0.11.1 Cargo.toml](https://raw.githubusercontent.com/RustCrypto/hashes/sha2-v0.11.1/sha2/Cargo.toml) independently confirms version/MSRV. The current workspace rust-version 1.99 exceeds either requirement. These checksums identify registry archives; this research did not fetch or hash the archives.

## API and features

The [0.11.1 documentation](https://docs.rs/crate/sha2/0.11.1) demonstrates `use sha2::{Sha256, Digest};`, `Sha256::digest(bytes)`, and incremental `Sha256::new()`, `update(bytes)`, `finalize()`. These are exact 0.11 APIs, independently verified from the cached 0.10 version. No subprocess or additional hex crate is necessary for hashing; formatting digest bytes can use existing Rust facilities.

The [tagged library](https://raw.githubusercontent.com/RustCrypto/hashes/sha2-v0.11.1/sha2/src/lib.rs) declares `#![no_std]`, reexports `Digest` and defines Sha256 with 32-byte output. Its [manifest](https://raw.githubusercontent.com/RustCrypto/hashes/sha2-v0.11.1/sha2/Cargo.toml) has default features `alloc` and `oid`, forwarded to digest; optional `zeroize` forwards to digest and is off by default. There is no `std` feature to request. Runtime dependencies are digest ^0.11 and cfg-if ^1, plus cpufeatures ^0.3 on aarch64/x86/x86_64. hex-literal is a dev dependency, not a normal dependency of xtask's sha2 use.

## Candidate transitive inventory and R18 basis

The following latest non-yanked versions satisfy the inspected constraints. Each date comes from that package's official index `pubtime`. Every listed new package has a release within 12 months of this report, so the alternate finished/widely-used basis is unnecessary for these candidates.

| Candidate package | Release date | Basis / dependency edge |
| --- | --- | --- |
| [digest 0.11.3](https://raw.githubusercontent.com/rust-lang/crates.io-index/master/di/ge/digest) | 2026-05-03 | New; sha2 dependency. Defaults enable block-api. |
| [crypto-common 0.2.2](https://raw.githubusercontent.com/rust-lang/crates.io-index/master/cr/yp/crypto-common) | 2026-05-19 | New; digest's required dependency, named `common`. |
| [block-buffer 0.12.1](https://raw.githubusercontent.com/rust-lang/crates.io-index/master/bl/oc/block-buffer) | 2026-06-11 | New; digest 0.11.3 block-api enables ^0.12. |
| [const-oid 0.10.2](https://raw.githubusercontent.com/rust-lang/crates.io-index/master/co/ns/const-oid) | 2026-01-07 | New with sha2 defaults through digest/oid. |
| [hybrid-array 0.4.15](https://raw.githubusercontent.com/rust-lang/crates.io-index/master/hy/br/hybrid-array) | 2026-09-08 | New; crypto-common ^0.4.7 and block-buffer ^0.4. |
| [typenum 1.20.1](https://raw.githubusercontent.com/rust-lang/crates.io-index/master/ty/pe/typenum) | 2026-05-29 | New; hybrid-array ^1.20 enables const-generics. |
| [cpufeatures 0.3.1](https://raw.githubusercontent.com/rust-lang/crates.io-index/master/cp/uf/cpufeatures) | 2026-08-26 | Already in inspected model/Cargo.lock; platform-specific libc dependency. |
| [cfg-if 1.0.5](https://raw.githubusercontent.com/rust-lang/crates.io-index/master/cf/g-/cfg-if) | 2026-09-16 | Already in inspected model/Cargo.lock. |

The first six package names were absent from the inspected model lock; cfg-if, cpufeatures and admitted libc 0.2.191 were present. This establishes a candidate delta, not an actual Cargo resolve or feature-unification receipt. The [digest 0.11.3 tagged manifest](https://raw.githubusercontent.com/RustCrypto/traits/digest-v0.11.3/digest/Cargo.toml) confirms block-buffer ^0.12 and default block-api. Do not copy the older digest 0.11.0 dependency graph, which used block-buffer ^0.11; docs.rs also marks digest 0.11.0 yanked.

For this standalone default graph, optional zeroize, ctutils/MAC, blobby/dev, getrandom and rand_core features are not requested. hybrid-array's optional serde/bytemuck/zerocopy/subtle integrations are also not requested. Existing workspace feature unification could change the actual inventory; the implementer's resolved before/after receipt is authoritative. Setting sha2 default-features=false would omit its alloc/oid forwarding, but digest still has its own default block-api because sha2's dependency declaration enables digest defaults. This fact is not a recommendation to change the approved feature choice.

## Access and remaining uncertainty

crates.io's version API failed through the browser tool and returned HTTP 403 through curl. The sparse index and one docs.rs source page were unavailable through the browser tool. Official GitHub index records, exact RustCrypto tags and docs.rs release pages succeeded; no retry loop or package download was needed. Actual resolver selection, archive verification, feature inventory and compilation remain normal implementation work. No admission block follows from this research, and no API claim relies on locally cached sha2 0.10.9.
