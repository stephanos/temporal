use std::alloc::{GlobalAlloc, Layout, System};
use std::sync::atomic::{AtomicBool, AtomicU64, Ordering};

struct MeasuredAllocator;

#[global_allocator]
static ALLOCATOR: MeasuredAllocator = MeasuredAllocator;
static ACTIVE: AtomicBool = AtomicBool::new(false);
static ALLOCATED: AtomicU64 = AtomicU64::new(0);
static ALLOCATIONS: AtomicU64 = AtomicU64::new(0);
static LIVE: AtomicU64 = AtomicU64::new(0);
static PEAK: AtomicU64 = AtomicU64::new(0);

fn allocated(bytes: usize) {
    let live = LIVE.fetch_add(bytes as u64, Ordering::Relaxed) + bytes as u64;
    if ACTIVE.load(Ordering::Relaxed) {
        ALLOCATED.fetch_add(bytes as u64, Ordering::Relaxed);
        ALLOCATIONS.fetch_add(1, Ordering::Relaxed);
        PEAK.fetch_max(live, Ordering::Relaxed);
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
    assert!(!ACTIVE.load(Ordering::Relaxed), "nested allocation capture");
    ALLOCATED.store(0, Ordering::Relaxed);
    ALLOCATIONS.store(0, Ordering::Relaxed);
    let before = LIVE.load(Ordering::Relaxed);
    PEAK.store(before, Ordering::Relaxed);
    struct Guard;
    impl Drop for Guard {
        fn drop(&mut self) {
            ACTIVE.store(false, Ordering::Relaxed);
        }
    }
    ACTIVE.store(true, Ordering::Relaxed);
    let guard = Guard;
    let result = std::hint::black_box(operation());
    drop(guard);
    let facts = Facts {
        allocated_bytes: ALLOCATED.load(Ordering::Relaxed),
        allocation_calls: ALLOCATIONS.load(Ordering::Relaxed),
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
    println!(
        "GOMAD_ALLOCATION_OBSERVATION {}",
        serde_json::json!({"schema":"gomad.native-allocation-observation/v1", "case":case,
            "inputs":inputs,"allocator":facts,"observed":observed})
    );
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
