//! The only unsafe boundary for backend process ownership.
//!
//! All owned strings, pointer tables, descriptors and parsing storage are prepared
//! before fork. The two child entry points never return to Rust: their transitive
//! callees use only libc, raw pointers and bounded scalar operations, and every
//! exit is `_exit`. In particular, no child drops parent-owned Rust values.
#![allow(unsafe_code)]

use std::{fs::File, io, path::Path, process::{Command, ExitStatus}};

fn platform_policy(platform: &str, uid: u32, euid: u32) -> io::Result<()> {
    match platform {
        "linux" => Ok(()),
        "macos" if uid != 0 && euid != 0 => Ok(()),
        "macos" => Err(io::Error::other(
            "Darwin descendant prevention requires a non-root user",
        )),
        _ => Err(io::Error::other(
            "backend process ownership supports Linux and macOS only",
        )),
    }
}

#[cfg(any(target_os = "linux", target_os = "macos"))]
mod supported {
    use super::{Command, ExitStatus, File, Path, io, platform_policy};
    use std::{
        collections::BTreeMap,
        ffi::{CString, OsStr, OsString},
        io::PipeWriter,
        os::{
            fd::{AsRawFd, FromRawFd, OwnedFd, RawFd},
            unix::{ffi::OsStrExt, process::ExitStatusExt},
        },
        ptr,
        sync::atomic::{AtomicI32, Ordering},
    };

    const BUFFER_BYTES: usize = 4096;
    #[cfg(not(target_has_atomic = "32"))]
    compile_error!("backend signal ownership requires lock-free 32-bit atomics");
    static STOP_SIGNAL: AtomicI32 = AtomicI32::new(0);

    pub(crate) struct Running {
        pid: libc::pid_t,
        owner: Option<PipeWriter>,
        status: Option<ExitStatus>,
    }

    impl Running {
        pub(crate) fn spawn(
            command: &Command,
            fallback_directory: &Path,
            out: &File,
            err: &File,
        ) -> io::Result<Self> {
            // SAFETY: identity syscalls have no pointer arguments or side effects.
            unsafe {
                platform_policy(std::env::consts::OS, libc::getuid(), libc::geteuid())?;
            }
            let mut prepared = Prepared::new(command, fallback_directory)?;
            let (owner_read, owner_write) = std::io::pipe()?;
            let null = File::open("/dev/null")?;
            let input = duplicate(owner_read.as_raw_fd())?;
            let output = duplicate(out.as_raw_fd())?;
            let errors = duplicate(err.as_raw_fd())?;
            let null = duplicate(null.as_raw_fd())?;
            let context = prepared.context(
                input.as_raw_fd(), output.as_raw_fd(), errors.as_raw_fd(), null.as_raw_fd(),
            );
            let mask = SignalMask::block()?;
            // SAFETY: all child memory and descriptors are live across fork. Only
            // the child takes child_main, which cannot return or drop these values.
            let pid = unsafe { fork_without_handlers() };
            if pid == 0 {
                // SAFETY: context points to immutable prepared tables and one
                // uniquely owned parsing buffer in this process's private memory.
                unsafe { child_main(&context, &mask.original) }
            }
            if pid < 0 {
                let error = io::Error::last_os_error();
                mask.restore()?;
                return Err(error);
            }
            let running = Self { pid, owner: Some(owner_write), status: None };
            mask.restore()?;
            Ok(running)
        }

        pub(crate) fn try_wait(&mut self) -> io::Result<Option<ExitStatus>> {
            if self.status.is_some() {
                return Ok(self.status);
            }
            loop {
                let mut status = 0;
                // SAFETY: pid is this object's unreaped child; status is writable.
                let result = unsafe { libc::waitpid(self.pid, &raw mut status, libc::WNOHANG) };
                if result == self.pid {
                    let status = ExitStatus::from_raw(status);
                    self.status = Some(status);
                    return Ok(Some(status));
                }
                if result == 0 {
                    return Ok(None);
                }
                let error = io::Error::last_os_error();
                if error.kind() != io::ErrorKind::Interrupted {
                    return Err(error);
                }
            }
        }

        #[cfg(test)]
        pub(crate) fn pid(&self) -> u32 {
            self.pid.cast_unsigned()
        }
    }

