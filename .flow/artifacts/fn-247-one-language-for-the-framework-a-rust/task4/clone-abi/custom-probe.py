#!/usr/bin/env python3
import base64,json,subprocess
from pathlib import Path
root=Path('/tmp/fn2474-clone-abi')
paths=['arch/sparc/kernel/process.c','arch/sparc/kernel/process_32.c','arch/sparc/kernel/process_64.c','arch/sparc/kernel/syscalls/syscall.tbl','arch/mips/kernel/syscall.c','arch/powerpc/kernel/syscalls.c','arch/s390/kernel/syscalls/syscall.tbl','arch/x86/entry/syscalls/syscall_64.tbl']
for path in paths:
 name=path.replace('/','-'); output=root/(name+'.json')
 cmd=['curl','-fsSL','-o',str(output),'https://api.github.com/repos/torvalds/linux/contents/'+path+'?ref=v6.12']
 print('COMMAND:',repr(cmd),flush=True);rc=subprocess.run(cmd).returncode;print('command_rc='+str(rc),flush=True)
 if rc:raise SystemExit(rc)
 x=json.loads(output.read_text());content=base64.b64decode(x['content']).decode();(root/name).write_text(content)
 print('SOURCE:',x['html_url'],'blob_sha='+x['sha'],flush=True)
 for line,text in enumerate(content.splitlines(),1):
  if 'clone' in text.lower():print(str(line)+':'+text,flush=True)
print('All source fetches exited zero.',flush=True)
