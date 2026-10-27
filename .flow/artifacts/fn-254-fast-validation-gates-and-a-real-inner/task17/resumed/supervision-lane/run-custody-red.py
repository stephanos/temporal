from pathlib import Path
import subprocess,hashlib,json
r=Path(__file__).resolve().parents[6];out=Path(__file__).resolve().parent
path=r/'model/umpire/backends/process/src/syscall.rs';green=path.read_bytes();text=green.decode()
needle='            let (status, completed, mut failed) = wait_for_target(context, target);'
assert text.count(needle)==1
red=text.replace(needle,'            if context.lease >= 0 { calls::close(api, super::LEASE_FD); }\n'+needle)
receipt={'mutation':'close FD9 in supervisor only after target fork; target still inherits descriptor','green_sha256':hashlib.sha256(green).hexdigest(),'red_sha256':hashlib.sha256(red.encode()).hexdigest()}
try:
 path.write_text(red)
 result=subprocess.run(['python3',str(out/'run-focused.py'),'custody-red'],cwd=r)
 receipt['exit_code']=result.returncode
 assert result.returncode!=0
 assert 'supervisor lost lease custody' in (out/'custody-red.log').read_text()
finally:
 path.write_bytes(green)
 receipt['green_restored']=path.read_bytes()==green
 (out/'custody-mutation.json').write_text(json.dumps(receipt,indent=2)+'\n')
