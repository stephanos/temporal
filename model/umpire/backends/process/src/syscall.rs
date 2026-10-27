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

pub(crate) const LEASE_FD: i32 = 9;
pub(crate) const LEASE_ENV: &str = "UMPIRE_JVM_LEASE_FD";

pub(crate) const COMPLETION_FD: i32 = 10;
pub(crate) const CANCELLATION_FD: i32 = 11;
const COMPLETION_ENV: &str = "UMPIRE_COMPLETION_FD";
const CANCELLATION_ENV: &str = "UMPIRE_CANCELLATION_FD";

pub(crate) fn inherited_lease() -> io::Result<File> {
    inherited_descriptor(LEASE_FD)
}

pub(crate) fn inherited_descriptor(source: i32) -> io::Result<File> {
    use std::os::fd::FromRawFd;
    // SAFETY: successful duplication creates a unique owned descriptor. Reserved
    // protocol slots must never be used as temporary duplicate destinations.
    let descriptor = unsafe { libc::fcntl(source, libc::F_DUPFD_CLOEXEC, 12) };
    if descriptor < 0 {
        return Err(io::Error::last_os_error());
    }
    // SAFETY: successful F_DUPFD_CLOEXEC returned a unique owned descriptor.
    Ok(unsafe { File::from_raw_fd(descriptor) })
}

fn inherited_lifecycle() -> io::Result<Option<(File, File)>> {
    use std::os::{fd::AsRawFd, unix::fs::FileTypeExt};
    match (std::env::var_os(COMPLETION_ENV), std::env::var_os(CANCELLATION_ENV)) {
        (None, None) => return Ok(None),
        (Some(completion), Some(cancellation)) if completion == "10" && cancellation == "11" => (),
        _ => return Err(io::Error::other("invalid inherited lifecycle descriptor locators")),
    }
    let completion = inherited_descriptor(COMPLETION_FD)?;
    let cancellation = inherited_descriptor(CANCELLATION_FD)?;
    for (file, access) in [(&completion, libc::O_WRONLY), (&cancellation, libc::O_RDONLY)] {
        // SAFETY: F_GETFL only queries the live, owned descriptor.
        let flags = unsafe { libc::fcntl(file.as_raw_fd(), libc::F_GETFL) };
        if !file.metadata()?.file_type().is_fifo() || flags < 0 || flags & libc::O_ACCMODE != access
        {
            return Err(io::Error::other(
                "inherited lifecycle requires directed pipe capabilities",
            ));
        }
    }
    // SAFETY: poll borrows initialized storage and never consumes broadcast data.
    let mut poll = libc::pollfd { fd: cancellation.as_raw_fd(), events: libc::POLLIN, revents: 0 };
    if unsafe { libc::poll(&raw mut poll, 1, 0) } < 0 {
        return Err(io::Error::last_os_error());
    }
    if poll.revents != 0 {
        return Err(io::Error::other("inherited lifecycle is already cancelled"));
    }
    Ok(Some((completion, cancellation)))
}

#[cfg(any(test, not(any(target_os = "linux", target_os = "macos"))))]
fn platform_policy(platform: &str, uid: u32, euid: u32) -> io::Result<()> {
    platform_policy_with_descendants(platform, uid, euid, crate::DescendantPolicy::Checker)
}

fn platform_policy_with_descendants(
    platform: &str,
    uid: u32,
    euid: u32,
    policy: crate::DescendantPolicy,
) -> io::Result<()> {
    match platform {
        "linux" => Ok(()),
        "macos" if policy == crate::DescendantPolicy::Orchestration || (uid != 0 && euid != 0) => {
            Ok(())
        }
        "macos" => Err(io::Error::other("Darwin descendant prevention requires a non-root user")),
        _ => Err(io::Error::other("backend process ownership supports Linux and macOS only")),
    }
}

