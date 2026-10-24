#!/usr/bin/env python3
import base64,json,subprocess
from pathlib import Path
root=Path('/tmp/fn2474-clone-abi')
sources=[
 ('https://api.github.com/repos/bminor/glibc/contents/sysdeps/unix/sysv/linux/sparc/sparc64/syscall.S?ref=04e750e75b73957cf1c791535a3f4319534a52fc','sparc64-syscall.S'),
 ('https://api.github.com/repos/bminor/glibc/contents/sysdeps/unix/sysv/linux/sparc/sparc32/syscall.S?ref=04e750e75b73957cf1c791535a3f4319534a52fc','sparc32-syscall.S'),
 ('https://api.github.com/repos/torvalds/linux/contents/kernel/sys.c?ref=v6.12','kernel-sys.c'),
]
for url,name in sources:
 output=root/(name+'.json');cmd=['curl','-fsSL','-o',str(output),url]
 print('COMMAND:',repr(cmd),flush=True);rc=subprocess.run(cmd).returncode;print('command_rc='+str(rc),flush=True)
 if rc:raise SystemExit(rc)
 x=json.loads(output.read_text());content=base64.b64decode(x['content']).decode();(root/name).write_text(content)
 print('SOURCE:',x['html_url'],'blob_sha='+x['sha'],flush=True)
 if name=='kernel-sys.c':
  command=['rg','-n','-A4','SYSCALL_DEFINE0\\(getpid\\)',str(root/name)]
 else:command=['tail','-24',str(root/name)]
 print('COMMAND:',repr(command),flush=True);rc=subprocess.run(command).returncode;print('command_rc='+str(rc),flush=True)
 if rc:raise SystemExit(rc)
for name in ['arch-sparc-kernel-process_32.c','arch-sparc-kernel-process_64.c']:
 command=['rg','-n','-A6','Set the return value for the child',str(root/name)]
 print('COMMAND:',repr(command),flush=True);rc=subprocess.run(command).returncode;print('command_rc='+str(rc),flush=True)
 if rc:raise SystemExit(rc)
print('All source and inspection commands exited zero; no native SPARC/s390x execution.',flush=True)
