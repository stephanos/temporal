(module
  (import "gomad_wasm_v1" "config" (func $config (param i32 i32) (result i32)))
  (import "gomad_wasm_v1" "decision" (func $decision (param i32 i32) (result i32)))
  (import "gomad_wasm_v1" "finish" (func $finish (param i32 i32) (result i32)))
  (import "wasi_snapshot_preview1" "path_open" (func $open (param i32 i32 i32 i32 i32 i64 i64 i32 i32) (result i32)))
  (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32)))
  (import "wasi_snapshot_preview1" "fd_sync" (func $sync (param i32) (result i32)))
  (memory (export "memory") 1)
  (data (i32.const 16) "\00\00\00\01")
  (data (i32.const 64) "\00\00\00\01\00\00\00\01\00\00\00\00\00\00\00\2a\00\00\00\00\00\00\00\02\00\00\00\01\00\00\00\00\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01")
  (data (i32.const 160) "\00\00\00\01\00\00\00\01\00\00\00\00\00\00\00\2b\00\00\00\00\00\00\00\02\00\00\00\01\00\00\00\00\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01")
  (data (i32.const 512) "db") (data (i32.const 528) "ok")
  (func (export "_start")
    i32.const 0 i32.const 16 call $config if unreachable end
    i32.const 64 i32.const 96 call $decision if unreachable end
    i32.const 3 i32.const 0 i32.const 512 i32.const 2 i32.const 1 i64.const 536870911 i64.const 0 i32.const 0 i32.const 560 call $open if unreachable end
    i32.const 544 i32.const 528 i32.store i32.const 548 i32.const 2 i32.store
    i32.const 560 i32.load i32.const 544 i32.const 1 i32.const 552 call $write if unreachable end
    i32.const 560 i32.load call $sync if unreachable end
    i32.const 160 i32.const 96 call $decision if unreachable end
    i32.const 16 i32.const 16 call $finish if unreachable end))
