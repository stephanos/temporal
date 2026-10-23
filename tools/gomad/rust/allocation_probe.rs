use std::alloc::{GlobalAlloc, Layout, System};
use std::cell::Cell;
use std::sync::atomic::{AtomicBool, AtomicU64, Ordering};

struct MeasuredAllocator;

#[global_allocator]
static ALLOCATOR: MeasuredAllocator = MeasuredAllocator;
static ACTIVE: AtomicBool = AtomicBool::new(false);
#[derive(Clone, Copy)]
struct Scope {
    active: bool,
    allocated_bytes: u64,
    allocation_calls: u64,
}

thread_local! {
    static SCOPE: Cell<Scope> = const { Cell::new(Scope {
        active: false,
        allocated_bytes: 0,
        allocation_calls: 0,
    }) };
}
static LIVE: AtomicU64 = AtomicU64::new(0);
static PEAK: AtomicU64 = AtomicU64::new(0);

fn allocated(bytes: usize) {
    let live = LIVE.fetch_add(bytes as u64, Ordering::Relaxed) + bytes as u64;
    if ACTIVE.load(Ordering::Relaxed) {
        PEAK.fetch_max(live, Ordering::Relaxed);
        // TLS can be gone during thread teardown; teardown never belongs to a capture.
        SCOPE
            .try_with(|cell| {
                let mut scope = cell.get();
                if scope.active {
                    scope.allocated_bytes += bytes as u64;
                    scope.allocation_calls += 1;
                    cell.set(scope);
                }
            })
            .unwrap_or(());
    }
}

// SAFETY: Every operation delegates the unchanged pointer/layout contract to System.
unsafe impl GlobalAlloc for MeasuredAllocator {
    unsafe fn alloc(&self, layout: Layout) -> *mut u8 {
        // SAFETY: The caller supplies a valid allocation layout.
        let pointer = unsafe { System.alloc(layout) };
        if !pointer.is_null() {
            allocated(layout.size());
        }
        pointer
    }

    unsafe fn alloc_zeroed(&self, layout: Layout) -> *mut u8 {
        // SAFETY: The caller supplies a valid allocation layout.
        let pointer = unsafe { System.alloc_zeroed(layout) };
        if !pointer.is_null() {
            allocated(layout.size());
        }
        pointer
    }

    unsafe fn dealloc(&self, pointer: *mut u8, layout: Layout) {
        LIVE.fetch_sub(layout.size() as u64, Ordering::Relaxed);
        // SAFETY: The caller supplies the allocation's pointer and layout.
        unsafe { System.dealloc(pointer, layout) };
    }

    unsafe fn realloc(&self, pointer: *mut u8, layout: Layout, size: usize) -> *mut u8 {
        // SAFETY: The caller supplies the allocation's pointer/layout and valid new size.
        let next = unsafe { System.realloc(pointer, layout, size) };
        if !next.is_null() {
            LIVE.fetch_sub(layout.size() as u64, Ordering::Relaxed);
            allocated(size);
        }
        next
    }
}

#[derive(Clone, Copy, serde::Serialize)]
pub(crate) struct Facts {
    pub(crate) allocated_bytes: u64,
    pub(crate) allocation_calls: u64,
    pub(crate) live_requested_bytes_before: u64,
    pub(crate) live_requested_bytes_after: u64,
    pub(crate) peak_live_requested_bytes: u64,
}

pub(crate) fn capture<T>(operation: impl FnOnce() -> T) -> (T, Facts) {
    struct Guard;
    impl Drop for Guard {
        fn drop(&mut self) {
            ACTIVE.store(false, Ordering::Relaxed);
            SCOPE.with(|cell| {
                let mut scope = cell.get();
                scope.active = false;
                cell.set(scope);
            });
        }
    }
    assert!(!ACTIVE.load(Ordering::Relaxed), "nested allocation capture");
    SCOPE.with(|cell| {
        assert!(!cell.get().active, "nested thread allocation capture");
        cell.set(Scope {
            active: true,
            allocated_bytes: 0,
            allocation_calls: 0,
        });
    });
    let before = LIVE.load(Ordering::Relaxed);
    PEAK.store(before, Ordering::Relaxed);
    ACTIVE.store(true, Ordering::Relaxed);
    let guard = Guard;
    let result = std::hint::black_box(operation());
    drop(guard);
    let scope = SCOPE.with(Cell::get);
    let facts = Facts {
        allocated_bytes: scope.allocated_bytes,
        allocation_calls: scope.allocation_calls,
        live_requested_bytes_before: before,
        live_requested_bytes_after: LIVE.load(Ordering::Relaxed),
        peak_live_requested_bytes: PEAK.load(Ordering::Relaxed),
    };
    (result, facts)
}

pub(crate) fn row(
    case: &str,
    inputs: serde_json::Value,
    facts: Facts,
    observed: serde_json::Value,
) {
    let value = serde_json::Value::Object(serde_json::Map::from_iter([
        (
            "schema".into(),
            serde_json::json!("gomad.native-allocation-observation/v1"),
        ),
        ("case".into(), serde_json::json!(case)),
        (
            "allocation_scope".into(),
            serde_json::json!("selected-test-thread"),
        ),
        ("live_scope".into(), serde_json::json!("whole-process")),
        ("inputs".into(), inputs),
        ("allocator".into(), serde_json::to_value(facts).unwrap()),
        ("observed".into(), observed),
    ]));
    println!("GOMAD_ALLOCATION_OBSERVATION {value}");
}

pub(crate) fn self_check() {
    let (bytes, facts) = capture(|| vec![0xa5_u8; std::hint::black_box(8192)]);
    assert_eq!(facts.allocated_bytes, 8192);
    assert_eq!(facts.allocation_calls, 1);
    assert_eq!(
        facts.live_requested_bytes_after - facts.live_requested_bytes_before,
        8192
    );
    assert_eq!(bytes[8191], 0xa5);
    row(
        "allocator-control",
        serde_json::json!({"bytes":8192}),
        facts,
        serde_json::json!({"length":bytes.len(),"last":bytes[8191]}),
    );
}
