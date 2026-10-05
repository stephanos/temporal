const fs=require('node:fs'),path=require('node:path');
const root=__dirname;
const load=file=>fs.readFileSync(file,'utf8').trim().split('\n').map(JSON.parse);
function normalize(rows) {
  const base=rows.find(x=>x.Name==='success_valid_json'),scratch=base.Directory,self=base.Command;
  const relative=path.relative('/Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad3',scratch);
  return rows.map(row=>{
    const v=JSON.parse(JSON.stringify(row));
    if(v.Started){delete v.Started.PID;delete v.Started.At;v.Started.GOPATH='<temporary-gopath>';}
    if(v.ExitError)delete v.ExitError.pid;
    const walk=x=>{
      if(typeof x==='string')return x.replaceAll(self,'<probe>').replaceAll(scratch,'<root>').replaceAll(relative,'<relative-root>').replace(/fn109-nonexistent-command-\d+/g,'fn109-nonexistent-command-<pid>');
      if(Array.isArray(x))return x.map(walk);
      if(x&&typeof x==='object')return Object.fromEntries(Object.entries(x).map(([k,v])=>[k,walk(v)]));
      return x;
    };
    return walk(v);
  });
}
const first=load(path.join(root,'process-probe-1/stdout.txt')),second=load(path.join(root,'process-probe-2/stdout.txt'));
const old=load(path.join(root,'../adapter-command-gap-2026-10-05/base-process-probe/conductor/stdout.jsonl'));
const diff=(a,b)=>a.filter((r,i)=>JSON.stringify(r)!==JSON.stringify(b[i])).map(r=>r.Name);
const summary={cases:first.length,repeat_differences:diff(normalize(first),normalize(second)),historical_differences:diff(normalize(first),normalize(old)),started:first.filter(r=>r.Started).length,all_started_reaped_and_gopaths_removed:[first,second].every(rows=>rows.filter(r=>r.Started).every(r=>r.ChildGone&&r.GOPATHRemoved)),normalization:'Exact probe executable, scratch root, relative scratch-root spelling, missing-command generated PID, Started.PID/At/GOPATH and ExitError.pid only. Keep every concrete error type, ordered chain, errno/op, signal/exit, context Is, stderr, quoting, digest and cleanup observation.',prior_analysis:'verification.json did not normalize temporary GOPATH and therefore reported all 18 started cases as differing. Raw observations are unchanged.'};
fs.writeFileSync(path.join(root,'probe-comparison.json'),JSON.stringify(summary,null,2)+'\n',{flag:'wx'});
console.log(JSON.stringify(summary,null,2));
