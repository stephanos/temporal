# Compatibility Pack Review: temporal-functional-tests-darwin-arm64

Review SHA-256: `sha256:b009df6c9380eeb9766b3bf7144df7d6477018ee879ae8586ed2a2742b841ce6`

Owner: `temporal-server`

Reviewed at: `2026-09-27T00:00:00Z`

Justification: Admits the host process-metrics reads (getrlimit, getrusage, and the kern.proc.pid sysctl) of the Prometheus client process collector that the gomad build of the Temporal functional test package reaches on darwin/arm64; the arm64 assembly it also reaches is admitted by temporal-functional-compute-darwin-arm64, and every other host-only path the closure reaches is cut by the gomad build seams or a deterministic adapter, so the two packs close the ./tests capability closure.

Target: `go-test ./tests`

Target module: `go.temporal.io/server`

Test arguments: `-test.run ^TestActivityAPIBatchCancelClientTestSuite$ -test.count=1`

Build tags: `disable_grpc_modules,gomad,test_dep`

Platform: `darwin/arm64`

Workload: `functional-tests`

## Activation

- `github.com/prometheus/client_golang@v1.21.0` (`h1:DIsaGmiaBkSangBgMtWdNfxbMNdku5IK6iNhrEqWvdA=`), replacement `none`

## Reviewed packages

### `github.com/prometheus/client_golang/prometheus`

Module: `github.com/prometheus/client_golang@v1.21.0` (`h1:DIsaGmiaBkSangBgMtWdNfxbMNdku5IK6iNhrEqWvdA=`), replacement `none`

Source set: `sha256:4d129c8975776c9a64237c527855712d4f8c02c20803b0a79dc3e08e6a764383`

Go sources:

- `atomic_update.go`: `sha256:5c72eb6322aa7feadbd43c3b2b8261a101023b57f80a30fd34a7ee52786696dd`
- `build_info_collector.go`: `sha256:ee4dcc3036980d3ae6c1dca475312ebd4fc2d1b5950d79ea96594324fdaf8c55`
- `collector.go`: `sha256:07f54d03f1f6f463d8dce8132bc0de79eb137553e88bb890c7ab75b1ff04c568`
- `counter.go`: `sha256:1b611ba3e525d4815a636a9ccb6569f8801fd1e0c1f759f3b24c005018f32860`
- `desc.go`: `sha256:a405c5efefe711b8e628bcd1ed243ad2a5969309a6811fe5b005c4ce18e188fd`
- `doc.go`: `sha256:6e120e8402699d5f622c2b548ce23fba0b4b7d11eb95c15319ac83533a93e03f`
- `expvar_collector.go`: `sha256:585f85484c0fe9fed58619632f571c9b1146f5b36378af9d12a90c78b2af4d57`
- `fnv.go`: `sha256:e1021823eb58059432669ef5485e5125219b69259f21effe69aa09349649799a`
- `gauge.go`: `sha256:df178108b4f34c27c2c1a5038ca9017217f45c240fefa3d5a8044eac7df1ea91`
- `get_pid.go`: `sha256:f4ed88436b9a3eb8bc85e2ea85841f42a6f2a7d2f84eac204ffb9d1d9f196fb7`
- `go_collector.go`: `sha256:4b0bff8a04e231305b91d50f0a470219cad9044799562b63c73548d0dbc9b8e7`
- `go_collector_latest.go`: `sha256:9bd335e44c1f2c54f40c525e8b1a5a168b8472d81429b04e2d8f51aa785d215d`
- `histogram.go`: `sha256:5362745f01cf9f33abe580b50c1632d088b36f10b94cc8479d2df5a059bb4e96`
- `labels.go`: `sha256:f44a9a969d39d84fe3b23d054713912f836336b58d50bd5e24c3d5a75974f29e`
- `metric.go`: `sha256:975fb8150654965c83b9edfcdc2962815d0e5172aa401c571a90082a053f3f97`
- `num_threads.go`: `sha256:0da811d17c6bdb7f74c2cdac0251102b46141799c7dd83c76bf9634487f330e0`
- `observer.go`: `sha256:2a5ba8ed590dd168fe4a495d973cefaae068603decd9ecfc04b58ed82f8211f0`
- `process_collector.go`: `sha256:e98f3af3c9a72b91d4671573cb368af3095992a87f7f63637eea0c3cb4f49688`
- `process_collector_darwin.go`: `sha256:7fe1e6019dfc44f7a3f8509e8de784198c556760121db8cfe2a46b7294d0070c`
- `process_collector_nocgo_darwin.go`: `sha256:3c478be065ec9c2b88c3a321899b6b5c1beaad090f875239c9997a3e53dd975d`
- `registry.go`: `sha256:5303d23f4f3ff278eced6b3e5584ab74fe17818aac6c8a47bef0bb6f2f224bb0`
- `summary.go`: `sha256:d52e95b398f723323f2cbc3fad76fb4754caed088e24c772725cbcc2b3170c6c`
- `timer.go`: `sha256:51ed96cd19c8c85a508a51ae8d5b4fb42228a3a666c8b1be189c7653d288cbe3`
- `untyped.go`: `sha256:c5e028a25970462302a68d21b747fdd7505f50c4ddd2450192dbe55f9d12b9cf`
- `value.go`: `sha256:ad12774dc6e7795be1286731e70ebf2fced462db3c376e9942c645ae2898fda5`
- `vec.go`: `sha256:974211df154540f00c8674473fbf73022f74ace6d2308ff89374832cfc4c315a`
- `vnext.go`: `sha256:86e0dfb70687bb650cca5f8eda0295a744383a5812bdf24b4107b96ac9211df7`
- `wrap.go`: `sha256:bce5889a7d303b6f0ca4f070b1ec7cad362b222582793505aa8cc01f03bfbb93`

Requested facts:

- `import:golang.org/x/sys/unix`: **allow** — **security-sensitive**
- `import:syscall`: **allow** — **security-sensitive**

