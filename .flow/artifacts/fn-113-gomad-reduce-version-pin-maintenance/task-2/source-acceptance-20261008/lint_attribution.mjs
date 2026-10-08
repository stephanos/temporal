import fs from 'node:fs';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';
const out='.flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-2/source-acceptance-20261008';
const hash=value=>crypto.createHash('sha256').update(value).digest('hex');
const diagnostics=file=>fs.readFileSync(out+'/'+file,'utf8').split('\n').filter(line=>/^\/.*\.go:\d+:\d+: /.test(line));
const before=diagnostics('baseline-lint.log'),after=diagnostics('accepted-lint-unfiltered.log');
const owned=line=>line.includes('/cmd/gomadtool/adapter_regenerate.go:')||line.includes('/upgrade/adapterregen/transaction.go:');
const other=before.filter(line=>!owned(line));
if(JSON.stringify(other)!==JSON.stringify(after)) throw Error('remaining diagnostics differ from unchanged OTHER baseline');
const receipt=JSON.parse(fs.readFileSync(out+'/accepted-lint-unfiltered-receipt.json','utf8'));
const baseline=JSON.parse(fs.readFileSync(out+'/baseline-lint-source.json','utf8'));
const final=JSON.parse(fs.readFileSync(out+'/accepted-lint-unfiltered-source.json','utf8'));
const snapshots=new Map(baseline.map(entry=>[entry.path,entry.sha256]));
const paths=[...new Set(after.map(line=>line.slice(0,line.indexOf('.go:')+3).replace(process.cwd()+'/','')))];
const unchanged=paths.map(path=>({path,before_sha256:snapshots.get(path),after_sha256:final.find(entry=>entry.path===path)?.sha256}));
for(const entry of unchanged.filter(entry=>entry.before_sha256!==entry.after_sha256)) {
  if(entry.path!=='tools/gomad3/deterministicio/profile.go') throw Error('OTHER diagnostic source changed: '+entry.path);
  const old=spawnSync('git',['show','ca6fd855868fac364b69cb31394c87ad2912e623:'+entry.path],{encoding:'utf8'});
  if(old.status!==0) throw Error(old.stderr);
  const extract=source=>{const start=source.indexOf('func mustSpec(');return source.slice(start,source.indexOf('\n}\n',start)+3);};
  const beforeFunction=extract(old.stdout),afterFunction=extract(fs.readFileSync(entry.path,'utf8'));
  if(beforeFunction!==afterFunction) throw Error('existing constructor finding changed');
  entry.unchanged_owner_function={function:'mustSpec',before_sha256:hash(beforeFunction),after_sha256:hash(afterFunction),finding:'existing invariant panic at profile.go:211; admitted target shape extraction is below this unchanged constructor'};
}
fs.writeFileSync(out+'/lint-attribution.json',JSON.stringify({same_command_scope:true,baseline_count:before.length,owned_resolved:before.filter(owned),remaining_count:after.length,exact_other_diagnostics_unchanged:true,other_source_bindings:unchanged,remaining:after,final_source_tree_sha256:receipt.source_tree_sha256,version_added_separately:'accepted-version-lint-receipt.json',baseline_log_sha256:hash(fs.readFileSync(out+'/baseline-lint.log')),final_log_sha256:hash(fs.readFileSync(out+'/accepted-lint-unfiltered.log'))},null,2)+'\n');
console.log(JSON.stringify({baseline:before.length,owned_resolved:before.filter(owned).length,remaining:after.length,unchanged_other_files:paths.length}));
