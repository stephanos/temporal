//! The only unsafe boundary for backend process ownership.
//!
//! All owned strings, pointer tables, descriptors and parsing storage are prepared
//! before fork. The two child entry points never return to Rust: their transitive
//! callees use only libc, raw pointers and bounded scalar operations, and every
//! exit is `_exit`. In particular, no child drops parent-owned Rust values.
#![allow(unsafe_code)]

use std::{
    fs::File,
    io,
    path::Path,
    process::{Command, ExitStatus},
};

#[cfg(any(target_os = "linux", target_os = "macos"))]
#[path = "syscall/calls.rs"]
mod calls;

fn platform_policy(platform: &str, uid: u32, euid: u32) -> io::Result<()> {
    match platform {
        "linux" => Ok(()),
        "macos" if uid != 0 && euid != 0 => Ok(()),
        "macos" => Err(io::Error::other("Darwin descendant prevention requires a non-root user")),
        _ => Err(io::Error::other("backend process ownership supports Linux and macOS only")),
    }
}

#[cfg(any(target_os = "linux", target_os = "macos"))]
mod supported {
    use super::{Command, ExitStatus, File, Path, calls, io, platform_policy};
    use std::{
        collections::BTreeMap,
        ffi::{CString, OsStr, OsString},
        io::PipeWriter,
        os::{
            fd::{AsFd, AsRawFd, OwnedFd, RawFd},
            unix::{ffi::OsStrExt, process::ExitStatusExt},
        },
        ptr,
        sync::atomic::{AtomicI32, Ordering},
    };

    use rustix::process::{self, Pid, Resource, WaitOptions};

    const BUFFER_BYTES: usize = 4096;
    #[cfg(target_os = "linux")]
    type SignalSet = [u64; 32];
    #[cfg(target_os = "macos")]
    type SignalSet = libc::sigset_t;
    #[cfg(not(target_has_atomic = "32"))]
    compile_error!("backend signal ownership requires lock-free 32-bit atomics");
    static STOP_SIGNAL: AtomicI32 = AtomicI32::new(0);

    pub(crate) struct Running {
        pid: Pid,
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
            platform_policy(
                std::env::consts::OS,
                process::getuid().as_raw(),
                process::geteuid().as_raw(),
            )?;
            let mut prepared = Prepared::new(command, fallback_directory)?;
            let (owner_read, owner_write) = io::pipe()?;
            let null = File::open("/dev/null")?;
            let input = duplicate(&owner_read)?;
            let output = duplicate(out)?;
            let errors = duplicate(err)?;
            let null = duplicate(&null)?;
            let context = prepared.context(
                input.as_raw_fd(),
                output.as_raw_fd(),
                errors.as_raw_fd(),
                null.as_raw_fd(),
            );
            let mask = SignalMask::block(&context)?;
            // SAFETY: all child memory and descriptors are live across fork. Only
            // the child takes child_main, which cannot return or drop these values.
            let pid = unsafe { fork_without_handlers(&prepared.api) };
            if pid == 0 {
                // SAFETY: context points to immutable prepared tables and one
                // uniquely owned parsing buffer in this process's private memory.
                unsafe { child_main(&raw const context, &raw const mask.original) }
            }
            if pid < 0 {
                let error = io::Error::last_os_error();
                mask.restore()?;
                return Err(error);
            }
            let pid = Pid::from_raw(pid).ok_or_else(|| io::Error::other("invalid child PID"))?;
            let running = Self { pid, owner: Some(owner_write), status: None };
            mask.restore()?;
            Ok(running)
        }

