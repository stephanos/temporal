(module
  (import "gomad_wasm_v1" "config" (func $config (param i32 i32) (result i32)))
  (import "gomad_wasm_v1" "decision" (func $decision (param i32 i32) (result i32)))
  (import "gomad_wasm_v1" "observation" (func (param i32 i32) (result i32)))
  (import "gomad_wasm_v1" "finish" (func $finish (param i32 i32) (result i32)))
  (import "gomad_wasm_v1" "idle" (func (param i32 i32) (result i32)))
  (memory (export "memory") 1)
  (data (i32.const 16) "\00\00\00\01")
  (data (i32.const 64) "\00\00\00\01\00\00\00\01\00\00\00\00\00\00\00\2a\00\00\00\00\00\00\00\02\00\00\00\01\00\00\00\00\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\02\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01\01")
  (func (export "_start")
    i32.const 0 i32.const 16 call $config if unreachable end
    i32.const 64 i32.const 96 call $decision if unreachable end
    i32.const 64 i32.const 96 call $decision if unreachable end
    i32.const 16 i32.const 16 call $finish if unreachable end))