    impl Drop for Running {
        fn drop(&mut self) {
            drop(self.owner.take());
            if self.status.is_none() {
                loop {
                    // SAFETY: a null status pointer is permitted by waitpid.
                    let result = unsafe { libc::waitpid(self.pid, ptr::null_mut(), 0) };
                    if result >= 0 || io::Error::last_os_error().kind() != io::ErrorKind::Interrupted {
                        break;
                    }
                }
            }
        }
    }

    fn duplicate(fd: RawFd) -> io::Result<OwnedFd> {
        // SAFETY: fcntl returns a new uniquely owned descriptor. Keeping all
        // inputs above 9 makes the child's four dup2 operations non-overlapping.
        let duplicate = unsafe { libc::fcntl(fd, libc::F_DUPFD_CLOEXEC, 10) };
        if duplicate < 0 {
            return Err(io::Error::last_os_error());
        }
        // SAFETY: fcntl succeeded and ownership has not been assigned elsewhere.
        Ok(unsafe { OwnedFd::from_raw_fd(duplicate) })
    }

    struct Prepared {
        directory: CString,
        executable: CString,
        arguments: Vec<CString>,
        environment: Vec<CString>,
        argv: Vec<*const libc::c_char>,
        envp: Vec<*const libc::c_char>,
        buffer: Box<[u8; BUFFER_BYTES]>,
        close_limit: RawFd,
        signal_limit: libc::c_int,
        launch_error: libc::c_int,
        default_action: libc::sigaction,
        stop_action: libc::sigaction,
        all_signals: libc::sigset_t,
        empty_signals: libc::sigset_t,
    }

    impl Prepared {
        fn new(command: &Command, fallback_directory: &Path) -> io::Result<Self> {
            let directory = command.get_current_dir().unwrap_or(fallback_directory).canonicalize()?;
            let environment = environment(command)?;
            let (executable, launch_error) = executable(command.get_program(), &directory, &environment)?;
            let mut arguments = vec![cstring(command.get_program())?];
            arguments.extend(command.get_args().map(cstring).collect::<io::Result<Vec<_>>>()?);
            let environment = environment.into_iter().map(|(key, value)| {
                let mut bytes = key.as_bytes().to_vec();
                bytes.push(b'=');
                bytes.extend_from_slice(value.as_bytes());
                CString::new(bytes).map_err(io::Error::other)
            }).collect::<io::Result<Vec<_>>>()?;
            let mut argv = arguments.iter().map(|arg| arg.as_ptr()).collect::<Vec<_>>();
            argv.push(ptr::null());
            let mut envp = environment.iter().map(|arg| arg.as_ptr()).collect::<Vec<_>>();
            envp.push(ptr::null());
            let close_limit = close_limit()?;
            // SAFETY: SIGRTMAX only reads libc's available signal range.
            #[cfg(target_os = "linux")]
            let signal_limit = libc::SIGRTMAX() + 1;
            #[cfg(target_os = "macos")]
            let signal_limit = libc::NSIG;
            // SAFETY: these structures are initialized by the named libc set
            // functions before the child can read them.
            let (default_action, stop_action, all_signals, empty_signals) = unsafe {
                let mut default_action = std::mem::zeroed::<libc::sigaction>();
                let mut stop_action = std::mem::zeroed::<libc::sigaction>();
                let mut all = std::mem::zeroed::<libc::sigset_t>();
                let mut empty = std::mem::zeroed::<libc::sigset_t>();
                if libc::sigfillset(&raw mut default_action.sa_mask) != 0
                    || libc::sigfillset(&raw mut stop_action.sa_mask) != 0
                    || libc::sigfillset(&raw mut all) != 0
                    || libc::sigemptyset(&raw mut empty) != 0 {
                    return Err(io::Error::last_os_error());
                }
                default_action.sa_sigaction = libc::SIG_DFL;
                stop_action.sa_sigaction = request_stop as *const () as usize;
                (default_action, stop_action, all, empty)
            };
            Ok(Self {
                default_action, stop_action, all_signals, empty_signals,
                directory: cstring(directory.as_os_str())?, executable, arguments, environment,
                argv, envp, buffer: Box::new([0; BUFFER_BYTES]), close_limit, signal_limit,
                launch_error,
            })
        }

