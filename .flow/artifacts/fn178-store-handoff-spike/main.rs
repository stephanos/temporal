//! fn-178.1 throwaway spike: two Wasmtime stores of a cooperative guest that
//! hand off control at every host call.
//!
//! (a) one OS thread per store with a baton (Mutex+Condvar, and park/unpark)
//! (b) Wasmtime async fibers on one thread (call_async + func_wrap_async)
//!
//! Plus the per-call cost of a typed sync import with no handoff, called from a
//! sync store and from an async (fiber) store, and an async import that never
//! yields, to show what the async model costs imports that never switch.
use std::future::Future;
use std::pin::Pin;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::{Arc, Condvar, Mutex};
use std::task::{Context, Poll, RawWaker, RawWakerVTable, Waker};
use std::time::Instant;
use wasmtime::{Config, Engine, Linker, Module, Store};

const GUEST: &str = r#"(module
  (import "env" "hop" (func $hop))
  (memory (export "memory") 1)
  (func (export "run") (param $n i32)
    (loop $l
      call $hop
      local.get $n i32.const 1 i32.sub local.tee $n
      br_if $l)))"#;

fn config(async_store: bool) -> Config {
    let mut config = Config::new();
    config
        .consume_fuel(true)
        .native_unwind_info(false)
        .cranelift_nan_canonicalization(true)
        .wasm_relaxed_simd(false)
        .wasm_memory64(false)
        .wasm_multi_memory(false);
    if async_store {
        config.max_wasm_stack(512 << 10);
        config.async_stack_size(2 << 20);
    }
    config
}

struct Baton {
    turn: Mutex<usize>,
    wake: [Condvar; 2],
}

fn threads_condvar(n: i32) -> f64 {
    let engine = Engine::new(&config(false)).unwrap();
    let module = Module::new(&engine, GUEST).unwrap();
    let mut linker: Linker<usize> = Linker::new(&engine);
    let baton = Arc::new(Baton {
        turn: Mutex::new(0),
        wake: [Condvar::new(), Condvar::new()],
    });
    let b = baton.clone();
    linker
        .func_wrap("env", "hop", move |caller: wasmtime::Caller<'_, usize>| {
            let me = *caller.data();
            let other = 1 - me;
            let mut turn = b.turn.lock().unwrap();
            *turn = other;
            b.wake[other].notify_one();
            while *turn != me {
                turn = b.wake[me].wait(turn).unwrap();
            }
        })
        .unwrap();
    let pre = linker.instantiate_pre(&module).unwrap();
    let pre = Arc::new(pre);
    let start = Instant::now();
    let handles: Vec<_> = (0..2usize)
        .map(|me| {
            let pre = pre.clone();
            let engine = engine.clone();
            let baton = baton.clone();
            std::thread::Builder::new()
                .stack_size(8 << 20)
                .spawn(move || {
                    let mut store = Store::new(&engine, me);
                    store.set_fuel(u64::MAX).unwrap();
                    let instance = pre.instantiate(&mut store).unwrap();
                    let run = instance.get_typed_func::<i32, ()>(&mut store, "run").unwrap();
                    {
                        let mut turn = baton.turn.lock().unwrap();
                        while *turn != me {
                            turn = baton.wake[me].wait(turn).unwrap();
                        }
                    }
                    run.call(&mut store, n).unwrap();
                    // Release the baton for the other store's tail.
                    let mut turn = baton.turn.lock().unwrap();
                    *turn = 1 - me;
                    baton.wake[1 - me].notify_one();
                })
                .unwrap()
        })
        .collect();
    for handle in handles {
        handle.join().unwrap();
    }
    start.elapsed().as_nanos() as f64 / (2.0 * n as f64)
}

