(module
  (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32)))
  (import "wasi_snapshot_preview1" "clock_time_get" (func $clock (param i32 i64 i32) (result i32)))
  (import "wasi_snapshot_preview1" "args_sizes_get" (func $args (param i32 i32) (result i32)))
  (import "wasi_snapshot_preview1" "random_get" (func $random (param i32 i32) (result i32)))
  (memory (export "memory") 1)
  (data (i32.const 128) "linked-output\0a")
  (func (export "_start")
    i32.const 0 i32.const 4 call $args if unreachable end
    i32.const 1 i64.const 1 i32.const 16 call $clock if unreachable end
    i32.const 32 i32.const 16 call $random if unreachable end
    i32.const 64 i32.const 128 i32.store
    i32.const 68 i32.const 14 i32.store
    i32.const 1 i32.const 64 i32.const 1 i32.const 72 call $write if unreachable end))
