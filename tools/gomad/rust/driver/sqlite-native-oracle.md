# Independent SQLite catalogue oracle

`sqlite.aggregate` accepts only `config.sqlite_oracle = {"build_receipt": "/absolute/actual/build.json"}`. A stage `oracle` command is rejected. The reader resolves the executable from an authenticated build of the fixed current Go package `./internal/model/cmd/sqlite-native`; a pinned Rust binary or caller-supplied argv supplies no independent comparison.

Before candidate freezing, create a closed plan with schema `gomad.sqlite-go-oracle-plan/v1` and exactly `repo`, `stock_go`, `go_cache`, and `module_cache`. All paths are absolute canonical assigned directories/files. The compiler is the actual pinned Go1.27.1 distribution. The caches are external to the candidate checkout. The module cache must already contain the source, zip, ziphash, mod and info for `github.com/go-json-experiment/json@v0.0.0-20260820222146-c27c302e5fc3` and `golang.org/x/sys@v0.48.0`, whose checksums remain the candidate's original go.sum identities. No module download/network fallback is enabled.

After the conductor grants the compiler lease:

```sh
python3 tools/gomad/rust/driver/sqlite_oracle.py build \
  --plan /absolute/sqlite-oracle-plan.json --root /absolute/fresh/oracle-build
```

This produces a genuine Go executable and `build.json`, retaining every command's raw stdout/stderr and actual process journals. It executes fixed stock-Go version, environment, package-graph, module verification, build and executable build-info commands. Module verification uses a retained verification-only modfile containing exactly those two requirements and their original go.sum lines; compilation still uses the unchanged actual tools/gomad module. This avoids borrowing unrelated server module-cache inputs for the checksum check. Compilation is pure Go, native host, `-p=1 -mod=readonly -tags=test_dep -trimpath -buildvcs=false`; the exact environment disables ambient Go flags/workspaces/toolchain switching/proxies and records the assigned caches and temporary directory.

The reader authenticates the whole stock SDK, current model/support-fs sources and go.mod/go.sum, both fixed source dependencies, actual complete reachable Go package graph, the fixed command vectors, exact compiler/environment, binary SHA, embedded Go package/module/build settings, clean numeric reaps and shared CLOCK_MONOTONIC observations. Rust/current host algorithms cannot be selected as the Go package. Every source, compiler, SDK, dependency, executable and raw receipt file joins the frozen file closure through `sqlite_oracle.freeze_inputs(config, entry)`; `validate_build` also requires the compiler to equal `config.primary.compiler` and the profile stock compiler when present. The freezer must require this binding and reject old stage `oracle` fields.

During the real catalogue stage the only oracle command is `[authenticated_binary, "--input", actual_oracle_input_path]`. `invoke` first asks the authenticated stock Go executable to reopen the actual oracle build information and refuses before invoking any non-Go/self-comparison executable. It then captures the fixed oracle call; `read_run` independently reopens both commands, their sequential shared-parent clocks, source build, request/output hashes and PID/reap facts. The request comes from authenticated primary artifact state, storage log, precrash image and actual descriptor model identity. The returned complete enumeration/context is compared exactly with the Rust primary and all four selected Results.

The original Go projector retains bounds **32 states / 8 operations / depth8 / 4MiB / 2^16 work**. Its existing request/state/log/precrash read bounds remain 64KiB/8MiB/32MiB/9MiB. No Go source, model algorithm, bound or selection is changed by this receiver.

Python mutation tests are source checks. Authentic Go compilation and actual execution against the produced SQLite evidence are mandatory prerequisites/final gates, respectively; a source review or fabricated receipt never supplies their pass.
