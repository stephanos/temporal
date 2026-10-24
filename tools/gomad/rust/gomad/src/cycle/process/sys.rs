//! Linux syscall-only custody supervisor. No allocator, unwinding, locks or Rust
//! destructors may run between fork and exec/_exit in either child.
use std::ffi::{c_char, c_int, c_long, c_void};

#[repr(C)]
#[derive(Clone, Copy, Default)]
pub(super) struct Timeval {
    pub seconds: c_long,
    pub micros: c_long,
}
#[repr(C)]
#[derive(Clone, Copy, Default)]
pub(super) struct Usage {
    pub user: Timeval,
    pub system: Timeval,
    pub counters: [c_long; 14],
}
#[repr(C)]
#[derive(Clone, Copy, Default)]
pub(super) struct Report {
    pub pid: i64,
    pub waited_pid: i64,
    pub status: i64,
    pub started: i64,
    pub reaped: i64,
    pub cleaned: i64,
    pub usage: Usage,
    pub descendants: i64,
    pub cleanup_error: i64,
    pub reason: i64,
    pub reaps: i64,
    pub exec_error: i64,
}
#[repr(C)]
struct Poll {
    fd: c_int,
    events: i16,
    revents: i16,
}
#[repr(C)]
#[derive(Default)]
struct Timespec {
    seconds: c_long,
    nanos: c_long,
}
unsafe extern "C" {
    #[link_name = "_Fork"]
    pub(super) fn fork() -> c_int;
    fn _exit(status: c_int) -> !;
    fn setsid() -> c_int;
    fn setpgid(pid: c_int, group: c_int) -> c_int;
    fn prctl(option: c_int, arg: c_long, ...) -> c_int;
    fn dup2(old: c_int, new: c_int) -> c_int;
    fn fcntl(fd: c_int, cmd: c_int, ...) -> c_int;
    fn close(fd: c_int) -> c_int;
    fn close_range(first: u32, last: u32, flags: c_int) -> c_int;
    fn read(fd: c_int, buffer: *mut c_void, count: usize) -> isize;
    fn write(fd: c_int, buffer: *const c_void, count: usize) -> isize;
    fn open(path: *const c_char, flags: c_int, ...) -> c_int;
    fn chdir(path: *const c_char) -> c_int;
    fn execve(path: *const c_char, argv: *const *const c_char, env: *const *const c_char) -> c_int;
    fn poll(fds: *mut Poll, count: usize, timeout: c_int) -> c_int;
    pub(super) fn wait4(pid: c_int, status: *mut c_int, flags: c_int, usage: *mut Usage) -> c_int;
    fn kill(pid: c_int, signal: c_int) -> c_int;
    fn signal(number: c_int, handler: usize) -> usize;
    fn sigprocmask(how: c_int, set: *const u64, old: *mut u64) -> c_int;
    fn clock_gettime(clock: c_int, value: *mut Timespec) -> c_int;
    fn __errno_location() -> *mut c_int;
}
pub(super) struct Prepared<'a> {
    pub fds: [c_int; 7],
    pub argv: &'a [*const c_char],
    pub env: &'a [*const c_char],
    pub executable: *const c_char,
    pub cwd: *const c_char,
    pub timeout_ns: i64,
}
fn now() -> i64 {
    let mut time = Timespec::default();
    // SAFETY: CLOCK_MONOTONIC writes the supported Linux 64-bit timespec ABI.
    if unsafe { clock_gettime(1, &raw mut time) } != 0 {
        return -1;
    }
    time.seconds
        .saturating_mul(1_000_000_000)
        .saturating_add(time.nanos)
}
fn errno() -> c_int {
    // SAFETY: libc returns this thread's valid errno pointer.
    unsafe { *__errno_location() }
}
fn send(fd: c_int, bytes: &[u8]) -> bool {
    let mut offset = 0;
    while offset < bytes.len() {
        // SAFETY: offset is bounded by bytes and the buffer remains live.
        let count = unsafe { write(fd, bytes.as_ptr().add(offset).cast(), bytes.len() - offset) };
        if count < 0 && errno() == 4 {
            continue;
        }
        let Ok(count) = usize::try_from(count) else {
            return false;
        };
        if count == 0 {
            return false;
        }
        offset += count;
    }
    true
}
fn wire(report: &Report) -> &[u8] {
    // SAFETY: Report has only initialized, padding-free 64-bit integer fields.
    unsafe { std::slice::from_raw_parts(std::ptr::from_ref(report).cast(), size_of::<Report>()) }
}
fn ready(fd: c_int, timeout: c_int) -> c_int {
    let mut item = Poll {
        fd,
        events: 1,
        revents: 0,
    };
    // SAFETY: one initialized pollfd; poll does not retain its pointer.
    unsafe { poll(&raw mut item, 1, timeout) }
}
fn child(prepared: &Prepared<'_>) -> ! {
    let mut admitted = 0u8;
    // SAFETY: only async-signal-safe C calls on owned, relocated descriptors;
    // all argv/env strings and pointers were allocated before the first fork.
    unsafe {
        close(4);
        close(8);
        if fcntl(9, 2, 1) != 0 {
            exec_failure(126);
        }
        if setpgid(0, 0) != 0
            || ready(3, 30_000) <= 0
            || read(3, (&raw mut admitted).cast(), 1) != 1
            || admitted != b'1'
        {
            _exit(130);
        }
        close(3);
        let empty = [0u64; 16];
        if sigprocmask(2, empty.as_ptr(), std::ptr::null_mut()) != 0 {
            exec_failure(126);
        }
        for number in [1, 2, 13, 15] {
            signal(number, 0);
        }
        if chdir(prepared.cwd) != 0 {
            exec_failure(126);
        }
        execve(
            prepared.executable,
            prepared.argv.as_ptr(),
            prepared.env.as_ptr(),
        );
        exec_failure(127);
    }
}
fn exec_failure(code: c_int) -> ! {
    let failure = errno();
    send(9, &failure.to_ne_bytes());
    // SAFETY: no destructors or allocator run in the raw-fork leaf.
    unsafe {
        _exit(code);
    }
}
fn kill_children(buffer: &mut [u8; 65536]) -> Result<usize, c_int> {
    // SAFETY: constant NUL-terminated procfs task path; open/read/close are
    // async-signal-safe and the writable fixed buffer is not shared.
    let fd = unsafe { open(c"/proc/thread-self/children".as_ptr(), 0x80000) };
    if fd < 0 {
        return Err(errno());
    }
    // SAFETY: buffer spans exactly its declared writable allocation.
    let length = unsafe { read(fd, buffer.as_mut_ptr().cast(), buffer.len()) };
    // SAFETY: closes only this function's descriptor.
    unsafe {
        close(fd);
    }
    let Ok(length) = usize::try_from(length) else {
        return Err(errno());
    };
    if length == buffer.len() {
        return Err(75);
    }
    let mut pid = 0i32;
    let mut count = 0;
    for &byte in buffer.iter().take(length).chain(std::iter::once(&b' ')) {
        if byte.is_ascii_digit() {
            let Some(next) = pid
                .checked_mul(10)
                .and_then(|value| value.checked_add(i32::from(byte - b'0')))
            else {
                return Err(75);
            };
            pid = next;
        } else if byte == b' ' || byte == b'\n' {
            if pid > 0 {
                // SAFETY: positive PID came from this supervisor's kernel-owned
                // child list. Its unreaped child identity cannot be reused.
                unsafe {
                    kill(pid, 9);
                }
                count += 1;
                pid = 0;
            }
        } else {
            return Err(22);
        }
    }
    Ok(count)
}
#[expect(
    clippy::large_stack_arrays,
    clippy::too_many_lines,
    reason = "The raw-fork supervisor cannot allocate; its bounded child list uses a fixed stack buffer"
)]
pub(super) fn supervise(prepared: &Prepared<'_>) -> ! {
    let mut report = Report::default();
    let mut buffer = [0u8; 65536];
    // SAFETY: each source is a preduplicated owned fd >=10, so relocation cannot
    // overwrite another source. All unrelated copies close before ready is sent.
    unsafe {
        for (source, target) in prepared.fds.iter().zip(3..10) {
            if dup2(*source, target) != target {
                _exit(125);
            }
        }
        if dup2(7, 0) != 0
            || dup2(5, 1) != 1
            || dup2(6, 2) != 2
            || close_range(10, u32::MAX, 0) != 0
        {
            _exit(125);
        }
        close(5);
        close(6);
        close(7);
        for number in [1, 2, 13, 15] {
            signal(number, 1);
        }
        signal(17, 0);
        let empty = [0u64; 16];
        if sigprocmask(2, empty.as_ptr(), std::ptr::null_mut()) != 0
            || setsid() < 0
            || prctl(36, 1 as c_long, 0 as c_long, 0 as c_long, 0 as c_long) != 0
        {
            _exit(125);
        }
    }
    report.started = now();
    // SAFETY: second fork inherits only supervisor-owned fds and preallocated
    // immutable memory. Both branches remain syscall-only until exec/_exit.
    let leaf = unsafe { fork() };
    if leaf == 0 {
        child(prepared);
    }
    report.pid = i64::from(leaf);
    // SAFETY: supervisor no longer reads the admission pipe.
    unsafe {
        close(3);
        close(9);
    }
    if leaf < 0
        || !send(4, &i64::from(leaf).to_ne_bytes())
        || !send(4, &report.started.to_ne_bytes())
    {
        report.reason = 3;
    }
    let deadline = report.started.saturating_add(prepared.timeout_ns);
    while report.reaps == 0 && leaf > 0 {
        let mut status = 0;
        let mut usage = Usage::default();
        // SAFETY: this exact positive PID is the supervisor's unreaped leaf;
        // wait4 writes the Linux 64-bit status/rusage allocations.
        let waited = unsafe { wait4(leaf, &raw mut status, 1, &raw mut usage) };
        if waited == leaf {
            report.waited_pid = i64::from(waited);
            report.status = i64::from(status);
            report.usage = usage;
            report.reaped = now();
            report.reaps = 1;
            break;
        }
        if waited < 0 && errno() != 4 {
            report.reason = 4;
            break;
        }
        if now() >= deadline && report.reason == 0 {
            report.reason = 1;
        }
        if ready(4, 5) > 0 && report.reason == 0 {
            report.reason = 2;
        }
        if report.reason != 0 {
            // SAFETY: the exact unreaped leaf and its group are supervisor-owned.
            unsafe {
                kill(leaf, 9);
                kill(-leaf, 9);
            }
        }
    }
    let cleanup_deadline = now().saturating_add(5_000_000_000);
    loop {
        if kill_children(&mut buffer).is_err() {
            report.cleanup_error = 1;
        }
        let mut status = 0;
        let mut usage = Usage::default();
        // SAFETY: this isolated supervisor owns every child it can wait for;
        // adopted descendants include changed sessions and double forks.
        let waited = unsafe { wait4(-1, &raw mut status, 1, &raw mut usage) };
        if waited > 0 {
            if waited == leaf && report.reaps == 0 {
                report.waited_pid = i64::from(waited);
                report.status = i64::from(status);
                report.usage = usage;
                report.reaped = now();
                report.reaps = 1;
            } else {
                report.descendants += 1;
            }
        } else if waited < 0 && errno() == 10 {
            break;
        } else if waited < 0 && errno() != 4 {
            report.cleanup_error = 2;
            break;
        }
        if cleanup_budget(now(), cleanup_deadline).is_err() {
            report.cleanup_error = 3;
            break;
        }
        if waited == 0 {
            ready(4, 5);
        }
    }
    let mut exec_error = [0u8; 4];
    // SAFETY: leaf exec closes the writer; failed exec writes one errno.
    let count = unsafe { read(8, exec_error.as_mut_ptr().cast(), 4) };
    if count == 4 {
        report.exec_error = i64::from(c_int::from_ne_bytes(exec_error));
    } else if count != 0 {
        report.cleanup_error = 4;
    }
    report.cleaned = now();
    send(4, wire(&report));
    // SAFETY: no Rust destructors run in the raw-fork supervisor.
    unsafe {
        _exit(if report.cleanup_error == 0 { 0 } else { 125 });
    }
}

pub(super) fn cleanup_budget(observed: i64, deadline: i64) -> Result<(), c_int> {
    if observed < 0 || observed >= deadline {
        Err(3)
    } else {
        Ok(())
    }
}
pub(super) fn reap_supervisor(pid: i32) -> Result<(), String> {
    loop {
        let mut status = 0;
        let mut usage = Usage::default();
        // SAFETY: caller owns this exact supervisor PID and reaps it once.
        let waited = unsafe { wait4(pid, &raw mut status, 0, &raw mut usage) };
        if waited == pid {
            return Ok(());
        }
        let error = std::io::Error::last_os_error();
        if error.kind() != std::io::ErrorKind::Interrupted {
            return Err(error.to_string());
        }
    }
}
pub(super) fn decode(bytes: &[u8]) -> Report {
    assert_eq!(bytes.len(), size_of::<Report>());
    // SAFETY: all bit patterns are valid in the padding-free integer report;
    // read_unaligned does not require the byte buffer to align to Report.
    unsafe { std::ptr::read_unaligned(bytes.as_ptr().cast()) }
}