        fn context(&mut self, input: RawFd, output: RawFd, errors: RawFd, null: RawFd) -> ChildContext {
            // Keeping the string owners alongside the pointer tables is the
            // lifetime invariant used by both child entry points.
            let _owners = (&self.arguments, &self.environment);
            ChildContext {
                directory: self.directory.as_ptr(), executable: self.executable.as_ptr(),
                argv: self.argv.as_ptr(), envp: self.envp.as_ptr(),
                buffer: self.buffer.as_mut_ptr(), close_limit: self.close_limit,
                signal_limit: self.signal_limit, launch_error: self.launch_error,
                input, output, errors, null,
                default_action: &raw const self.default_action, stop_action: &raw const self.stop_action,
                all_signals: &raw const self.all_signals, empty_signals: &raw const self.empty_signals,
            }
        }
    }

    fn cstring(value: &OsStr) -> io::Result<CString> {
        CString::new(value.as_bytes()).map_err(io::Error::other)
    }

    fn environment(command: &Command) -> io::Result<BTreeMap<OsString, OsString>> {
        let mut environment = std::env::vars_os().collect::<BTreeMap<_, _>>();
        for (key, value) in command.get_envs() {
            if key.as_bytes().contains(&b'=') || key.as_bytes().contains(&0) {
                return Err(io::Error::new(io::ErrorKind::InvalidInput, "invalid environment key"));
            }
            if let Some(value) = value {
                environment.insert(key.to_owned(), value.to_owned());
            } else {
                environment.remove(key);
            }
        }
        Ok(environment)
    }

    fn executable(
        program: &OsStr,
        directory: &Path,
        environment: &BTreeMap<OsString, OsString>,
    ) -> io::Result<(CString, libc::c_int)> {
        if program.as_bytes().contains(&b'/') {
            return Ok((cstring(program)?, 0));
        }
        let path = environment.get(OsStr::new("PATH")).map_or(OsStr::new("/bin:/usr/bin"), |v| v.as_os_str());
        let mut failure = libc::ENOENT;
        for entry in std::env::split_paths(path) {
            let candidate = directory.join(entry).join(program);
            let candidate = cstring(candidate.as_os_str())?;
            // SAFETY: the candidate is a live NUL-terminated pathname.
            if unsafe { libc::access(candidate.as_ptr(), libc::X_OK) } == 0 {
                if std::fs::metadata(OsStr::from_bytes(candidate.as_bytes())).is_ok_and(|m| m.is_file()) {
                    return Ok((candidate, 0));
                }
                failure = libc::EACCES;
            } else if io::Error::last_os_error().raw_os_error() == Some(libc::EACCES) {
                failure = libc::EACCES;
            }
        }
        Ok((cstring(program)?, failure))
    }

    fn close_limit() -> io::Result<RawFd> {
        // SAFETY: getrlimit initializes the supplied writable structure.
        let mut limit = unsafe { std::mem::zeroed::<libc::rlimit>() };
        if unsafe { libc::getrlimit(libc::RLIMIT_NOFILE, &raw mut limit) } != 0 {
            return Err(io::Error::last_os_error());
        }
        let mut maximum = i32::try_from(limit.rlim_cur).unwrap_or(i32::MAX);
        // Existing descriptors can exceed a subsequently lowered soft limit.
        for entry in std::fs::read_dir("/dev/fd")? {
            let entry = entry?;
            if let Some(fd) = entry.file_name().to_str().and_then(|name| name.parse::<RawFd>().ok()) {
                maximum = maximum.max(fd.saturating_add(1));
            }
        }
        Ok(maximum)
    }

    struct SignalMask {
        original: libc::sigset_t,
    }

    impl SignalMask {
        fn block() -> io::Result<Self> {
            // SAFETY: libc writes only to these properly sized signal sets.
            unsafe {
                let mut all = std::mem::zeroed::<libc::sigset_t>();
                let mut original = std::mem::zeroed::<libc::sigset_t>();
                if libc::sigfillset(&raw mut all) != 0 {
                    return Err(io::Error::last_os_error());
                }
                let result = libc::pthread_sigmask(libc::SIG_SETMASK, &raw const all, &raw mut original);
                if result != 0 {
                    return Err(io::Error::from_raw_os_error(result));
                }
                Ok(Self { original })
            }
        }

