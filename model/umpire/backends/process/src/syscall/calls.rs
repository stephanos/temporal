//! Child-only calls that bypass pthread cancellation machinery.

use std::{io, os::fd::RawFd, ptr};

#[cfg(all(target_os = "linux", target_env = "gnu"))]
type Resource = libc::__rlimit_resource_t;
#[cfg(not(all(target_os = "linux", target_env = "gnu")))]
type Resource = libc::c_int;

type Setpgid = unsafe extern "C" fn(libc::pid_t, libc::pid_t) -> libc::c_int;
type Sigaction =
    unsafe extern "C" fn(libc::c_int, *const libc::sigaction, *mut libc::sigaction) -> libc::c_int;
type Dup2 = unsafe extern "C" fn(RawFd, RawFd) -> libc::c_int;
type Kill = unsafe extern "C" fn(libc::pid_t, libc::c_int) -> libc::c_int;
type Setsid = unsafe extern "C" fn() -> libc::pid_t;
type Chdir = unsafe extern "C" fn(*const libc::c_char) -> libc::c_int;
type Execve = unsafe extern "C" fn(
    *const libc::c_char,
    *const *const libc::c_char,
    *const *const libc::c_char,
) -> libc::c_int;
type Setrlimit = unsafe extern "C" fn(Resource, *const libc::rlimit) -> libc::c_int;
type Exit = unsafe extern "C" fn(libc::c_int) -> !;
#[cfg(target_os = "linux")]
type RawSyscall = unsafe extern "C" fn(libc::c_long, ...) -> libc::c_long;
#[cfg(target_os = "linux")]
type Prctl = unsafe extern "C" fn(libc::c_int, ...) -> libc::c_int;
#[cfg(target_os = "linux")]
type Lseek = unsafe extern "C" fn(RawFd, libc::off_t, libc::c_int) -> libc::off_t;
#[cfg(target_os = "macos")]
type Read = unsafe extern "C" fn(RawFd, *mut libc::c_void, usize) -> isize;
#[cfg(target_os = "macos")]
type Write = unsafe extern "C" fn(RawFd, *const libc::c_void, usize) -> isize;
#[cfg(target_os = "macos")]
type Close = unsafe extern "C" fn(RawFd) -> libc::c_int;
#[cfg(target_os = "macos")]
type Poll = unsafe extern "C" fn(*mut libc::pollfd, libc::nfds_t, libc::c_int) -> libc::c_int;
#[cfg(target_os = "macos")]
type Wait4 = unsafe extern "C" fn(
    libc::pid_t,
    *mut libc::c_int,
    libc::c_int,
    *mut libc::rusage,
) -> libc::pid_t;
#[cfg(target_os = "macos")]
type Sigprocmask =
    unsafe extern "C" fn(libc::c_int, *const libc::sigset_t, *mut libc::sigset_t) -> libc::c_int;

#[cfg(target_os = "macos")]
type CancelState = unsafe extern "C" fn(libc::c_int) -> libc::c_int;

// Resolution occurs in the parent. Children only load existing function pointers
// and the existing errno cell, even when their consumer linked with lazy binding.
macro_rules! resolve {
    ($name:expr, $direct:path, $kind:ty) => {{
        #[cfg(target_feature = "crt-static")]
        {
            $direct as $kind
        }
        #[cfg(not(target_feature = "crt-static"))]
        {
            let address = libc::dlsym(libc::RTLD_DEFAULT, $name.as_ptr());
            if address.is_null() {
                return Err(io::Error::other(format!(
                    "cannot resolve backend syscall {}",
                    $name.to_string_lossy()
                )));
            }
            std::mem::transmute::<*mut libc::c_void, $kind>(address)
        }
    }};
}

pub(super) struct Api {
    pub(super) errno: *mut libc::c_int,
    setpgid: Setpgid,
    sigaction: Sigaction,
    dup2: Dup2,
    kill: Kill,
    setsid: Setsid,
    chdir: Chdir,
    execve: Execve,
    setrlimit: Setrlimit,
    exit: Exit,
    #[cfg(target_os = "linux")]
    syscall: RawSyscall,
    #[cfg(target_os = "linux")]
    prctl: Prctl,
    #[cfg(target_os = "linux")]
    lseek: Lseek,
    #[cfg(target_os = "macos")]
    fork: Setsid,
    #[cfg(target_os = "macos")]
    cancel_state: CancelState,
    #[cfg(target_os = "macos")]
    read: Read,
    #[cfg(target_os = "macos")]
    write: Write,
    #[cfg(target_os = "macos")]
    close: Close,
    #[cfg(target_os = "macos")]
    poll: Poll,
    #[cfg(target_os = "macos")]
    wait4: Wait4,
    #[cfg(target_os = "macos")]
    sigprocmask: Sigprocmask,
}

