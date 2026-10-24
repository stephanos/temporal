#!/usr/bin/env python3
import subprocess
from pathlib import Path
root=Path('/tmp/fn2474-wrapper-audit')
base='https://raw.githubusercontent.com/'
glibc='bminor/glibc/04e750e75b73957cf1c791535a3f4319534a52fc/'
xnu='apple-oss-distributions/xnu/f6217f891ac0bb64f3d375211650a4c1ff8ca1ea/'
apple='apple-oss-distributions/Libc/71bbe350ab79eef58113991d817ccc6165061a64/'
platform='apple-oss-distributions/libplatform/2512ffd8bb5c6caff3c8ea83331ab8a0adc820c3/'
pthread='apple-oss-distributions/libpthread/42d026df5b07825070f60134b980a1ec2552dfee/'
paths={
'glibc-open.c':glibc+'sysdeps/unix/sysv/linux/open.c',
'glibc-read.c':glibc+'sysdeps/unix/sysv/linux/read.c',
'glibc-write.c':glibc+'sysdeps/unix/sysv/linux/write.c',
'glibc-close.c':glibc+'sysdeps/unix/sysv/linux/close.c',
'glibc-poll.c':glibc+'sysdeps/unix/sysv/linux/poll.c',
'glibc-wait4.c':glibc+'sysdeps/unix/sysv/linux/wait4.c',
'glibc-waitpid.c':glibc+'posix/waitpid.c',
'glibc-nanosleep.c':glibc+'sysdeps/unix/sysv/linux/nanosleep.c',
'glibc-clock_nanosleep.c':glibc+'sysdeps/unix/sysv/linux/clock_nanosleep.c',
'glibc-dup2.c':glibc+'sysdeps/unix/sysv/linux/dup2.c',
'glibc-setrlimit.c':glibc+'sysdeps/unix/sysv/linux/setrlimit.c',
'glibc-lseek.c':glibc+'sysdeps/unix/sysv/linux/lseek.c',
'glibc-exit.c':glibc+'sysdeps/unix/sysv/linux/_exit.c',
'glibc-errno.c':glibc+'csu/errno-loc.c',
'glibc-aarch64-syscall.S':glibc+'sysdeps/unix/sysv/linux/aarch64/syscall.S',
'glibc-x86_64-syscall.S':glibc+'sysdeps/unix/sysv/linux/x86_64/syscall.S',
'glibc-aarch64-sysdep.h':glibc+'sysdeps/unix/sysv/linux/aarch64/sysdep.h',
'glibc-linux-syscalls.list':glibc+'sysdeps/unix/sysv/linux/syscalls.list',
'glibc-unix-syscalls.list':glibc+'sysdeps/unix/syscalls.list',
'glibc-cancellation.c':glibc+'nptl/cancellation.c',
'glibc-sigaction.c':glibc+'signal/sigaction.c',
'glibc-pthread_sigmask.c':glibc+'nptl/pthread_sigmask.c',
'apple-sigaction.c':apple+'sys/sigaction.c',
'apple-platform-sigaction.c':platform+'src/setjmp/generic/sigaction.c',
'apple-waitpid.c':apple+'gen/FreeBSD/waitpid.c',
'apple-errno.c':xnu+'libsyscall/custom/errno.c',
'apple-tsd.h':xnu+'libsyscall/os/tsd.h',
'apple-kill.c':xnu+'libsyscall/wrappers/kill.c',
'apple-setrlimit.c':xnu+'libsyscall/wrappers/unix03/setrlimit.c',
'apple-syscalls.master':xnu+'bsd/kern/syscalls.master',
'apple-create-syscalls.pl':xnu+'libsyscall/xcodescripts/create-syscalls.pl',
'apple-syscall.map':xnu+'libsyscall/Platforms/syscall.map',
'apple-arm64-syscall.map':xnu+'libsyscall/Platforms/MacOSX/arm64/syscall.map',
'apple-x86-syscall.map':xnu+'libsyscall/Platforms/MacOSX/x86_64/syscall.map',
'apple-fork.s':xnu+'libsyscall/custom/__fork.s',
'apple-SYS.h':xnu+'libsyscall/custom/SYS.h',
'apple-pthread-cancellation.c':pthread+'src/pthread_cancelable.c',
}
failures=[]
pattern=r'SYSCALL_CANCEL|INTERNAL_SYSCALL|INLINE_SYSCALL|sigaction|sigprocmask|cancel|__TSD_ERRNO|__error|errno|_exit|execve|setpgid|setsid|chdir|prctl|dup2|read\$|write\$|close\$|poll\$|wait4|clock_nanosleep|nanosleep|\bsvc\b|\bsyscall\b|\.globl|_os_tsd_get_direct'
for name,path in paths.items():
 cmd=['curl','-fsSL','-o',str(root/name),base+path]
 print('COMMAND:',repr(cmd),flush=True)
 rc=subprocess.run(cmd).returncode
 print('command_rc='+str(rc),flush=True)
 if rc:
  failures.append((name,rc));continue
 cmd=['rg','-n',pattern,str(root/name)]
 print('COMMAND:',repr(cmd),flush=True)
 p=subprocess.run(cmd,text=True,capture_output=True)
 print(p.stdout,end='',flush=True)
 print('command_rc='+str(p.returncode),flush=True)
 if p.returncode not in [0,1]:failures.append((name,p.returncode))
print('fetch_or_tool_failures='+repr(failures),flush=True)
raise SystemExit(bool(failures))