#[cfg(any(target_os = "linux", target_os = "macos"))]
mod supported {
    use super::{Command, ExitStatus, File, Path, calls, io, platform_policy_with_descendants};
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
            Self::spawn_with_policy(
                command,
                fallback_directory,
                out,
                err,
                crate::DescendantPolicy::Checker,
            )
        }

        pub(crate) fn spawn_with_policy(
            command: &Command,
            fallback_directory: &Path,
            out: &File,
            err: &File,
            policy: crate::DescendantPolicy,
        ) -> io::Result<Self> {
            let lease = crate::JvmLease::inherited_file()?;
            Self::spawn_with_custody(
                command,
                fallback_directory,
                (out, err),
                policy,
                lease.as_ref(),
                true,
            )
        }

        pub(crate) fn spawn_with_custody(
            command: &Command,
            fallback_directory: &Path,
            (out, err): (&File, &File),
            policy: crate::DescendantPolicy,
            lease: Option<&File>,
            inherit_environment: bool,
        ) -> io::Result<Self> {
            platform_policy_with_descendants(
                std::env::consts::OS,
                process::getuid().as_raw(),
                process::geteuid().as_raw(),
                policy,
            )?;
            let delegate_lease =
                lease.is_some() && policy == crate::DescendantPolicy::Orchestration;
            let inherited = super::inherited_lifecycle()?;
            let lifecycle = policy == crate::DescendantPolicy::Orchestration || inherited.is_some();
            let mut prepared = Prepared::new(
                command,
                fallback_directory,
                delegate_lease,
                lifecycle && policy == crate::DescendantPolicy::Orchestration,
                inherit_environment,
            )?;
            let (completion_read, completion_write) = io::pipe()?;
            let (broadcast_read, broadcast_write) = io::pipe()?;
            let completion_read = duplicate(&completion_read)?;
            let completion_write = duplicate(&completion_write)?;
            let cancellation = duplicate(
                inherited.as_ref().map_or(broadcast_read.as_fd(), |(_, read)| read.as_fd()),
            )?;
            let root_writer = if lifecycle && inherited.is_none() {
                Some(duplicate(&broadcast_write)?)
            } else {
                None
            };
            let parent_completion =
                inherited.as_ref().map(|(write, _)| duplicate(write)).transpose()?;
            let (owner_read, owner_write) = io::pipe()?;
            let null = File::open("/dev/null")?;
            let input = duplicate(&owner_read)?;
            let output = duplicate(out)?;
            let errors = duplicate(err)?;
            let null = duplicate(&null)?;
            let lease = lease.map(duplicate).transpose()?;
            let context = prepared.context(
                [input.as_raw_fd(), output.as_raw_fd(), errors.as_raw_fd(), null.as_raw_fd()],
                policy,
                lease.as_ref().map_or(-1, AsRawFd::as_raw_fd),
                delegate_lease,
                (
                    lifecycle,
                    [
                        completion_read.as_raw_fd(),
                        completion_write.as_raw_fd(),
                        cancellation.as_raw_fd(),
                        root_writer.as_ref().map_or(-1, AsRawFd::as_raw_fd),
                        parent_completion.as_ref().map_or(-1, AsRawFd::as_raw_fd),
                    ],
                ),
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
        rustix::io::fcntl_dupfd_cloexec(fd, 12).map_err(Into::into)
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
        fn new(
            command: &Command,
            fallback_directory: &Path,
            delegate_lease: bool,
            delegate_lifecycle: bool,
            inherit_environment: bool,
        ) -> io::Result<Self> {
            let api = calls::Api::new()?;
            let directory =
                command.get_current_dir().unwrap_or(fallback_directory).canonicalize()?;
            let mut environment = environment(command, inherit_environment)?;
            environment.remove(OsStr::new(super::LEASE_ENV));
            if delegate_lease {
                environment.insert(super::LEASE_ENV.into(), super::LEASE_FD.to_string().into());
            }
            for name in [super::COMPLETION_ENV, super::CANCELLATION_ENV] {
                environment.remove(OsStr::new(name));
            }
            if delegate_lifecycle {
                environment.insert(super::COMPLETION_ENV.into(), "10".into());
                environment.insert(super::CANCELLATION_ENV.into(), "11".into());
            }
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
            [input, output, errors, null]: [RawFd; 4],
            policy: crate::DescendantPolicy,
            lease: RawFd,
            delegate_lease: bool,
            (lifecycle, lifecycle_fds): (bool, [RawFd; 5]),
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
                lease,
                delegate_lease,
                lifecycle,
                lifecycle_fds,
                allow_descendants: policy == crate::DescendantPolicy::Orchestration,
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

    fn environment(command: &Command, inherit: bool) -> io::Result<BTreeMap<OsString, OsString>> {
        let mut environment =
            if inherit { std::env::vars_os().collect::<BTreeMap<_, _>>() } else { BTreeMap::new() };
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
        lease: RawFd,
        delegate_lease: bool,
        lifecycle: bool,
        lifecycle_fds: [RawFd; 5],
        allow_descendants: bool,
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
            prepare_guardian_custody(context);
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
            if context.lifecycle {
                let (cancelled, failed) = pipe_closed(api, 6);
                if cancelled || failed {
                    api.exit(125);
                }
            }
            let target = fork_without_handlers(api);
            if target < 0 {
                child_error(api, c"cannot fork backend subprocess".as_ptr());
            }
            if target == 0 {
                target_main(context, original_mask);
            }
            if context.lifecycle {
                calls::close(api, 5);
            }
            let (mut status, mut completed, mut failed) = if context.lifecycle {
                wait_for_lifecycle(context, target)
            } else {
                wait_for_target(context, target)
            };
            if api.kill(-target, libc::SIGKILL) != 0 && errno(api) != libc::ESRCH {
                failed = true;
            }
            {
                if !completed && api.kill(target, libc::SIGKILL) != 0 && errno(api) != libc::ESRCH {
                    failed = true;
                }
                loop {
                    let result = calls::waitpid(api, target, &raw mut status, 0);
                    if result == target {
                        completed |= context.lifecycle;
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

    unsafe fn prepare_guardian_custody(context: &ChildContext) {
        unsafe {
            let api = &*context.api;
            if context.lease >= 0 && api.dup2(context.lease, super::LEASE_FD) < 0 {
                child_error(api, c"cannot retain inherited lease custody".as_ptr());
            }
            if context.lifecycle {
                let mut index = 0;
                while index < 5 {
                    let source = *context.lifecycle_fds.as_ptr().add(index);
                    if source >= 0 && api.dup2(source, 4 + index as RawFd) < 0 {
                        child_error(api, c"cannot retain lifecycle custody".as_ptr());
                    }
                    index += 1;
                }
            }
            close_guardian_descriptors(api, context);
        }
    }

    unsafe fn pipe_closed(api: &calls::Api, fd: RawFd) -> (bool, bool) {
        unsafe {
            let mut poll = libc::pollfd { fd, events: libc::POLLIN, revents: 0 };
            let result = calls::poll_owner(api, &raw mut poll);
            (
                poll.revents & libc::POLLHUP != 0,
                (result < 0 && errno(api) != libc::EINTR)
                    || poll.revents & (libc::POLLERR | libc::POLLNVAL | libc::POLLIN) != 0,
            )
        }
    }

    unsafe fn wait_for_lifecycle(
        context: &ChildContext,
        target: libc::pid_t,
    ) -> (libc::c_int, bool, bool) {
        unsafe {
            let api = &*context.api;
            let mut cancelled = false;
            let mut broadcast_closed = false;
            let mut failed = false;
            loop {
                let (exited, wait_failed) = calls::target_exited(api, target);
                let (owner_closed, owner_failed) = pipe_closed(api, 0);
                let (root_cancelled, root_failed) = pipe_closed(api, 6);
                let (cohort_closed, cohort_failed) = pipe_closed(api, 4);
                failed |= wait_failed || owner_failed || root_failed || cohort_failed;
                let stop = STOP_SIGNAL.load(Ordering::Relaxed);
                if (exited || owner_closed || root_cancelled || stop != 0 || failed) && !cancelled {
                    cancelled = true;
                    if context.lifecycle_fds[3] >= 0 {
                        calls::close(api, 7);
                        broadcast_closed = true;
                    }
                    let signal =
                        if context.allow_descendants { libc::SIGINT } else { libc::SIGKILL };
                    if api.kill(-target, signal) != 0 && errno(api) != libc::ESRCH {
                        failed = true;
                    }
                }
                #[cfg(target_os = "linux")]
                if exited {
                    // Linux retains stronger subreaper ownership: adopted raw
                    // children can keep FD10 open without observing cancellation.
                    // The caller kills and reaps every descendant before releasing
                    // this guardian's parent completion writer or lease.
                    return (0, false, failed);
                }
                // Trusted carriers retain FD10 until their children finish; a
                // nested guardian retains its parent's writer until its own
                // target wait and cohort drain. EOF alone is not a reap claim.
                if cohort_closed && (exited || cancelled) {
                    if context.lifecycle_fds[3] >= 0 && !broadcast_closed {
                        calls::close(api, 7);
                    }
                    return (0, false, failed);
                }
            }
        }
    }

    unsafe fn wait_for_target(
        context: &ChildContext,
        target: libc::pid_t,
    ) -> (libc::c_int, bool, bool) {
        unsafe {
            let api = &*context.api;
            let status = 0;
            let mut completed = false;
            let mut failed = false;
            while STOP_SIGNAL.load(Ordering::Relaxed) == 0 {
                let (exited, wait_failed) = calls::target_exited(api, target);
                if exited {
                    completed = true;
                    break;
                }
                if wait_failed {
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
            if context.lifecycle
                && context.allow_descendants
                && (api.dup2(5, super::COMPLETION_FD) < 0
                    || api.dup2(6, super::CANCELLATION_FD) < 0)
            {
                child_error(api, c"cannot delegate lifecycle custody".as_ptr());
            }
            close_target_descriptors(api, context);
            if !context.allow_descendants {
                #[cfg(target_os = "macos")]
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

    unsafe fn close_guardian_descriptors(api: &calls::Api, context: &ChildContext) {
        unsafe {
            let mut fd = 4;
            while fd < 12 {
                let keep = (fd == super::LEASE_FD && context.lease >= 0)
                    || (context.lifecycle && (4..=6).contains(&fd))
                    || (context.lifecycle && fd == 7 && context.lifecycle_fds[3] >= 0)
                    || (context.lifecycle && fd == 8 && context.lifecycle_fds[4] >= 0);
                if !keep {
                    calls::close(api, fd);
                }
                fd += 1;
            }
            close_descriptors(api, 12, context.close_limit);
        }
    }

    unsafe fn close_target_descriptors(api: &calls::Api, context: &ChildContext) {
        unsafe {
            let mut fd = 3;
            while fd < 12 {
                let keep = (fd == super::LEASE_FD && context.delegate_lease)
                    || (context.lifecycle && context.allow_descendants && fd >= 10);
                if !keep {
                    calls::close(api, fd);
                }
                fd += 1;
            }
            close_descriptors(api, 12, context.close_limit);
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
    pub(crate) fn spawn_with_policy(
        _: &Command,
        _: &Path,
        _: &File,
        _: &File,
        policy: crate::DescendantPolicy,
    ) -> io::Result<Self> {
        platform_policy_with_descendants(std::env::consts::OS, 0, 0, policy)?;
        Ok(Self)
    }
    pub(crate) fn spawn_with_custody(
        _: &Command,
        _: &Path,
        _: (&File, &File),
        policy: crate::DescendantPolicy,
        _: Option<&File>,
        _: bool,
    ) -> io::Result<Self> {
        platform_policy_with_descendants(std::env::consts::OS, 0, 0, policy)?;
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
    fn orchestration_allows_darwin_children_without_weakening_checker_admission() {
        for (uid, euid) in [(501, 501), (0, 501), (501, 0), (0, 0)] {
            assert!(
                super::platform_policy_with_descendants(
                    "macos",
                    uid,
                    euid,
                    crate::DescendantPolicy::Orchestration
                )
                .is_ok()
            );
            assert_eq!(
                super::platform_policy_with_descendants(
                    "macos",
                    uid,
                    euid,
                    crate::DescendantPolicy::Checker
                )
                .is_ok(),
                uid != 0 && euid != 0
            );
        }
        assert!(
            super::platform_policy_with_descendants(
                "unsupported",
                501,
                501,
                crate::DescendantPolicy::Orchestration
            )
            .is_err()
        );
    }

    #[test]
    fn unsupported_platform_is_an_explicit_refusal() {
        assert!(platform_policy("unsupported", 501, 501).is_err_and(|e| {
            e.to_string() == "backend process ownership supports Linux and macOS only"
        }));
    }
}
