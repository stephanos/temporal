# Cross-OS Rust/Wasm byte identity

Research date: 2026-10-21. This investigation combines upstream primary sources,
existing proof artifacts and an isolated feasibility probe. It does not change
the guest builder, alter task state, or certify the full guests.

## Feasibility

**Yes: native macOS and Linux Rust toolchains can produce byte-identical Wasm.**
An isolated local probe confirms this with our pinned Rust/Cargo 1.99.1 and exact
`wasm32-wasip1` target. Native macOS/arm64 and Linux/arm64 each compiled this code
directly with rustc and through Cargo:

```rust
#[unsafe(no_mangle)]
pub extern "C" fn add(a: u32, b: u32) -> u32 {
    a.wrapping_add(b)
}
```

All four outputs are identical: **97 bytes**, SHA-256
`5a3de7bad08fedb5dd5b589e86aee5b89f88983f4694e7526ca6a7db1d05167b`.
Both Cargo commands pass `-Cmetadata=bfa0673fbd87bdb0`. Direct compilation uses
optimization level 3, no debug information and stripped symbols; the Cargo
profile additionally uses fat LTO, one codegen unit and overflow checks.
This probe has no dependencies, proc macros or build scripts and uses
`panic=abort`, without our guests' rebuilt unwinding std. It demonstrates
feasibility, not full-guest reproducibility. Sources, manifest, lockfile,
transcripts and binaries are in [the probe directory](../../tmp/fn232-feasibility/).