        fn restore(self) -> io::Result<()> {
            // SAFETY: original is the initialized mask saved on the same thread.
            let result = unsafe { libc::pthread_sigmask(libc::SIG_SETMASK, &raw const self.original, ptr::null_mut()) };
            if result == 0 { Ok(()) } else { Err(io::Error::from_raw_os_error(result)) }
        }
    }

    struct ChildContext {
        directory: *const libc::c_char,
        executable: *const libc::c_char,
        argv: *const *const libc::c_char,
        envp: *const *const libc::c_char,
        buffer: *mut u8,
        close_limit: RawFd,
        signal_limit: libc::c_int,
        launch_error: libc::c_int,
        input: RawFd,
        output: RawFd,
        errors: RawFd,
        null: RawFd,
        default_action: *const libc::sigaction,
        stop_action: *const libc::sigaction,
        all_signals: *const libc::sigset_t,
        empty_signals: *const libc::sigset_t,
    }

    // SAFETY: all child-only functions below are reached with signals blocked,
    // live prepared memory and owned descriptors. Raw buffer offsets are bounded
    // by the successful read length (at most BUFFER_BYTES); decimal arithmetic is
    // checked against pid_t::MAX before multiplication. No index, formatting,
    // heap operation, Rust destructor or unwinding path occurs in this region.
    unsafe fn child_main(context: *const ChildContext, original_mask: *const libc::sigset_t) -> ! {
        unsafe {
            let context = &*context;
            STOP_SIGNAL.store(0, Ordering::Relaxed);
            if libc::setpgid(0, 0) != 0 || !install_signals(context) {
                child_error(c"cannot establish backend supervisor".as_ptr());
            }
            if libc::dup2(context.input, 0) < 0 || libc::dup2(context.output, 1) < 0
                || libc::dup2(context.errors, 2) < 0 || libc::dup2(context.null, 3) < 0 {
                child_error(c"cannot prepare backend descriptors".as_ptr());
            }
            close_descriptors(4, context.close_limit);
            #[cfg(target_os = "linux")]
            let children = {
                if libc::prctl(libc::PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0) != 0 {
                    child_error(c"cannot establish backend subreaper".as_ptr());
                }
                let fd = libc::open(c"/proc/thread-self/children".as_ptr(), libc::O_RDONLY | libc::O_CLOEXEC);
                if fd < 0 { child_error(c"cannot inspect backend descendants".as_ptr()); }
                fd
            };
            if libc::sigprocmask(libc::SIG_SETMASK, context.empty_signals, ptr::null_mut()) != 0 {
                child_error(c"cannot establish backend signal mask".as_ptr());
            }
            let stopped = STOP_SIGNAL.load(Ordering::Relaxed);
            if stopped != 0 { libc::_exit(128 + stopped); }
            let target = fork_without_handlers();
            if target < 0 { child_error(c"cannot fork backend subprocess".as_ptr()); }
            if target == 0 { target_main(context, original_mask); }
            let mut status = 0;
            let mut completed = false;
            let mut failed = false;
            while STOP_SIGNAL.load(Ordering::Relaxed) == 0 {
                let result = libc::waitpid(target, &raw mut status, libc::WNOHANG);
                if result == target { completed = true; break; }
                if result < 0 && errno() != libc::EINTR { failed = true; break; }
                let mut owner = libc::pollfd { fd: 0, events: libc::POLLIN, revents: 0 };
                let polled = libc::poll(&raw mut owner, 1, 5);
                if polled < 0 && errno() != libc::EINTR { failed = true; break; }
                if polled > 0 {
                    let mut byte = 0u8;
                    let read = libc::read(0, (&raw mut byte).cast(), 1);
                    if read == 0 { break; }
                    if read < 0 && errno() != libc::EINTR { failed = true; break; }
                }
            }
            if libc::kill(-target, libc::SIGKILL) != 0 && errno() != libc::ESRCH { failed = true; }
            if !completed {
                if libc::kill(target, libc::SIGKILL) != 0 && errno() != libc::ESRCH { failed = true; }
                loop {
                    let result = libc::waitpid(target, ptr::null_mut(), 0);
                    if result == target { break; }
                    if result < 0 && errno() != libc::EINTR { failed = true; break; }
                }
            }
            #[cfg(target_os = "linux")]
            if !reap_descendants(children, context.buffer) { failed = true; }
            let stopped = STOP_SIGNAL.load(Ordering::Relaxed);
            if stopped != 0 { libc::_exit(128 + stopped); }
            if failed { libc::_exit(1); }
            if !completed { libc::_exit(125); }
            if libc::WIFEXITED(status) { libc::_exit(libc::WEXITSTATUS(status)); }
            if libc::WIFSIGNALED(status) { libc::_exit(128 + libc::WTERMSIG(status)); }
            libc::_exit(1)
        }
    }