fn threads_park(n: i32) -> f64 {
    let engine = Engine::new(&config(false)).unwrap();
    let module = Module::new(&engine, GUEST).unwrap();
    let turn = Arc::new(AtomicUsize::new(0));
    let threads: Arc<Mutex<Vec<std::thread::Thread>>> = Arc::new(Mutex::new(Vec::new()));
    let mut linker: Linker<(usize, Option<std::thread::Thread>)> = Linker::new(&engine);
    let t = turn.clone();
    linker
        .func_wrap(
            "env",
            "hop",
            move |caller: wasmtime::Caller<'_, (usize, Option<std::thread::Thread>)>| {
                let (me, other_thread) = caller.data();
                let me = *me;
                t.store(1 - me, Ordering::Release);
                other_thread.as_ref().unwrap().unpark();
                while t.load(Ordering::Acquire) != me {
                    std::thread::park();
                }
            },
        )
        .unwrap();
    let pre = Arc::new(linker.instantiate_pre(&module).unwrap());
    let ready = Arc::new(std::sync::Barrier::new(3));
    let start_gate = Arc::new(std::sync::Barrier::new(3));
    let handles: Vec<_> = (0..2usize)
        .map(|me| {
            let (pre, engine, turn, threads, ready, start_gate) = (
                pre.clone(),
                engine.clone(),
                turn.clone(),
                threads.clone(),
                ready.clone(),
                start_gate.clone(),
            );
            std::thread::Builder::new()
                .stack_size(8 << 20)
                .spawn(move || {
                    threads.lock().unwrap().push(std::thread::current());
                    ready.wait();
                    start_gate.wait();
                    let other = threads
                        .lock()
                        .unwrap()
                        .iter()
                        .find(|t| t.id() != std::thread::current().id())
                        .unwrap()
                        .clone();
                    let mut store = Store::new(&engine, (me, Some(other.clone())));
                    store.set_fuel(u64::MAX).unwrap();
                    let instance = pre.instantiate(&mut store).unwrap();
                    let run = instance.get_typed_func::<i32, ()>(&mut store, "run").unwrap();
                    while turn.load(Ordering::Acquire) != me {
                        std::thread::park();
                    }
                    run.call(&mut store, n).unwrap();
                    turn.store(1 - me, Ordering::Release);
                    other.unpark();
                })
                .unwrap()
        })
        .collect();
    ready.wait();
    let start = Instant::now();
    start_gate.wait();
    for handle in handles {
        handle.join().unwrap();
    }
    start.elapsed().as_nanos() as f64 / (2.0 * n as f64)
}

/// Returns Pending exactly once, then Ready.
struct YieldOnce(bool);
impl Future for YieldOnce {
    type Output = ();
    fn poll(mut self: Pin<&mut Self>, context: &mut Context<'_>) -> Poll<()> {
        if self.0 {
            Poll::Ready(())
        } else {
            self.0 = true;
            context.waker().wake_by_ref();
            Poll::Pending
        }
    }
}

fn noop_waker() -> Waker {
    fn clone(_: *const ()) -> RawWaker {
        RawWaker::new(std::ptr::null(), &VTABLE)
    }
    fn noop(_: *const ()) {}
    static VTABLE: RawWakerVTable = RawWakerVTable::new(clone, noop, noop, noop);
    unsafe { Waker::from_raw(RawWaker::new(std::ptr::null(), &VTABLE)) }
}

fn fibers(n: i32) -> f64 {
    let engine = Engine::new(&config(true)).unwrap();
    let module = Module::new(&engine, GUEST).unwrap();
    let mut linker: Linker<usize> = Linker::new(&engine);
    linker
        .func_wrap_async("env", "hop", |_caller, ()| Box::new(YieldOnce(false)))
        .unwrap();
    let pre = linker.instantiate_pre(&module).unwrap();
    let waker = noop_waker();
    let mut context = Context::from_waker(&waker);
    let mut stores: Vec<Store<usize>> = (0..2)
        .map(|me| {
            let mut store = Store::new(&engine, me);
            store.set_fuel(u64::MAX).unwrap();
            store
        })
        .collect();
    let (left, right) = stores.split_at_mut(1);
    let start = Instant::now();
    let mut futures: Vec<Pin<Box<dyn Future<Output = ()>>>> = [&mut left[0], &mut right[0]]
        .into_iter()
        .map(|store| {
            let pre = &pre;
            Box::pin(async move {
                let instance = pre.instantiate_async(&mut *store).await.unwrap();
                let run = instance
                    .get_typed_func::<i32, ()>(&mut *store, "run")
                    .unwrap();
                run.call_async(&mut *store, n).await.unwrap();
            }) as Pin<Box<dyn Future<Output = ()>>>
        })
        .collect();
    let mut done = [false, false];
    let mut switches = 0u64;
    while !(done[0] && done[1]) {
        for index in 0..2 {
            if done[index] {
                continue;
            }
            switches += 1;
            if futures[index].as_mut().poll(&mut context).is_ready() {
                done[index] = true;
            }
        }
    }
    let elapsed = start.elapsed().as_nanos() as f64;
    assert!(switches >= 2 * n as u64);
    elapsed / switches as f64
}

