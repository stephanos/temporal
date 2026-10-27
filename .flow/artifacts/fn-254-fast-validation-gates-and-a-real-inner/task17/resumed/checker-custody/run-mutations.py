from pathlib import Path
import subprocess,json,hashlib
r=Path(__file__).resolve().parents[6];out=Path(__file__).resolve().parent
syscall=r/'model/umpire/backends/process/src/syscall.rs';support=r/'model/umpire/backends/process/src/support.rs'
green={path:path.read_bytes() for path in [syscall,support]}
cases=[
 ('mutation-guardian',syscall,'                lease.as_ref(),\n                true,','                lease.as_ref().filter(|_| false),\n                true,','Checker guardian lost inherited lease custody'),
 ('mutation-target',syscall,'                lease.as_ref().map_or(-1, AsRawFd::as_raw_fd),\n                delegate_lease,','                lease.as_ref().map_or(-1, AsRawFd::as_raw_fd),\n                lease.is_some(),','Checker target received FD9'),
 ('mutation-env',syscall,'Prepared::new(command, fallback_directory, delegate_lease, inherit_environment)?','Prepared::new(command, fallback_directory, lease.is_some(), inherit_environment)?','Checker target received lease locator'),
 ('mutation-unheld',support,'        match probe.try_lock() {\n            Ok(()) => return Err(std::io::Error::other("unheld inherited JVM lease refused")),\n            Err(fs::TryLockError::WouldBlock) => (),\n            Err(fs::TryLockError::Error(error)) => return Err(error),\n        }\n','', 'called `Option::unwrap()` on a `None` value'),
]
results=[]
for name,path,before,after,expected in cases:
 original=green[path];text=original.decode();assert text.count(before)==1
 receipt={'name':name,'source':str(path.relative_to(r)),'green_sha256':hashlib.sha256(original).hexdigest(),'expected_runtime_failure':expected}
 try:
  red=text.replace(before,after);path.write_text(red);receipt['red_sha256']=hashlib.sha256(red.encode()).hexdigest()
  status=subprocess.run(['python3',str(out/'run-focused.py'),name],cwd=r).returncode;receipt['exit_code']=status
  assert status!=0 and expected in (out/(name+'.log')).read_text()
 finally:
  path.write_bytes(original);receipt['green_restored']=path.read_bytes()==original;results.append(receipt)
  (out/'mutations.json').write_text(json.dumps(results,indent=2)+'\n')
assert all(path.read_bytes()==body for path,body in green.items())