    unsafe fn target_main(context: &ChildContext, original_mask: *const libc::sigset_t) -> ! {
        unsafe {
            // Reset dispositions while still blocking stop signals, so no
            // application handler can run in either forked process.
            if libc::sigprocmask(libc::SIG_SETMASK, context.all_signals, ptr::null_mut()) != 0 {
                child_error(c"cannot prepare backend signal mask".as_ptr());
            }
            if !reset_signals(context) || libc::setsid() < 0
                || libc::dup2(3, 0) < 0 || libc::chdir(context.directory) != 0 {
                child_error(c"cannot prepare backend subprocess".as_ptr());
            }
            close_descriptors(3, context.close_limit);
            #[cfg(target_os = "macos")]
            {
                let limit = libc::rlimit { rlim_cur: 0, rlim_max: 0 };
                if libc::setrlimit(libc::RLIMIT_NPROC, &raw const limit) != 0 {
                    child_error(c"cannot prevent backend descendants".as_ptr());
                }
            }
            if libc::sigprocmask(libc::SIG_SETMASK, original_mask, ptr::null_mut()) != 0 {
                child_error(c"cannot restore backend signal mask".as_ptr());
            }
            if context.launch_error != 0 {
                child_error(c"cannot execute backend subprocess".as_ptr());
            }
            libc::execve(context.executable, context.argv, context.envp);
            child_error(c"cannot execute backend subprocess".as_ptr())
        }
    }

    extern "C" fn request_stop(signal: libc::c_int) {
        // The handler's sa_mask blocks all signals, making this first write
        // indivisible with respect to subsequent stop handlers in this process.
        let _ = STOP_SIGNAL.compare_exchange(0, signal, Ordering::Relaxed, Ordering::Relaxed);
    }

    unsafe fn reset_signals(context: &ChildContext) -> bool {
        unsafe {
            let mut signal = 1;
            while signal < context.signal_limit {
                if signal != libc::SIGKILL && signal != libc::SIGSTOP
                    && libc::sigaction(signal, context.default_action, ptr::null_mut()) != 0
                    && errno() != libc::EINVAL {
                    return false;
                }
                signal += 1;
            }
            true
        }
    }

    unsafe fn install_signals(context: &ChildContext) -> bool {
        unsafe {
            if !reset_signals(context) { return false; }
            libc::sigaction(libc::SIGTERM, context.stop_action, ptr::null_mut()) == 0
                && libc::sigaction(libc::SIGINT, context.stop_action, ptr::null_mut()) == 0
                && libc::sigaction(libc::SIGHUP, context.stop_action, ptr::null_mut()) == 0
                && libc::sigaction(libc::SIGQUIT, context.stop_action, ptr::null_mut()) == 0
        }
    }

    #[allow(clippy::cast_possible_truncation)]
    unsafe fn fork_without_handlers() -> libc::pid_t {
        #[cfg(target_os = "linux")]
        unsafe {
            // SIGCHLD with no shared-address-space flags gives fork semantics.
            // Null arguments make clone's architecture-dependent argument order
            // immaterial; libc's syscall shim invokes no pthread_atfork handlers.
            let zero: libc::c_long = 0;
            libc::syscall(libc::SYS_clone, libc::c_long::from(libc::SIGCHLD), zero, zero, zero, zero) as libc::pid_t
        }
        #[cfg(target_os = "macos")]
        unsafe { darwin_fork() }
    }

