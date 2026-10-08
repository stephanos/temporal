import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
const repo='/Users/stephan/Workspace/skunkworks/gomad/temporal',out=path.join(repo,'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-7/source-acceptance-20261008');
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const read=name=>JSON.parse(fs.readFileSync(path.join(out,name),'utf8').split('\n').find(line=>line.startsWith('[')));
const observed=read('link-count-diagnostic-tmpfs.log');
if(observed.length!==6)throw Error('wrong diagnostic selection');
for(const mode of ['os.Root.RemoveAll','os.RemoveAll']){
 const phases=observed.filter(o=>o.Mode===mode);
 for(const [index,o] of phases.entries()){
  if(o.RootSys!=='*syscall.Stat_t'||o.PathSys!=='*syscall.Stat_t'||o.RootNlink!==3-index||o.PathNlink!==3-index||o.Remaining.length!==5-2*index)throw Error('filesystem lacks required link metadata');
  if(index>0&&o.Remaining.some(p=>p.includes('/campaign-first/')))throw Error('first removed path remains');
  if(index===2&&o.Remaining.some(p=>p.includes('/campaign-second/')))throw Error('second removed path remains');
 }
}
const inputs=['link-diagnostic.go','tools/gomad3/qualification/set/prune.go','tools/gomad3/qualification/set/prune_test.go','tools/gomad3/artifact/target_pool.go','tools/gomad3/artifact/link_count_unix.go'].map(file=>{const p=file.startsWith('tools/')?path.join(repo,file):path.join(out,file);return {path:p,sha256:hash(fs.readFileSync(p))};});
fs.writeFileSync(path.join(out,'filesystem-admission.json'),JSON.stringify({scope:'one exact portable prune assertion only; no full qualification retry or broad tmpfs admission',tmpdir:'/dev/shm/.fn1097-prune-source-XjdO2ItK',compiler_tmpdir:'/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX',compiler_cache:'/Users/stephan/Workspace/skunkworks/.gomad-fn1129-cache-EseD1r',mount:'tmpfs shm rw,nosuid,nodev,noexec,relatime,size=65536k',available_before_kib:65536,private_mode:'0700',workspace_observations:read('link-count-diagnostic.log'),tmpfs_observations:observed,capability_assertions:'both Sys types *syscall.Stat_t; both Nlink sequences 3/2/1; both removed campaign paths absent; remaining file counts 5/3/1',inputs},null,2)+'\n');