impl Api {
    pub(super) fn new() -> io::Result<Self> {
        // SAFETY: every symbol has the ABI documented by its libc declaration or
        // Apple syscall source; none of the system libraries can be unloaded.
        unsafe {
            let api = Self {
                setpgid: resolve!(c"setpgid", libc::setpgid, Setpgid),
                sigaction: resolve!(c"sigaction", libc::sigaction, Sigaction),
                dup2: resolve!(c"dup2", libc::dup2, Dup2),
                kill: resolve!(c"kill", libc::kill, Kill),
                setsid: resolve!(c"setsid", libc::setsid, Setsid),
                chdir: resolve!(c"chdir", libc::chdir, Chdir),
                #[cfg(target_os = "linux")]
                execve: resolve!(c"execve", libc::execve, Execve),
                #[cfg(target_os = "macos")]
                execve: resolve!(c"__execve", darwin_execve, Execve),
                #[cfg(target_os = "linux")]
                setrlimit: resolve!(c"setrlimit", libc::setrlimit, Setrlimit),
                #[cfg(target_os = "macos")]
                setrlimit: resolve!(c"__setrlimit", darwin_setrlimit, Setrlimit),
                exit: resolve!(c"_exit", libc::_exit, Exit),
                #[cfg(target_os = "linux")]
                syscall: resolve!(c"syscall", libc::syscall, RawSyscall),
                #[cfg(target_os = "linux")]
                prctl: resolve!(c"prctl", libc::prctl, Prctl),
                #[cfg(target_os = "linux")]
                lseek: resolve!(c"lseek", libc::lseek, Lseek),
                #[cfg(target_os = "macos")]
                fork: resolve!(c"__fork", darwin_fork, Setsid),
                #[cfg(target_os = "macos")]
                cancel_state: resolve!(c"__pthread_canceled", darwin_cancel_state, CancelState),
                #[cfg(target_os = "macos")]
                read: resolve!(c"read$NOCANCEL", darwin_read, Read),
                #[cfg(target_os = "macos")]
                write: resolve!(c"write$NOCANCEL", darwin_write, Write),
                #[cfg(target_os = "macos")]
                close: resolve!(c"close$NOCANCEL", darwin_close, Close),
                #[cfg(target_os = "macos")]
                poll: resolve!(c"poll$NOCANCEL", darwin_poll, Poll),
                #[cfg(target_os = "macos")]
                wait4: resolve!(c"__wait4_nocancel", darwin_wait4, Wait4),
                #[cfg(target_os = "macos")]
                sigprocmask: resolve!(c"sigprocmask", libc::sigprocmask, Sigprocmask),
                #[cfg(target_os = "linux")]
                errno: libc::__errno_location(),
                #[cfg(target_os = "macos")]
                errno: libc::__error(),
            };
            // Harmless queries bind the public signal wrapper's transitive
            // conversion and the common errno path before either fork.
            let mut prior = std::mem::zeroed::<libc::sigaction>();
            if api.sigaction(libc::SIGTERM, ptr::null(), &raw mut prior) != 0 {
                return Err(io::Error::last_os_error());
            }
            let _ = api.setpgid(-1, -1);
            #[cfg(target_os = "macos")]
            if api.kill(libc::getpid(), 0) != 0 {
                return Err(io::Error::last_os_error());
            }
            #[cfg(target_os = "macos")]
            {
                // __fork's rare EINTR path uses cerror. Bind its cancellation
                // query in the parent while cancellation is disabled; the child
                // kernel flag then keeps pthread_exit unreachable on that path.
                let set_cancel_state = resolve!(
                    c"pthread_setcancelstate",
                    darwin_set_cancel_state,
                    unsafe extern "C" fn(libc::c_int, *mut libc::c_int) -> libc::c_int
                );
                let prime_cancel_error = resolve!(
                    c"_pthread_exit_if_canceled",
                    darwin_prime_cancel_error,
                    unsafe extern "C" fn(libc::c_int)
                );
                let mut prior = 0;
                let disabled = set_cancel_state(libc::PTHREAD_CANCEL_DISABLE, &raw mut prior);
                if disabled != 0 {
                    return Err(io::Error::from_raw_os_error(disabled));
                }
                prime_cancel_error(libc::EINTR);
                let restored = set_cancel_state(prior, ptr::null_mut());
                if restored != 0 {
                    return Err(io::Error::from_raw_os_error(restored));
                }
            }
            Ok(api)
        }
    }
}