fn sync_import_cost(n: i32, async_store: bool) -> f64 {
    let engine = Engine::new(&config(async_store)).unwrap();
    let module = Module::new(&engine, GUEST).unwrap();
    let mut linker: Linker<u64> = Linker::new(&engine);
    linker
        .func_wrap("env", "hop", |mut caller: wasmtime::Caller<'_, u64>| {
            *caller.data_mut() += 1;
        })
        .unwrap();
    let pre = linker.instantiate_pre(&module).unwrap();
    let mut store = Store::new(&engine, 0u64);
    store.set_fuel(u64::MAX).unwrap();
    if async_store {
        let waker = noop_waker();
        let mut context = Context::from_waker(&waker);
        let mut future = Box::pin(async {
            let instance = pre.instantiate_async(&mut store).await.unwrap();
            let run = instance.get_typed_func::<i32, ()>(&mut store, "run").unwrap();
            let start = Instant::now();
            run.call_async(&mut store, n).await.unwrap();
            start.elapsed().as_nanos() as f64 / n as f64
        });
        loop {
            if let Poll::Ready(value) = future.as_mut().poll(&mut context) {
                return value;
            }
        }
    }
    let instance = pre.instantiate(&mut store).unwrap();
    let run = instance.get_typed_func::<i32, ()>(&mut store, "run").unwrap();
    let start = Instant::now();
    run.call(&mut store, n).unwrap();
    let elapsed = start.elapsed().as_nanos() as f64 / n as f64;
    assert_eq!(*store.data(), n as u64);
    elapsed
}

fn async_import_no_yield_cost(n: i32) -> f64 {
    let engine = Engine::new(&config(true)).unwrap();
    let module = Module::new(&engine, GUEST).unwrap();
    let mut linker: Linker<u64> = Linker::new(&engine);
    linker
        .func_wrap_async("env", "hop", |_caller, ()| Box::new(YieldOnce(true)))
        .unwrap();
    let pre = linker.instantiate_pre(&module).unwrap();
    let mut store = Store::new(&engine, 0u64);
    store.set_fuel(u64::MAX).unwrap();
    let waker = noop_waker();
    let mut context = Context::from_waker(&waker);
    let mut future = Box::pin(async {
        let instance = pre.instantiate_async(&mut store).await.unwrap();
        let run = instance.get_typed_func::<i32, ()>(&mut store, "run").unwrap();
        let start = Instant::now();
        run.call_async(&mut store, n).await.unwrap();
        start.elapsed().as_nanos() as f64 / n as f64
    });
    loop {
        if let Poll::Ready(value) = future.as_mut().poll(&mut context) {
            return value;
        }
    }
}

fn median(mut values: Vec<f64>) -> f64 {
    values.sort_by(|a, b| a.partial_cmp(b).unwrap());
    values[values.len() / 2]
}

fn main() {
    let n: i32 = std::env::args()
        .nth(1)
        .map(|value| value.parse().unwrap())
        .unwrap_or(200_000);
    let rounds: usize = std::env::args()
        .nth(2)
        .map(|value| value.parse().unwrap())
        .unwrap_or(7);
    let mut condvar = Vec::new();
    let mut park = Vec::new();
    let mut fiber = Vec::new();
    let mut sync_plain = Vec::new();
    let mut sync_fiber = Vec::new();
    let mut async_ready = Vec::new();
    // Interleave the variants so host load drifts affect all of them alike.
    for _ in 0..rounds {
        condvar.push(threads_condvar(n));
        park.push(threads_park(n));
        fiber.push(fibers(n));
        sync_plain.push(sync_import_cost(n, false));
        sync_fiber.push(sync_import_cost(n, true));
        async_ready.push(async_import_no_yield_cost(n));
    }
    println!(
        "{{\"calls_per_store\":{n},\"rounds\":{rounds},\
\"a_threads_condvar_ns_per_switch\":{:.1},\
\"a_threads_park_ns_per_switch\":{:.1},\
\"b_fibers_ns_per_switch\":{:.1},\
\"sync_typed_import_sync_store_ns_per_call\":{:.1},\
\"sync_typed_import_async_store_ns_per_call\":{:.1},\
\"async_import_no_yield_ns_per_call\":{:.1},\
\"raw\":{{\"a_condvar\":{:?},\"a_park\":{:?},\"b_fibers\":{:?},\"sync_plain\":{:?},\"sync_fiber\":{:?},\"async_ready\":{:?}}}}}",
        median(condvar.clone()),
        median(park.clone()),
        median(fiber.clone()),
        median(sync_plain.clone()),
        median(sync_fiber.clone()),
        median(async_ready.clone()),
        condvar,
        park,
        fiber,
        sync_plain,
        sync_fiber,
        async_ready
    );
}
