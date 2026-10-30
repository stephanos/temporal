import os,time,json,threading,signal,ctypes
from pathlib import Path
class Observer:
 def __init__(self,out):
  self.out=out;self.pid=None;self.seen={};self.events=[];self.stop=threading.Event();self.thread=threading.Thread(target=self.loop,daemon=True);self.thread.start()
 def table(self):
  rows={}
  for p in Path('/proc').iterdir():
   if not p.name.isdigit():continue
   try:
    a=(p/'stat').read_text().rsplit(')',1)[1].split();rows[int(p.name)]=(int(a[1]),a[19],a[0])
   except (OSError,IndexError,ValueError):pass
  return rows
 def detail(self,pid,stat):
  p=Path('/proc')/str(pid);d=dict(pid=pid,ppid=stat[0],start_time=stat[1],state=stat[2])
  try:
   d['argv']=(p/'cmdline').read_bytes().split(b'\0')[:-1];d['argv']=[x.decode(errors='replace') for x in d['argv']]
   raw=(p/'environ').read_bytes().split(b'\0');d['scope_env']={k.decode():v.decode(errors='replace') for x in raw if b'=' in x for k,v in [x.split(b'=',1)] if k in [b'UMPIRE_JVM_LEASE_FD',b'UMPIRE_COMPLETION_FD',b'UMPIRE_CANCELLATION_FD']}
   d['rss_kib']=int(next(x.split()[1] for x in (p/'status').read_text().splitlines() if x.startswith('VmRSS:')))
   d['cwd']=os.readlink(p/'cwd')
   try:
    f=(p/'fd/9').stat();d['fd9']=dict(device=f.st_dev,inode=f.st_ino,target=os.readlink(p/'fd/9'))
   except OSError:d['fd9']=None
  except (OSError,StopIteration):pass
  return d
 def sample(self,log):
  rows=self.table();owned={pid for pid,(start,_) in self.seen.items() if pid in rows and rows[pid][1]==start}
  if self.pid in rows:owned.add(self.pid)
  while True:
   expanded=owned|{pid for pid,s in rows.items() if s[0] in owned}
   if expanded==owned:break
   owned=expanded
  java=[]
  for pid in owned:
   if pid not in rows:continue
   s=rows[pid];key=(pid,s[1]);fresh=pid not in self.seen or self.seen[pid][0]!=s[1]
   d=self.detail(pid,s);self.seen[pid]=(s[1],s[2]);args=d.get('argv',[]);isjava=bool(args) and Path(args[0]).name=='java'
   if fresh:log.write(json.dumps(dict(kind='new-owned-process',at=time.time(),process=d))+'\n');log.flush()
   if isjava:java.append(d)
  if java:log.write(json.dumps(dict(kind='jvm-sample',at=time.time(),jvms=java))+'\n');log.flush()
 def loop(self):
  with (self.out/'owned-process-samples.jsonl').open('w') as log:
   while not self.stop.wait(.2):
    if self.pid:self.sample(log)
 def live(self):
  rows=self.table();return {p:s for p,(start,_) in self.seen.items() if p in rows and (s:=rows[p])[1]==start}
 def finish(self):
  deadline=time.monotonic()+15
  while self.live() and time.monotonic()<deadline:
   try:
    while os.waitpid(-1,os.WNOHANG)[0]:pass
   except ChildProcessError:pass
   time.sleep(.1)
  self.stop.set();self.thread.join();live=self.live();(self.out/'owned-cleanup.json').write_text(json.dumps(dict(live=live,all_observed_owned_pids_absent=not live),indent=2)+'\n');assert not live,live