macro_rules! entries {
    ($($name:ident($($argument:ident: $kind:ty),*) -> $result:ty;)*) => {
        impl Api {
            $(pub(super) unsafe fn $name(&self, $($argument: $kind),*) -> $result {
                // SAFETY: the parent resolved the declared ABI and owns the table;
                // volatile loading also prevents direct-call devirtualization.
                unsafe { (ptr::read_volatile(&raw const self.$name))($($argument),*) }
            })*
        }
    };
}
entries! {
    setpgid(pid: libc::pid_t, group: libc::pid_t) -> libc::c_int;
    sigaction(signal: libc::c_int, action: *const libc::sigaction, previous: *mut libc::sigaction) -> libc::c_int;
    dup2(source: RawFd, target: RawFd) -> libc::c_int;
    kill(pid: libc::pid_t, signal: libc::c_int) -> libc::c_int;
    setsid() -> libc::pid_t;
    chdir(directory: *const libc::c_char) -> libc::c_int;
    execve(program: *const libc::c_char, argv: *const *const libc::c_char, envp: *const *const libc::c_char) -> libc::c_int;
    setrlimit(resource: Resource, limit: *const libc::rlimit) -> libc::c_int;
    exit(status: libc::c_int) -> !;
}
#[cfg(target_os = "linux")]
entries! {
    lseek(fd: RawFd, offset: libc::off_t, whence: libc::c_int) -> libc::off_t;
}
#[cfg(target_os = "macos")]
entries! {
    fork() -> libc::pid_t;
    cancel_state(action: libc::c_int) -> libc::c_int;
    sigprocmask(operation: libc::c_int, set: *const libc::sigset_t, previous: *mut libc::sigset_t) -> libc::c_int;
}
impl Api {
    #[cfg(target_os = "linux")]
    pub(super) unsafe fn syscall(&self) -> RawSyscall {
        unsafe { ptr::read_volatile(&raw const self.syscall) }
    }
    #[cfg(target_os = "linux")]
    pub(super) unsafe fn subreaper(&self) -> bool {
        unsafe {
            (ptr::read_volatile(&raw const self.prctl))(
                libc::PR_SET_CHILD_SUBREAPER,
                1 as libc::c_ulong,
                0 as libc::c_ulong,
                0 as libc::c_ulong,
                0 as libc::c_ulong,
            ) == 0
        }
    }
}

#[cfg(target_os = "linux")]
#[repr(C)]
struct KernelTimespec {
    seconds: KernelTime,
    nanoseconds: KernelTime,
}

// Native 64-bit and x32 syscalls use two 64-bit slots. The RISC-V 32-bit
// interface has only time64 syscalls. Other 32-bit legacy syscall numbers
// select the kernel's time32 layout, including MIPS n32.
#[cfg(all(
    target_os = "linux",
    any(target_pointer_width = "64", target_arch = "x86_64", target_arch = "riscv32")
))]
type KernelTime = i64;
#[cfg(all(
    target_os = "linux",
    not(any(target_pointer_width = "64", target_arch = "x86_64", target_arch = "riscv32"))
))]
type KernelTime = libc::c_long;

#[cfg(all(target_os = "linux", target_arch = "riscv32"))]
const PPOLL: libc::c_long = libc::SYS_ppoll_time64;
#[cfg(all(target_os = "linux", not(target_arch = "riscv32")))]
const PPOLL: libc::c_long = libc::SYS_ppoll;
#[cfg(all(target_os = "linux", target_arch = "riscv32"))]
const CLOCK_NANOSLEEP: libc::c_long = libc::SYS_clock_nanosleep_time64;
#[cfg(all(target_os = "linux", not(target_arch = "riscv32")))]
const CLOCK_NANOSLEEP: libc::c_long = libc::SYS_clock_nanosleep;

#[allow(clippy::cast_possible_truncation, reason = "kernel byte counts are ssize_t values or -1")]
pub(super) unsafe fn read(api: &Api, fd: RawFd, bytes: *mut libc::c_void, length: usize) -> isize {
    #[cfg(target_os = "linux")]
    unsafe {
        (api.syscall())(libc::SYS_read, fd, bytes, length) as isize
    }
    #[cfg(target_os = "macos")]
    unsafe {
        (ptr::read_volatile(&raw const api.read))(fd, bytes, length)
    }
}

#[allow(clippy::cast_possible_truncation, reason = "kernel byte counts are ssize_t values or -1")]
pub(super) unsafe fn write(
    api: &Api,
    fd: RawFd,
    bytes: *const libc::c_void,
    length: usize,
) -> isize {
    #[cfg(target_os = "linux")]
    unsafe {
        (api.syscall())(libc::SYS_write, fd, bytes, length) as isize
    }
    #[cfg(target_os = "macos")]
    unsafe {
        (ptr::read_volatile(&raw const api.write))(fd, bytes, length)
    }
}