    #[cfg(target_os = "macos")]
    unsafe extern "C" {
        // Apple's libsyscall shim fixes up the kernel's two-register fork return
        // without Libc's application callbacks. Its ABI is private; native Darwin
        // validation remains distinct from source/compile evidence.
        #[link_name = "__fork"]
        fn darwin_fork() -> libc::pid_t;
    }

    unsafe fn close_descriptors(first: RawFd, limit: RawFd) {
        unsafe {
            #[cfg(target_os = "linux")]
            if libc::syscall(libc::SYS_close_range, first.cast_unsigned(), u32::MAX, 0) == 0 { return; }
            let mut fd = first;
            while fd < limit {
                libc::close(fd);
                fd += 1;
            }
        }
    }

    #[cfg(target_os = "linux")]
    unsafe fn reap_descendants(children: RawFd, buffer: *mut u8) -> bool {
        unsafe {
            let mut success = true;
            loop {
                let reaped = libc::waitpid(-1, ptr::null_mut(), libc::WNOHANG);
                if reaped > 0 { continue; }
                if reaped < 0 {
                    if errno() == libc::ECHILD { return success; }
                    if errno() == libc::EINTR { continue; }
                    return false;
                }
                if !kill_adopted(children, buffer) { success = false; }
                let delay = libc::timespec { tv_sec: 0, tv_nsec: 5_000_000 };
                libc::nanosleep(&raw const delay, ptr::null_mut());
            }
        }
    }

    #[cfg(target_os = "linux")]
    unsafe fn kill_adopted(children: RawFd, buffer: *mut u8) -> bool {
        unsafe {
            if libc::lseek(children, 0, libc::SEEK_SET) < 0 { return false; }
            let mut pid = 0;
            loop {
                let count = libc::read(children, buffer.cast(), BUFFER_BYTES);
                if count < 0 {
                    if errno() == libc::EINTR { continue; }
                    return false;
                }
                let mut offset = 0;
                while offset < count {
                    let byte = *buffer.offset(offset);
                    if byte >= b'0' && byte <= b'9' {
                        let digit = libc::pid_t::from(byte - b'0');
                        if pid > (libc::pid_t::MAX - digit) / 10 { return false; }
                        pid = pid * 10 + digit;
                    } else if byte == b' ' || byte == b'\n' {
                        if pid > 0 && libc::kill(pid, libc::SIGKILL) != 0 && errno() != libc::ESRCH { return false; }
                        pid = 0;
                    } else { return false; }
                    offset += 1;
                }
                if count == 0 {
                    return pid == 0 || libc::kill(pid, libc::SIGKILL) == 0 || errno() == libc::ESRCH;
                }
            }
        }
    }

    unsafe fn child_error(message: *const libc::c_char) -> ! {
        unsafe {
            let mut length = 0;
            while *message.add(length) != 0 { length += 1; }
            let mut offset = 0;
            while offset < length {
                let count = libc::write(2, message.add(offset).cast(), length - offset);
                if count > 0 { offset += count.cast_unsigned(); }
                else if count == 0 || errno() != libc::EINTR { break; }
            }
            libc::write(2, c"\n".as_ptr().cast(), 1);
            libc::_exit(1)
        }
    }

    unsafe fn errno() -> libc::c_int {
        #[cfg(target_os = "linux")]
        unsafe { *libc::__errno_location() }
        #[cfg(target_os = "macos")]
        unsafe { *libc::__error() }
    }

    #[allow(dead_code)]
    pub(crate) fn parent_identity() -> (u32, bool) {
        // SAFETY: these scalar process-identity queries have no pointer arguments.
        unsafe {
            let parent = libc::getppid();
            (parent.cast_unsigned(), libc::getpgid(parent) == parent)
        }
    }

    #[cfg(target_os = "linux")]
    #[allow(dead_code)]
    pub(crate) fn set_subreaper() -> io::Result<()> {
        // SAFETY: this option takes only integer arguments and modifies the caller.
        if unsafe { libc::prctl(libc::PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0) } == 0 {
            Ok(())
        } else { Err(io::Error::last_os_error()) }
    }

