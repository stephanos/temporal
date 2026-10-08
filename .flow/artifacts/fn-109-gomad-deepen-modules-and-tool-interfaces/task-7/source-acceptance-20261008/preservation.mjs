import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';
const repo='/Users/stephan/Workspace/skunkworks/gomad/temporal',base=path.join(repo,'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-7'),out=path.join(base,'source-acceptance-20261008');
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const git=(...args)=>{const r=spawnSync('git',args,{cwd:repo,encoding:'utf8'});if(r.status!==0)throw Error(r.stderr);return r.stdout;};
const original=JSON.parse(fs.readFileSync(path.join(out,'caller-controls.json'))),preimages=JSON.parse(fs.readFileSync(path.join(base,'preimages.json'))),preserved=[];
for(const [file,input] of Object.entries(preimages)){
 const bytes=fs.readFileSync(path.join(repo,input.preimage)),digest=hash(bytes);if(digest!==input.sha256||bytes.length!==input.bytes)throw Error('original preimage changed '+file);
 const production=!file.endsWith('_test.go')||file==='tools/gomad3/architecture_test.go';
 const originalCaller=production||file.includes('preparation_equivalence_legacy_test');
 if(originalCaller&&hash(fs.readFileSync(path.join(original.original,path.relative('tools/gomad3',file))))!==digest)throw Error('original caller/probe changed '+file);
 preserved.push({path:input.preimage,sha256:digest,bytes:bytes.length,provenance:input.provenance,original_caller_tree_byte_exact:originalCaller?true:null});
}
const historical=JSON.parse(fs.readFileSync(path.join(base,'handover.json')));
if(hash(fs.readFileSync(path.join(base,'task-only.patch')))!==historical.task_only_patch.sha256)throw Error('historical patch changed');
const snapshots=[];
for(const [name,expected] of [['equivalence','29195da5e9abda68ae04dc414b0c500d0be353c6da9343328668f2039d51b34f'],['equivalence-adapter','f393401b100b6c311cf962f5dd7b3ffcda0355ce5d91050cd788b9c15a7c5aed']]){
 const before=fs.readFileSync(path.join(base,name,'before/snapshot.json')),after=fs.readFileSync(path.join(base,name,'after/snapshot.json'));
 if(!before.equals(after)||hash(before)!==expected)throw Error('historical snapshot changed '+name);
 snapshots.push({path:name,sha256:expected,comparison:'historical byte pair retained; protocol probe, not current actual caller/native evidence'});
}
const before=fs.readFileSync(path.join(out,'original-caller-snapshot.json')),after=fs.readFileSync(path.join(out,'current-caller-snapshot.json')),rows=JSON.parse(before);
if(!before.equals(after)||rows.length!==8||rows.some(r=>r.prepared_from_validated_caller_plan.Path!=='<prepared-path>'||r.build_count!==1))throw Error('actual caller comparison failed');
const bodyFiles=['tools/gomad3/runner/runner.go','tools/gomad3/runner/portable_plan.go','tools/gomad3/runner/deterministicio.go','tools/gomad3/architecture_test.go','tools/gomad3/runner/preparation_equivalence_legacy_test.go','tools/gomad3/runner/preparation_owner_test.go','tools/gomad3/internal/preparation/preparation_test.go'];
const unchanged=bodyFiles.map(file=>{const admitted=hash(git('show','b4602685b3184387cf2d713178095247f0c11d8f:'+file)),current=hash(fs.readFileSync(path.join(repo,file)));if(admitted!==current)throw Error('existing caller/probe/native wrapper changed '+file);return {path:file,sha256:current,unchanged_from_admission:true};});
const preparation=fs.readFileSync(path.join(repo,'tools/gomad3/internal/preparation/preparation.go'),'utf8'),admitted=git('show','b4602685b3184387cf2d713178095247f0c11d8f:tools/gomad3/internal/preparation/preparation.go');
const comments=admitted.split('\n').filter(l=>l.trim().startsWith('//'));if(comments.some(l=>!preparation.includes(l)))throw Error('owning comment removed');
const userFiles=[['.turbo/plans/gomad3-glossary-update.md','97868a86c0a263fbea61c336bd9557e4d71cf43ae4390e9a2449e6f7cd815188'],['.turbo/technical-debt.md','c219247c01fb305592f0314ec46971cee30f00e5985dbd280c1c9e3aafe60287']].map(([file,expected])=>{if(hash(fs.readFileSync(path.join(repo,file)))!==expected)throw Error('user file changed');return {path:file,sha256:expected};});
const runtime=JSON.parse(fs.readFileSync(path.join(out,'runtime-inputs.json')));if(runtime.inputs.some(i=>hash(fs.readFileSync(i.path))!==i.sha256))throw Error('runtime binary/SDK/shim input changed');
const final=JSON.parse(fs.readFileSync(path.join(out,'qualified-prune-runtime-resolved-exact-receipt.json'))),manifest=JSON.parse(fs.readFileSync(path.join(out,final.source_manifest)));
if(manifest.some(i=>i.sha256!==null&&hash(fs.readFileSync(path.join(repo,i.path)))!==i.sha256))throw Error('frozen dependency graph changed');
fs.writeFileSync(path.join(out,'preservation.json'),JSON.stringify({preimages:preserved,task_only_patch:{path:'../task-only.patch',sha256:historical.task_only_patch.sha256},historical_snapshots:snapshots,current_actual_callers:{original:'original-caller-snapshot.json',current:'current-caller-snapshot.json',sha256:hash(before),rows:rows.length,normalization:'Prepared.Path only',explore_prepared_value:'complete value reconstructed from actual validated caller journal; every Prepared field copied',plan_prepared_value:'actual validated openCampaignPlan prepared value',cross_built_targets_executed:0},unchanged_existing_bodies:unchanged,original_comments_retained_in_changed_owner:comments.length,protected_user_files:userFiles,runtime_inputs_unchanged_after_execution:true,frozen_source_manifest:final.source_manifest,frozen_source_tree_sha256:final.source_tree_sha256,final_source_graph_unchanged:true,original_whole_host_make_exit:'unknown; retained wrapper failed assigning reserved status variable',old_watchdog:'transcript-only first failure; no current/native pass claimed',native_scope:'fn149/fn128 deferred and unverified'},null,2)+'\n');