pub(super) unsafe fn close(api: &Api, fd: RawFd) {
    #[cfg(target_os = "linux")]
    unsafe {
        (api.syscall())(libc::SYS_close, fd);
    }
    #[cfg(target_os = "macos")]
    unsafe {
        (ptr::read_volatile(&raw const api.close))(fd);
    }
}

#[allow(clippy::cast_possible_truncation)]
pub(super) unsafe fn waitpid(
    api: &Api,
    pid: libc::pid_t,
    status: *mut libc::c_int,
    options: libc::c_int,
) -> libc::pid_t {
    #[cfg(target_os = "linux")]
    unsafe {
        // The kernel result is a pid_t or -1, so narrowing cannot lose bits.
        (api.syscall())(libc::SYS_wait4, pid, status, options, ptr::null_mut::<libc::c_void>())
            as libc::pid_t
    }
    #[cfg(target_os = "macos")]
    unsafe {
        (ptr::read_volatile(&raw const api.wait4))(pid, status, options, ptr::null_mut())
    }
}

pub(super) unsafe fn poll_owner(api: &Api, owner: *mut libc::pollfd) -> libc::c_long {
    #[cfg(target_os = "linux")]
    unsafe {
        let mut timeout = KernelTimespec { seconds: 0, nanoseconds: 5_000_000 };
        (api.syscall())(
            PPOLL,
            owner,
            1 as libc::c_ulong,
            &raw mut timeout,
            ptr::null::<libc::c_void>(),
            0 as libc::c_ulong,
        )
    }
    #[cfg(target_os = "macos")]
    unsafe {
        libc::c_long::from((ptr::read_volatile(&raw const api.poll))(owner, 1, 5))
    }
}

#[cfg(target_os = "linux")]
#[allow(clippy::cast_possible_truncation)]
pub(super) unsafe fn open_children(api: &Api) -> RawFd {
    // The kernel result is a descriptor or -1, both representable by RawFd.
    unsafe {
        (api.syscall())(
            libc::SYS_openat,
            libc::AT_FDCWD,
            c"/proc/thread-self/children".as_ptr(),
            libc::O_RDONLY | libc::O_CLOEXEC,
            0 as libc::mode_t,
        ) as RawFd
    }
}

#[cfg(target_os = "linux")]
pub(super) unsafe fn pause(api: &Api) {
    unsafe {
        let delay = KernelTimespec { seconds: 0, nanoseconds: 5_000_000 };
        (api.syscall())(
            CLOCK_NANOSLEEP,
            libc::CLOCK_MONOTONIC,
            0,
            &raw const delay,
            ptr::null_mut::<libc::c_void>(),
        );
    }
}

#[cfg(target_os = "macos")]
unsafe extern "C" {
    #[link_name = "pthread_setcancelstate"]
    fn darwin_set_cancel_state(state: libc::c_int, previous: *mut libc::c_int) -> libc::c_int;
    #[link_name = "_pthread_exit_if_canceled"]
    fn darwin_prime_cancel_error(error: libc::c_int);
    #[link_name = "__pthread_canceled"]
    fn darwin_cancel_state(action: libc::c_int) -> libc::c_int;
    #[link_name = "__fork"]
    fn darwin_fork() -> libc::pid_t;
    #[link_name = "__execve"]
    fn darwin_execve(
        program: *const libc::c_char,
        argv: *const *const libc::c_char,
        envp: *const *const libc::c_char,
    ) -> libc::c_int;
    #[link_name = "__setrlimit"]
    fn darwin_setrlimit(resource: Resource, limit: *const libc::rlimit) -> libc::c_int;
    // Apple's MacOSX syscall maps export these aliases to their non-cancelling
    // kernel stubs. They bypass cerror's pthread-exit branch, including EINTR.
    #[link_name = "read$NOCANCEL"]
    fn darwin_read(fd: RawFd, bytes: *mut libc::c_void, length: usize) -> isize;
    #[link_name = "write$NOCANCEL"]
    fn darwin_write(fd: RawFd, bytes: *const libc::c_void, length: usize) -> isize;
    #[link_name = "close$NOCANCEL"]
    fn darwin_close(fd: RawFd) -> libc::c_int;
    #[link_name = "poll$NOCANCEL"]
    fn darwin_poll(
        fds: *mut libc::pollfd,
        count: libc::nfds_t,
        timeout: libc::c_int,
    ) -> libc::c_int;
    #[link_name = "__wait4_nocancel"]
    fn darwin_wait4(
        pid: libc::pid_t,
        status: *mut libc::c_int,
        options: libc::c_int,
        usage: *mut libc::rusage,
    ) -> libc::pid_t;
}