    #[cfg(target_os = "linux")]
    #[allow(dead_code)]
    pub(crate) fn reap_owned_children() -> io::Result<()> {
        loop {
            // SAFETY: null status is permitted; the isolated harness owns all children.
            if unsafe { libc::waitpid(-1, ptr::null_mut(), 0) } >= 0 { continue; }
            let error = io::Error::last_os_error();
            if error.raw_os_error() == Some(libc::ECHILD) { return Ok(()); }
            if error.kind() != io::ErrorKind::Interrupted { return Err(error); }
        }
    }

    #[cfg(target_os = "linux")]
    #[allow(dead_code)]
    pub(crate) fn poison_atfork_child() -> io::Result<()> {
        unsafe extern "C" fn poison() {
            // SAFETY: a callback reached after an unintended libc fork must not
            // return to the test; _exit is async-signal-safe and never unwinds.
            unsafe { libc::_exit(99) }
        }
        // SAFETY: the callback is a static function; the isolated fixture never
        // calls libc fork again and exits immediately after its control.
        let result = unsafe { libc::pthread_atfork(None, None, Some(poison)) };
        if result == 0 { Ok(()) } else { Err(io::Error::from_raw_os_error(result)) }
    }

    #[cfg(test)]
    mod tests {
        use super::*;
        use std::{fs, thread, time::{Duration, Instant}};

        #[test]
        fn supervisor_owner_eof_returns_125() -> io::Result<()> {
            let directory = std::env::temp_dir().join(format!("umpire-owner-eof-{}", std::process::id()));
            fs::create_dir(&directory)?;
            let out = File::create(directory.join("stdout"))?;
            let err = File::create(directory.join("stderr"))?;
            let mut command = Command::new("/bin/sleep");
            command.arg("60");
            let mut running = Running::spawn(&command, &directory, &out, &err)?;
            drop(running.owner.take());
            let deadline = Instant::now() + Duration::from_secs(5);
            let result = loop {
                if let Some(status) = running.try_wait()? { break status.code(); }
                assert!(Instant::now() < deadline, "supervisor ignored owner EOF");
                thread::sleep(Duration::from_millis(5));
            };
            assert_eq!(result, Some(125));
            drop(running);
            fs::remove_dir_all(directory)
        }

        #[test]
        fn supervisor_retains_first_stop_signal() {
            STOP_SIGNAL.store(0, Ordering::Relaxed);
            request_stop(libc::SIGTERM);
            request_stop(libc::SIGINT);
            assert_eq!(STOP_SIGNAL.load(Ordering::Relaxed), libc::SIGTERM);
            STOP_SIGNAL.store(0, Ordering::Relaxed);
        }
    }
}

#[cfg(any(target_os = "linux", target_os = "macos"))]
pub(crate) use supported::Running;
#[cfg(any(target_os = "linux", target_os = "macos"))]
#[allow(unused_imports)]
pub(crate) use supported::parent_identity;
#[cfg(target_os = "linux")]
#[allow(unused_imports)]
pub(crate) use supported::{poison_atfork_child, reap_owned_children, set_subreaper};

#[cfg(not(any(target_os = "linux", target_os = "macos")))]
pub(crate) struct Running;
#[cfg(not(any(target_os = "linux", target_os = "macos")))]
impl Running {
    pub(crate) fn spawn(_: &Command, _: &Path, _: &File, _: &File) -> io::Result<Self> {
        platform_policy(std::env::consts::OS, 0, 0)?;
        Ok(Self)
    }
    pub(crate) fn try_wait(&mut self) -> io::Result<Option<ExitStatus>> {
        platform_policy(std::env::consts::OS, 0, 0)?;
        Ok(None)
    }
}

#[cfg(test)]
mod tests {
    use super::platform_policy;

    #[test]
    fn all_darwin_helpers_prevent_process_creation() {
        assert!(platform_policy("macos", 501, 501).is_ok());
        for (uid, euid) in [(0, 501), (501, 0), (0, 0)] {
            assert!(platform_policy("macos", uid, euid).is_err_and(|e| e.to_string().contains("non-root")));
        }
        assert!(platform_policy("linux", 0, 0).is_ok());
    }

    #[test]
    fn unsupported_platform_is_an_explicit_refusal() {
        assert!(platform_policy("unsupported", 501, 501).is_err_and(|e| {
            e.to_string() == "backend process ownership supports Linux and macOS only"
        }));
    }
}