This is demonstrated upstream, rather than merely inferred from Wasm portability.
In Rust issue 117597, a small program initially matched across macOS/Ubuntu when
compiled directly with rustc, but differed through Cargo. After Cargo PR 14107,
the reporter confirmed consistent output. The example targets
`wasm32-unknown-unknown`; it does not establish identity for our `wasm32-wasip1`
guests with rebuilt unwinding std. [Rust issue and confirmation](https://github.com/rust-lang/rust/issues/117597#issuecomment-2199537540)

Cargo PR 14107, merged June 20, 2024, removed the compiler's host triple from the
version hash for cross-target units. Its author reports a before/after test where
two platforms produced different binaries before the change and identical ones
after it. [Cargo PR 14107](https://github.com/rust-lang/cargo/pull/14107)

Our target supports cross-compilation from any host and has a self-contained
sysroot. Thus `wasm32-wasip1` does not intrinsically require different target
instructions on macOS and Linux. This establishes a viable target architecture;
it is not a byte identity warranty. [Rust target documentation](https://doc.rust-lang.org/rustc/platform-support/wasm32-wasip1.html)

## What is and is not guaranteed

The evidence supports an intended, achievable property, not a blanket promise
that a pinned Rust release plus `Cargo.lock` makes every project identical across
host operating systems. `--locked` fixes dependency resolution; it does not
normalize host tools, generated code, environment, or compiler inputs.
[Cargo build documentation](https://doc.rust-lang.org/cargo/commands/cargo-build.html#manifest-options)

LLVM explicitly distinguishes builds stable across paths from builds stable
across machines. Its guidance requires deterministic tools, controlled compiler,
linker and SDK versions, and controlled environmental inputs. Its claim that the
tools can be deterministic with suitable flags does not certify our Rust/LLVM
pipeline or say the defaults suffice. [LLVM reproducibility guidance](https://blog.llvm.org/2019/11/deterministic-builds-with-clang-and-lld.html)

## Check against our pinned Cargo

The proof records Cargo 1.99.1 commit `5f94df478`. Inspection of that exact upstream
commit, expanded to `5f94df4789f005f9a352888e8355ffc645b7ed0e`, confirms:

- `hash_rustc_version` already skips the `host:` version line for target units.
  It deliberately retains the host for units compiled for the host.
- `compute_metadata` incorporates dependency metadata hashes into each unit's
  `c_metadata`. It also considers features, profile, compilation kind, package
  identity and target, among other inputs.
- Rebuilt std package identities are hashed relative to the sysroot, explicitly
  avoiding dependence on its installation directory.

[Pinned Cargo metadata implementation](https://github.com/rust-lang/cargo/blob/5f94df4789f005f9a352888e8355ffc645b7ed0e/src/compiler/build_runner/compilation_files.rs#L737-L940)

Consequently the old unconditional host-version-hashing bug is **not** an adequate
explanation for our failure. Cargo issue 13922 explicitly closed as fixed by
14107; treating it as an outstanding defect would be misleading.
[Cargo issue closure](https://github.com/rust-lang/cargo/issues/13922#issuecomment-2180948148)

An inference from the pinned implementation is that host dependencies can still
introduce host-specific metadata into a graph. Whether a particular build-script
or proc-macro edge does so here requires tracing the actual units and compiler
arguments; the source alone does not identify our root cause. Rust documents
`-Cmetadata` as an input to symbol mangling, so differences deserve investigation
even for stripped artifacts. [rustc metadata option](https://doc.rust-lang.org/rustc/codegen-options/index.html#metadata)

## Remaining sources of divergence

Build scripts execute native programs, can generate Rust, and receive host and
target configuration. The Cargo documentation specifically warns that `cfg!`
inside a build script tests its host, not its target. Printed instruction order
can also affect compiler and linker argument order. These are concrete places to
inspect rather than assume identical from a lockfile. [Cargo build scripts](https://doc.rust-lang.org/cargo/reference/build-scripts.html)

Procedural macros execute during compilation and have access to the compiler's
resources, including files. Their output can therefore depend on inputs outside
the target source. [Rust procedural macro reference](https://doc.rust-lang.org/reference/procedural-macros.html)

Different metadata, generated item order, object order, or link inputs can explain
different final layouts without a nondeterministic linker. The observed function
and data-layout changes do not establish an LLVM or rust-lld bug. LLVM/backend
differences remain a hypothesis to test after matching earlier inputs.

## What our existing proof establishes

At source `8d429e97409b1537ecb347c359129bffafa63f78`, both guests matched
between two clean paths on each OS, but neither guest matched across OSes. Stock
was 192255 bytes on Linux and 192263 on macOS; Temporal was 3329990 and 3329996.
Both platforms passed the retained host behavior suites. The stock WAT comparison
includes element-table, function-body and data-layout differences. These are
observations about this recipe, not evidence of impossibility.
[Existing proof](proof.md), [builder description](../../../model/GUEST_BUILDS.md)

Path independence on one OS does not eliminate host configuration or native
build-time dependencies. The actual compiler/linker cause remains unresolved by
the retained certification evidence.

## Practical next step

Retain native cross-OS identity as an attainable requirement and investigate the
earliest differing target input before changing it. A focused follow-up should:

1. Capture complete rustc invocations for each platform and compare per-crate
   `-Cmetadata`, features, cfgs, environment, generated source and link arguments.
   Start with rebuilt std and host build-script/proc-macro dependencies, since
   both guests already differ.
2. Trace the first differing target crate through Cargo's metadata dependency
   graph. Cache fingerprint differences alone are not proof that compiler
   metadata differs. Host executables are expected to differ; compare their
   generated outputs and effect on target compilation.
3. Once target source and compiler arguments agree, compare emitted LLVM IR,
   objects, archive contents/order and final linker inputs to locate the first
   divergence. Only identical inputs with divergent output implicate that tool
   stage.
4. Correct the specific input or toolchain issue and rerun the unchanged clean
   native two-OS byte comparisons and host suites. Upstream examples do not
   substitute for that certification.

At the research checkpoint these diagnostic steps had not been performed.
The subsequent [causal diagnosis](diagnosis.md) reproduced both full Linux guests
on native macOS by controlling Cargo metadata and generated-output path hashes.
It records the experiments and distinguishes them from a permanent builder fix.
Changing debug stripping alone is not supported by our observations:
the existing recipe already strips symbols and its differences reach executable
content.

If native reproducibility proves too costly, one alternative is a canonical
Linux build environment used from both developer OSes (a pinned VM/container or
remote builder). That controls the compiler's execution environment but changes
the claim: macOS developers can reproduce the canonical artifact, while native
macOS rustc has not been shown to produce it. This would require an explicit
change to fn-232's native cross-OS acceptance criteria.
