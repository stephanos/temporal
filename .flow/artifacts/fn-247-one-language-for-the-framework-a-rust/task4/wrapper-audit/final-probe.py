#!/usr/bin/env python3
import base64,json,subprocess
from pathlib import Path
root=Path('/tmp/fn2474-wrapper-audit')
def command(argv):
 print('COMMAND:',repr(argv),flush=True)
 rc=subprocess.run(argv).returncode
 print('command_rc='+str(rc),flush=True)
 if rc:raise SystemExit(rc)
for name,path in [
 ('linux-mipsn32.tbl','arch/mips/kernel/syscalls/syscall_n32.tbl'),
 ('linux-time_types.h','include/uapi/linux/time_types.h'),
 ('linux-x86.tbl','arch/x86/entry/syscalls/syscall_64.tbl')]:
 url='https://api.github.com/repos/torvalds/linux/contents/'+path+'?ref=v6.12'
 output=root/(name+'.json')
 command(['curl','-fsSL','-o',str(output),url])
 record=json.loads(output.read_text())
 (root/name).write_bytes(base64.b64decode(record['content']))
 print('Decoded API source:',record['html_url'],'blob_sha='+record['sha'],flush=True)
 command(['rg','-n','ppoll|clock_nanosleep|tv_sec|tv_nsec',str(root/name)])
xnu='https://raw.githubusercontent.com/apple-oss-distributions/xnu/f6217f891ac0bb64f3d375211650a4c1ff8ca1ea/'
for name,path,pattern in [
 ('apple-kern_exec.c','bsd/kern/kern_exec.c','For execve case|CLONEPROC_EXEC|new_uthread->uu_sigmask'),
 ('apple-thread.c','osfmk/kern/thread.c','size \+= roundup\(uthread_size|ZC_SEQUESTER \| ZC_ZFREE_CLEARMEM|uthread_init\(parent_task'),
 ('apple-kern_fork.c','bsd/kern/kern_fork.c','^uthread_init\(|uth->uu_flag \|= UT_LP64|uth->uu_sigmask = uth_parent'),
 ('apple-kern_sig.c','bsd/kern/kern_sig.c','^__pthread_canceled\(|UT_CANCELDISABLE|case 2:'),
 ('apple-zalloc.h','osfmk/kern/zalloc.h','Force the created zone|ZC_ZFREE_CLEARMEM'),
 ('apple-resource.h','bsd/sys/resource.h','RLIMIT_NPROC|RLIMIT_POSIX_FLAG')]:
 command(['curl','-fsSL','-o',str(root/name),xnu+path])
 command(['rg','-n',pattern,str(root/name)])
command(['curl','-fsSL','-o',str(root/'apple-pthread.c'),'https://raw.githubusercontent.com/apple-oss-distributions/libpthread/42d026df5b07825070f60134b980a1ec2552dfee/src/pthread.c'])
command(['rg','-n','t->cancel_state =|_pthread_struct_init\(thread',str(root/'apple-pthread.c')])
command(['rg','-n','__pthread_canceled',str(root/'apple-syscalls.master')])
print('All observed fetch/inspection commands exited zero. Darwin conclusions remain source-only; no native/SDK link observation.',flush=True)
