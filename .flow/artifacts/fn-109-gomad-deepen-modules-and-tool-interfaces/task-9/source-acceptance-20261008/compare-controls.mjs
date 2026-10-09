import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import vm from 'node:vm';
const out=path.dirname(new URL(import.meta.url).pathname),task=path.dirname(out);
const bytes=name=>fs.readFileSync(path.join(out,name+'.stdout'));
const hash=data=>crypto.createHash('sha256').update(data).digest('hex');
const rows=name=>bytes(name).toString().trim().split('\n').map(JSON.parse);
const differences=(a,b)=>a.flatMap((row,i)=>JSON.stringify(row)===JSON.stringify(b[i])?[]:[row.Name??i]);
const summary={source_manifest:JSON.parse(fs.readFileSync(path.join(out,'isolated-candidate-build-receipt.json'))).source,telemetry:'Same recorded isolated TEST_TELEMETRY_DIR across all three coherent graphs; original controls retained separately.',public:{},outcomes:{}};
const publicNames=['isolated-matched-public-original','isolated-matched-public-historical-final','isolated-candidate-public-fresh','isolated-candidate-public-cached'];
summary.public.outputs=publicNames.map(name=>({name,bytes:bytes(name).length,sha256:hash(bytes(name)),literal_equal_original:bytes(name).equals(bytes(publicNames[0]))}));
summary.public.normalization='Only unchanged fixture clearing Prepared.Path; no comparison normalization.';
for(const kind of ['download','query','build']) {
 const original=rows('isolated-matched-'+kind+'-original'),historical=rows('isolated-matched-'+kind+'-historical-final'),current=rows('isolated-candidate-'+kind);
 const normalize=values=>kind==='build'?values.map(row=>{const copy=structuredClone(row);delete copy.PID;return copy;}):values;
 summary.outcomes[kind]={cases:original.length,historical_differences:differences(normalize(original),normalize(historical)),current_differences:differences(normalize(original),normalize(current)),normalization:kind==='build'?'Actual child PID field only; complete causes/text/types/unwrap/Is/As/ProcessState/stderr/result/cache/cleanup retained.':'None; complete literal observations.',original_sha256:hash(bytes('isolated-matched-'+kind+'-original')),current_sha256:hash(bytes('isolated-candidate-'+kind))};
 if(kind==='build')Object.assign(summary.outcomes[kind],{started:current.filter(x=>x.BuildStarted).length,all_started_children_gone:current.filter(x=>x.BuildStarted).every(x=>x.ChildGone),all_exclusive_locks_reacquired:current.every(x=>x.ExclusiveLockReacquired),acknowledged_shared_locks:current.filter(x=>['cancel','deadline'].includes(x.Name)).map(x=>({name:x.Name,shared:x.SharedLockObserved}))});
}
const normalizerPath=path.join(task,'adapter-integration-20261005/verify-probes.cjs'),normalizer=fs.readFileSync(normalizerPath,'utf8');
const context={path};vm.createContext(context);vm.runInContext(normalizer.slice(normalizer.indexOf('function normalize'),normalizer.indexOf('const first='))+'\nthis.normalize = normalize;',context);
const adapterOriginal=fs.readFileSync(path.join(task,'adapter-command-gap-2026-10-05/base-process-probe/conductor/stdout.jsonl'),'utf8').trim().split('\n').map(JSON.parse),adapterCurrent=rows('isolated-candidate-adapter');
summary.adapter={cases:adapterCurrent.length,differences:differences(context.normalize(adapterOriginal),context.normalize(adapterCurrent)),started:adapterCurrent.filter(x=>x.Started).length,all_started_reaped_and_gopath_removed:adapterCurrent.filter(x=>x.Started).every(x=>x.ChildGone&&x.GOPATHRemoved),normalizer:{path:normalizerPath,sha256:hash(Buffer.from(normalizer))},normalization:'Exact existing normalizer only; no type/text/errno/ordering/signal/exit/context/stderr/digest/cleanup normalization.'};
summary.selection=rows('isolated-candidate-selection').map(row=>row.platform?{platform:row.platform,inventory:row.inventory,digest:row.digest,source_sha256:row.source_sha256,exit:row.command_exit}: {source_bytes:row.source_bytes,source_sha256:row.source_sha256});
summary.scope='Controlled source preservation; simulated successful build-copy and failure-output transport are neither actual compiler/native execution nor a universal public-cause preservation claim.';
fs.writeFileSync(path.join(out,'matched-control-comparison.json'),JSON.stringify(summary,null,2)+'\n',{flag:'wx'});
console.log(JSON.stringify(summary,null,2));
if(summary.public.outputs.some(x=>!x.literal_equal_original)||Object.values(summary.outcomes).some(x=>x.current_differences.length)||summary.adapter.differences.length)process.exitCode=1;