        pub(crate) fn try_wait(&mut self) -> io::Result<Option<ExitStatus>> {
            if self.status.is_some() {
                return Ok(self.status);
            }
            loop {
                match process::waitpid(Some(self.pid), WaitOptions::NOHANG) {
                    Ok(Some((_, status))) => {
                        let status = ExitStatus::from_raw(status.as_raw());
                        self.status = Some(status);
                        return Ok(Some(status));
                    }
                    Ok(None) => return Ok(None),
                    Err(rustix::io::Errno::INTR) => {}
                    Err(error) => return Err(error.into()),
                }
            }
        }
    }

    impl Drop for Running {
        fn drop(&mut self) {
            drop(self.owner.take());
            if self.status.is_none() {
                while matches!(
                    process::waitpid(Some(self.pid), WaitOptions::empty()),
                    Err(rustix::io::Errno::INTR)
                ) {}
            }
        }
    }

    fn duplicate(fd: impl AsFd) -> io::Result<OwnedFd> {
        rustix::io::fcntl_dupfd_cloexec(fd, 10).map_err(Into::into)
    }

    struct Prepared {
        api: calls::Api,
        directory: CString,
        _executables: Vec<CString>,
        executables: Vec<*const libc::c_char>,
        _arguments: Vec<CString>,
        _environment: Vec<CString>,
        argv: Vec<*const libc::c_char>,
        envp: Vec<*const libc::c_char>,
        buffer: Box<[u8; BUFFER_BYTES]>,
        close_limit: RawFd,
        signal_limit: libc::c_int,
        #[cfg(target_os = "macos")]
        default_action: libc::sigaction,
        stop_action: libc::sigaction,
        all_signals: SignalSet,
        empty_signals: SignalSet,
        #[cfg(target_os = "linux")]
        kernel_signal_bytes: usize,
        #[cfg(target_os = "linux")]
        kernel_default_action: [u64; 32],
    }

    impl Prepared {
        fn new(command: &Command, fallback_directory: &Path) -> io::Result<Self> {
            let api = calls::Api::new()?;
            let directory =
                command.get_current_dir().unwrap_or(fallback_directory).canonicalize()?;
            let environment = environment(command)?;
            let executable_bytes =
                executable_candidates(command.get_program(), &directory, &environment)?;
            let mut executables =
                executable_bytes.iter().map(|value| value.as_ptr()).collect::<Vec<_>>();
            executables.push(ptr::null());
            let mut arguments = vec![cstring(command.get_program())?];
            arguments.extend(command.get_args().map(cstring).collect::<io::Result<Vec<_>>>()?);
            let environment = environment
                .into_iter()
                .map(|(key, value)| {
                    let mut bytes = key.as_bytes().to_vec();
                    bytes.push(b'=');
                    bytes.extend_from_slice(value.as_bytes());
                    CString::new(bytes).map_err(io::Error::other)
                })
                .collect::<io::Result<Vec<_>>>()?;
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
            let stop_action = unsafe {
                let mut action = std::mem::zeroed::<libc::sigaction>();
                if libc::sigfillset(&raw mut action.sa_mask) != 0 {
                    return Err(io::Error::last_os_error());
                }
                action.sa_sigaction = request_stop as *const () as usize;
                action
            };
            #[cfg(target_os = "macos")]
            let default_action = unsafe {
                let mut action = std::mem::zeroed::<libc::sigaction>();
                if libc::sigfillset(&raw mut action.sa_mask) != 0 {
                    return Err(io::Error::last_os_error());
                }
                action.sa_sigaction = libc::SIG_DFL;
                action
            };
            #[cfg(target_os = "linux")]
            let (all_signals, empty_signals) = ([u64::MAX; 32], [0; 32]);
            #[cfg(target_os = "macos")]
            let (all_signals, empty_signals) = unsafe {
                let mut all = std::mem::zeroed::<libc::sigset_t>();
                let mut empty = std::mem::zeroed::<libc::sigset_t>();
                if libc::sigfillset(&raw mut all) != 0 || libc::sigemptyset(&raw mut empty) != 0 {
                    return Err(io::Error::last_os_error());
                }
                (all, empty)
            };
            #[cfg(target_os = "linux")]
            let kernel_signal_bytes =
                usize::try_from((signal_limit - 1) / 8).map_err(io::Error::other)?;
            Ok(Self {
                api,
                #[cfg(target_os = "macos")]
                default_action,
                stop_action,
                all_signals,
                empty_signals,
                #[cfg(target_os = "linux")]
                kernel_default_action: [0; 32],
                #[cfg(target_os = "linux")]
                kernel_signal_bytes,
                directory: cstring(directory.as_os_str())?,
                _executables: executable_bytes,
                executables,
                _arguments: arguments,
                _environment: environment,
                argv,
                envp,
                buffer: Box::new([0; BUFFER_BYTES]),
                close_limit,
                signal_limit,
            })
        }

        fn context(
            &mut self,
            input: RawFd,
            output: RawFd,
            errors: RawFd,
            null: RawFd,
        ) -> ChildContext {
            ChildContext {
                api: &raw const self.api,
                directory: self.directory.as_ptr(),
                executables: self.executables.as_ptr(),
                argv: self.argv.as_ptr(),
                envp: self.envp.as_ptr(),
                buffer: self.buffer.as_mut_ptr(),
                close_limit: self.close_limit,
                signal_limit: self.signal_limit,
                input,
                output,
                errors,
                null,
                #[cfg(target_os = "macos")]
                default_action: &raw const self.default_action,
                stop_action: &raw const self.stop_action,
                all_signals: &raw const self.all_signals,
                empty_signals: &raw const self.empty_signals,
                #[cfg(target_os = "linux")]
                kernel_default_action: self.kernel_default_action.as_ptr(),
                #[cfg(target_os = "linux")]
                kernel_signal_bytes: self.kernel_signal_bytes,
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

    fn executable_candidates(
        program: &OsStr,
        directory: &Path,
        environment: &BTreeMap<OsString, OsString>,
    ) -> io::Result<Vec<CString>> {
        if program.as_bytes().contains(&b'/') {
            return Ok(vec![cstring(program)?]);
        }
        let path = environment
            .get(OsStr::new("PATH"))
            .map_or(OsStr::new("/bin:/usr/bin"), |v| v.as_os_str());
        std::env::split_paths(path)
            .map(|entry| cstring(directory.join(entry).join(program).as_os_str()))
            .collect()
    }

    fn close_limit() -> io::Result<RawFd> {
        let limit = process::getrlimit(Resource::Nofile);
        let mut maximum =
            limit.current.and_then(|value| i32::try_from(value).ok()).unwrap_or(i32::MAX);
        // Existing descriptors can exceed a subsequently lowered soft limit.
        for entry in std::fs::read_dir("/dev/fd")? {
            let entry = entry?;
            if let Some(fd) = entry.file_name().to_str().and_then(|name| name.parse::<RawFd>().ok())
            {
                maximum = maximum.max(fd.saturating_add(1));
            }
        }
        Ok(maximum)
    }

    struct SignalMask {
        original: SignalSet,
        #[cfg(target_os = "linux")]
        kernel_signal_bytes: usize,
    }

    impl SignalMask {
        fn block(context: &ChildContext) -> io::Result<Self> {
            // SAFETY: the original mask storage is writable and both Linux
            // masks cover every kernel signal, including NPTL's reserved pair.
            unsafe {
                let mut original = std::mem::zeroed::<SignalSet>();
                if replace_mask(context, context.all_signals, &raw mut original) != 0 {
                    return Err(io::Error::last_os_error());
                }
                Ok(Self {
                    original,
                    #[cfg(target_os = "linux")]
                    kernel_signal_bytes: context.kernel_signal_bytes,
                })
            }
        }

        fn restore(self) -> io::Result<()> {
            // SAFETY: only this parent thread reads its saved initialized mask.
            #[cfg(target_os = "linux")]
            let result = unsafe {
                libc::syscall(
                    libc::SYS_rt_sigprocmask,
                    libc::SIG_SETMASK,
                    &raw const self.original,
                    ptr::null_mut::<libc::c_void>(),
                    self.kernel_signal_bytes,
                )
            };
            #[cfg(target_os = "macos")]
            let result = unsafe {
                libc::sigprocmask(libc::SIG_SETMASK, &raw const self.original, ptr::null_mut())
            };
            if result == 0 { Ok(()) } else { Err(io::Error::last_os_error()) }
        }
    }

    struct ChildContext {
        api: *const calls::Api,
        directory: *const libc::c_char,
        executables: *const *const libc::c_char,
        argv: *const *const libc::c_char,
        envp: *const *const libc::c_char,
        buffer: *mut u8,
        close_limit: RawFd,
        signal_limit: libc::c_int,
        input: RawFd,
        output: RawFd,
        errors: RawFd,
        null: RawFd,
        #[cfg(target_os = "macos")]
        default_action: *const libc::sigaction,
        stop_action: *const libc::sigaction,
        all_signals: *const SignalSet,
        empty_signals: *const SignalSet,
        #[cfg(target_os = "linux")]
        kernel_signal_bytes: usize,
        #[cfg(target_os = "linux")]
        kernel_default_action: *const u64,
    }

    // SAFETY: all child-only functions below are reached with signals blocked,
    // live prepared memory and owned descriptors. Raw buffer offsets are bounded
    // by the successful read length (at most BUFFER_BYTES); decimal arithmetic is
    // checked against pid_t::MAX before multiplication. No index, formatting,
    // heap operation, Rust destructor or unwinding path occurs in this region.
    unsafe fn child_main(context: *const ChildContext, original_mask: *const SignalSet) -> ! {
        unsafe {
            let context = &*context;
            let api = &*context.api;
            #[cfg(target_os = "macos")]
            if api.cancel_state(2) != 0 {
                child_error(api, c"cannot disable backend pthread cancellation".as_ptr());
            }
            STOP_SIGNAL.store(0, Ordering::Relaxed);
            if api.dup2(context.input, 0) < 0
                || api.dup2(context.output, 1) < 0
                || api.dup2(context.errors, 2) < 0
                || api.dup2(context.null, 3) < 0
            {
                child_error(api, c"cannot prepare backend descriptors".as_ptr());
            }
            if api.setpgid(0, 0) != 0 || !install_signals(context) {
                child_error(api, c"cannot establish backend supervisor".as_ptr());
            }
            close_descriptors(api, 4, context.close_limit);
            #[cfg(target_os = "linux")]
            let children = {
                if !api.subreaper() {
                    child_error(api, c"cannot establish backend subreaper".as_ptr());
                }
                let fd = calls::open_children(api);
                if fd < 0 {
                    child_error(api, c"cannot inspect backend descendants".as_ptr());
                }
                fd
            };
            if replace_mask(context, context.empty_signals, ptr::null_mut()) != 0 {
                child_error(api, c"cannot establish backend signal mask".as_ptr());
            }
            let stopped = STOP_SIGNAL.load(Ordering::Relaxed);
            if stopped != 0 {
                api.exit(128 + stopped);
            }
            let target = fork_without_handlers(api);
            if target < 0 {
                child_error(api, c"cannot fork backend subprocess".as_ptr());
            }
            if target == 0 {
                target_main(context, original_mask);
            }
            let (status, completed, mut failed) = wait_for_target(context, target);
            if api.kill(-target, libc::SIGKILL) != 0 && errno(api) != libc::ESRCH {
                failed = true;
            }
            if !completed {
                if api.kill(target, libc::SIGKILL) != 0 && errno(api) != libc::ESRCH {
                    failed = true;
                }
                loop {
                    let result = calls::waitpid(api, target, ptr::null_mut(), 0);
                    if result == target {
                        break;
                    }
                    if result < 0 && errno(api) != libc::EINTR {
                        failed = true;
                        break;
                    }
                }
            }
            #[cfg(target_os = "linux")]
            if !reap_descendants(api, children, context.buffer) {
                failed = true;
            }
            let stopped = STOP_SIGNAL.load(Ordering::Relaxed);
            if stopped != 0 {
                api.exit(128 + stopped);
            }
            if failed {
                api.exit(1);
            }
            if !completed {
                api.exit(125);
            }
            if libc::WIFEXITED(status) {
                api.exit(libc::WEXITSTATUS(status));
            }
            if libc::WIFSIGNALED(status) {
                api.exit(128 + libc::WTERMSIG(status));
            }
            api.exit(1)
        }
    }

    unsafe fn wait_for_target(
        context: &ChildContext,
        target: libc::pid_t,
    ) -> (libc::c_int, bool, bool) {
        unsafe {
            let api = &*context.api;
            let mut status = 0;
            let mut completed = false;
            let mut failed = false;
            while STOP_SIGNAL.load(Ordering::Relaxed) == 0 {
                let result = calls::waitpid(api, target, &raw mut status, libc::WNOHANG);
                if result == target {
                    completed = true;
                    break;
                }
                if result < 0 && errno(api) != libc::EINTR {
                    failed = true;
                    break;
                }
                let mut owner = libc::pollfd { fd: 0, events: libc::POLLIN, revents: 0 };
                let polled = calls::poll_owner(api, &raw mut owner);
                if polled < 0 && errno(api) != libc::EINTR {
                    failed = true;
                    break;
                }
                if polled > 0 {
                    let mut byte = 0u8;
                    let read = calls::read(api, 0, (&raw mut byte).cast(), 1);
                    if read == 0 {
                        break;
                    }
                    if read < 0 && errno(api) != libc::EINTR {
                        failed = true;
                        break;
                    }
                }
            }
            (status, completed, failed)
        }
    }

    unsafe fn target_main(context: &ChildContext, original_mask: *const SignalSet) -> ! {
        unsafe {
            let api = &*context.api;
            #[cfg(target_os = "macos")]
            if api.cancel_state(2) != 0 {
                child_error(api, c"cannot disable backend pthread cancellation".as_ptr());
            }
            // Reset dispositions while still blocking stop signals, so no
            // application handler can run in either forked process.
            if replace_mask(context, context.all_signals, ptr::null_mut()) != 0 {
                child_error(api, c"cannot prepare backend signal mask".as_ptr());
            }
            if !reset_signals(context)
                || api.setsid() < 0
                || api.dup2(3, 0) < 0
                || api.chdir(context.directory) != 0
            {
                child_error(api, c"cannot prepare backend subprocess".as_ptr());
            }
            close_descriptors(api, 3, context.close_limit);
            #[cfg(target_os = "macos")]
            {
                if !prevent_descendants(api) {
                    child_error(api, c"cannot prevent backend descendants".as_ptr());
                }
            }
            if replace_mask(context, original_mask, ptr::null_mut()) != 0 {
                child_error(api, c"cannot restore backend signal mask".as_ptr());
            }
            let mut candidates = context.executables;
            let mut saved_errno = 0;
            while !(*candidates).is_null() {
                api.execve(*candidates, context.argv, context.envp);
                let error = errno(api);
                if saved_errno == 0 && error != libc::ENOENT && error != libc::ENOTDIR {
                    saved_errno = error;
                }
                candidates = candidates.add(1);
            }
            if saved_errno != 0 {
                *api.errno = saved_errno;
            }
            child_error(api, c"cannot execute backend subprocess".as_ptr())
        }
    }

    extern "C" fn request_stop(signal: libc::c_int) {
        // The handler's sa_mask blocks all signals, making this first write
        // indivisible with respect to subsequent stop handlers in this process.
        let _ = STOP_SIGNAL.compare_exchange(0, signal, Ordering::Relaxed, Ordering::Relaxed);
    }

    unsafe fn reset_signals(context: &ChildContext) -> bool {
        unsafe {
            let api = &*context.api;
            let mut signal = 1;
            while signal < context.signal_limit {
                if signal != libc::SIGKILL
                    && signal != libc::SIGSTOP
                    && !reset_signal(context, signal)
                    && errno(api) != libc::EINVAL
                {
                    return false;
                }
                signal += 1;
            }
            true
        }
    }

    unsafe fn reset_signal(context: &ChildContext, signal: libc::c_int) -> bool {
        unsafe {
            let api = &*context.api;
            #[cfg(target_os = "linux")]
            {
                // Every kernel field is zero: SIG_DFL, no restorer, no mask.
                // This bypasses glibc's abort lock and reserved-signal filtering.
                (api.syscall())(
                    libc::SYS_rt_sigaction,
                    signal,
                    context.kernel_default_action,
                    ptr::null_mut::<libc::c_void>(),
                    context.kernel_signal_bytes,
                ) == 0
            }
            #[cfg(target_os = "macos")]
            {
                api.sigaction(signal, context.default_action, ptr::null_mut()) == 0
            }
        }
    }

    unsafe fn replace_mask(
        context: &ChildContext,
        set: *const SignalSet,
        old: *mut SignalSet,
    ) -> libc::c_long {
        unsafe {
            let api = &*context.api;
            #[cfg(target_os = "linux")]
            {
                (api.syscall())(
                    libc::SYS_rt_sigprocmask,
                    libc::SIG_SETMASK,
                    set,
                    old,
                    context.kernel_signal_bytes,
                )
            }
            #[cfg(target_os = "macos")]
            {
                libc::c_long::from(api.sigprocmask(libc::SIG_SETMASK, set, old))
            }
        }
    }

    unsafe fn prevent_descendants(api: &calls::Api) -> bool {
        unsafe {
            let limit = libc::rlimit { rlim_cur: 0, rlim_max: 0 };
            #[cfg(target_os = "macos")]
            let resource = libc::RLIMIT_NPROC | 0x1000;
            #[cfg(target_os = "linux")]
            let resource = libc::RLIMIT_NPROC;
            api.setrlimit(resource, &raw const limit) == 0
        }
    }

    unsafe fn install_signals(context: &ChildContext) -> bool {
        unsafe {
            let api = &*context.api;
            if !reset_signals(context) {
                return false;
            }
            api.sigaction(libc::SIGTERM, context.stop_action, ptr::null_mut()) == 0
                && api.sigaction(libc::SIGINT, context.stop_action, ptr::null_mut()) == 0
                && api.sigaction(libc::SIGHUP, context.stop_action, ptr::null_mut()) == 0
                && api.sigaction(libc::SIGQUIT, context.stop_action, ptr::null_mut()) == 0
        }
    }

    #[cfg(target_os = "linux")]
    fn clone_prefix(stack_first: bool) -> (libc::c_long, libc::c_long) {
        let flags = libc::c_long::from(libc::SIGCHLD);
        if stack_first { (0, flags) } else { (flags, 0) }
    }

    #[cfg(all(target_os = "linux", any(test, target_arch = "sparc", target_arch = "sparc64")))]
    #[allow(clippy::cast_possible_truncation, reason = "the kernel returns pid_t values or -1")]
    fn clone_pid(result: libc::c_long, before: libc::c_long, after: libc::c_long) -> libc::pid_t {
        if result >= 0 && before != after { 0 } else { result as libc::pid_t }
    }

    #[allow(clippy::cast_possible_truncation)]
    unsafe fn fork_without_handlers(api: &calls::Api) -> libc::pid_t {
        #[cfg(target_os = "linux")]
        unsafe {
            // SIGCHLD with no shared-address-space flags gives fork semantics.
            // s390x puts the null stack before flags; every remaining ABI slot
            // is zero, including the extra CLONE_BACKWARDS3 stack-size slot.
            let zero: libc::c_long = 0;
            let (first, second) = clone_prefix(cfg!(target_arch = "s390x"));
            #[cfg(any(target_arch = "sparc", target_arch = "sparc64"))]
            let before = (api.syscall())(libc::SYS_getpid);
            let result = (api.syscall())(libc::SYS_clone, first, second, zero, zero, zero, zero);
            #[cfg(any(target_arch = "sparc", target_arch = "sparc64"))]
            {
                let after = (api.syscall())(libc::SYS_getpid);
                clone_pid(result, before, after)
            }
            #[cfg(not(any(target_arch = "sparc", target_arch = "sparc64")))]
            {
                result as libc::pid_t
            }
        }
        #[cfg(target_os = "macos")]
        unsafe {
            api.fork()
        }
    }

    unsafe fn close_descriptors(api: &calls::Api, first: RawFd, limit: RawFd) {
        unsafe {
            #[cfg(target_os = "linux")]
            if (api.syscall())(libc::SYS_close_range, first.cast_unsigned(), u32::MAX, 0) == 0 {
                return;
            }
            let mut fd = first;
            while fd < limit {
                calls::close(api, fd);
                fd += 1;
            }
        }
    }

    #[cfg(target_os = "linux")]
    unsafe fn reap_descendants(api: &calls::Api, children: RawFd, buffer: *mut u8) -> bool {
        unsafe {
            let mut success = true;
            loop {
                let reaped = calls::waitpid(api, -1, ptr::null_mut(), libc::WNOHANG);
                if reaped > 0 {
                    continue;
                }
                if reaped < 0 {
                    if errno(api) == libc::ECHILD {
                        return success;
                    }
                    if errno(api) == libc::EINTR {
                        continue;
                    }
                    return false;
                }
                if !kill_adopted(api, children, buffer) {
                    success = false;
                }
                calls::pause(api);
            }
        }
    }

    #[cfg(target_os = "linux")]
    unsafe fn kill_adopted(api: &calls::Api, children: RawFd, buffer: *mut u8) -> bool {
        unsafe {
            if api.lseek(children, 0, libc::SEEK_SET) < 0 {
                return false;
            }
            let mut pid = 0;
            loop {
                let count = calls::read(api, children, buffer.cast(), BUFFER_BYTES);
                if count < 0 {
                    if errno(api) == libc::EINTR {
                        continue;
                    }
                    return false;
                }
                let mut offset = 0;
                while offset < count {
                    let byte = *buffer.offset(offset);
                    if byte.wrapping_sub(b'0') <= 9 {
                        let digit = libc::pid_t::from(byte - b'0');
                        if pid > (libc::pid_t::MAX - digit) / 10 {
                            return false;
                        }
                        pid = pid * 10 + digit;
                    } else if byte == b' ' || byte == b'\n' {
                        if pid > 0 && api.kill(pid, libc::SIGKILL) != 0 && errno(api) != libc::ESRCH
                        {
                            return false;
                        }
                        pid = 0;
                    } else {
                        return false;
                    }
                    offset += 1;
                }
                if count == 0 {
                    return pid == 0
                        || api.kill(pid, libc::SIGKILL) == 0
                        || errno(api) == libc::ESRCH;
                }
            }
        }
    }

    unsafe fn child_error(api: &calls::Api, message: *const libc::c_char) -> ! {
        unsafe {
            let mut length = 0;
            while *message.add(length) != 0 {
                length += 1;
            }
            let mut offset = 0;
            while offset < length {
                let count = calls::write(api, 2, message.add(offset).cast(), length - offset);
                if count > 0 {
                    offset += count.cast_unsigned();
                } else if count == 0 || errno(api) != libc::EINTR {
                    break;
                }
            }
            calls::write(api, 2, c"\n".as_ptr().cast(), 1);
            api.exit(1)
        }
    }

    unsafe fn errno(api: &calls::Api) -> libc::c_int {
        unsafe { *api.errno }
    }

    #[allow(dead_code)]
    pub(crate) fn restrict_descendants() -> io::Result<()> {
        // SAFETY: the fixture deliberately restricts only its own process;
        // production calls the identical helper in the target before exec.
        let api = calls::Api::new()?;
        if unsafe { prevent_descendants(&api) } { Ok(()) } else { Err(io::Error::last_os_error()) }
    }

    #[allow(dead_code)]
    pub(crate) fn parent_identity() -> (u32, bool) {
        process::getppid().map_or((0, false), |parent| {
            (parent.as_raw_pid().cast_unsigned(), process::getpgid(Some(parent)) == Ok(parent))
        })
    }

    #[cfg(target_os = "linux")]
    #[allow(dead_code)]
    pub(crate) fn set_subreaper() -> io::Result<()> {
        process::set_child_subreaper(Some(Pid::INIT)).map_err(Into::into)
    }

    #[cfg(target_os = "linux")]
    #[allow(dead_code)]
    pub(crate) fn reap_owned_children() -> io::Result<()> {
        loop {
            match process::wait(WaitOptions::empty()) {
                Ok(_) | Err(rustix::io::Errno::INTR) => {}
                Err(rustix::io::Errno::CHILD) => return Ok(()),
                Err(error) => return Err(error.into()),
            }
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
        use std::{
            fs, thread,
            time::{Duration, Instant},
        };

        #[test]
        fn duplicated_descriptors_are_private_and_close_on_exec() -> io::Result<()> {
            let file = File::open("/dev/null")?;
            let descriptor = duplicate(&file)?;
            assert!(descriptor.as_raw_fd() >= 10);
            assert!(rustix::io::fcntl_getfd(&descriptor)?.contains(rustix::io::FdFlags::CLOEXEC));
            Ok(())
        }

        #[test]
        fn close_limit_retains_descriptors_above_lowered_soft_limit() -> io::Result<()> {
            const CONTROL: &str = "UMPIRE_CLOSE_LIMIT_CONTROL";
            if std::env::var_os(CONTROL).is_none() {
                let output = Command::new(std::env::current_exe()?)
                    .args(["--exact", "syscall::supported::tests::close_limit_retains_descriptors_above_lowered_soft_limit"])
                    .env(CONTROL, "1").output()?;
                assert!(
                    output.status.success(),
                    "{}{}",
                    String::from_utf8_lossy(&output.stdout),
                    String::from_utf8_lossy(&output.stderr)
                );
                return Ok(());
            }
            let file = File::open("/dev/null")?;
            let high = rustix::io::fcntl_dupfd_cloexec(&file, 128)?;
            let prior = process::getrlimit(Resource::Nofile);
            process::setrlimit(Resource::Nofile, process::Rlimit { current: Some(64), ..prior })?;
            assert!(close_limit()? > high.as_raw_fd());
            Ok(())
        }

        #[test]
        fn supervisor_owner_eof_returns_125() -> io::Result<()> {
            let directory =
                std::env::temp_dir().join(format!("umpire-owner-eof-{}", std::process::id()));
            fs::create_dir(&directory)?;
            let out = File::create(directory.join("stdout"))?;
            let err = File::create(directory.join("stderr"))?;
            let mut command = Command::new("/bin/sleep");
            command.arg("60");
            let mut running = Running::spawn(&command, &directory, &out, &err)?;
            drop(running.owner.take());
            let deadline = Instant::now() + Duration::from_secs(5);
            let result = loop {
                if let Some(status) = running.try_wait()? {
                    break status.code();
                }
                assert!(Instant::now() < deadline, "supervisor ignored owner EOF");
                thread::sleep(Duration::from_millis(5));
            };
            assert_eq!(result, Some(125));
            drop(running);
            fs::remove_dir_all(directory)
        }

        #[test]
        #[cfg(target_os = "linux")]
        fn raw_clone_prefix_preserves_flags_and_null_stack() {
            let flags = libc::c_long::from(libc::SIGCHLD);
            assert_eq!([clone_prefix(false), clone_prefix(true)], [(flags, 0), (0, flags)]);
        }

        #[test]
        #[cfg(target_os = "linux")]
        fn raw_clone_normalizes_sparc_child_identity() {
            assert_eq!(
                [clone_pid(42, 42, 43), clone_pid(43, 42, 42), clone_pid(-1, 42, 42)],
                [0, 43, -1]
            );
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
#[allow(unused_imports)]
pub(crate) use supported::Running;
#[cfg(any(target_os = "linux", target_os = "macos"))]
#[allow(unused_imports)]
pub(crate) use supported::{parent_identity, restrict_descendants};
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
            assert!(
                platform_policy("macos", uid, euid)
                    .is_err_and(|e| e.to_string().contains("non-root"))
            );
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
