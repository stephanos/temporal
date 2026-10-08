import assert from 'node:assert/strict';
import {readdirSync,existsSync,statSync} from 'node:fs';
import {resolve} from 'node:path';
import {read,sha,git,out,sources} from './capture.mjs';
const proof=JSON.parse(read(out+'/sealed-source-proof.json')),mapping=JSON.parse(read(out+'/final-current-mapping-analysis.json'));
assert.deepEqual(sources(),proof.current_sources);assert.equal(proof.mapping_rows.length,282);assert.equal(mapping.rows.length,282);
assert.equal(mapping.rows.filter(r=>r.replacement==='-').length,11);assert.equal(mapping.rows.filter(r=>r.actual_current_status==='NOT REACHED; ancestor FAIL').length,104);
assert.equal(mapping.extraCauseRows.length,4);assert.equal(proof.accepted_owner_proofs.task6.current_exact_body_bindings,383);assert(proof.accepted_owner_proofs.task3.all_four_exact);
const evidenceNames=readdirSync(out).filter(n=>n.endsWith('.json')),receipts=evidenceNames.flatMap(n=>{let r;try{r=JSON.parse(read(out+'/'+n));}catch{return[];}return r.label&&r.argv?[r]:[];});
function receiptCheck(r){
 assert(Number.isInteger(r.exit)&&r.ended&&r.started&&r.signal===null&&r.error===null,r.label+' terminal');
 assert.equal(r.native,false);assert.equal(sha(read(out+'/'+r.label+'.stdout')),r.stdout_sha256);assert.equal(sha(read(out+'/'+r.label+'.stderr')),r.stderr_sha256);
 assert.deepEqual(r.source_changes,[]);
 for(const tool of Object.values(r.tool_identities))assert.equal(sha(read(tool.path)),tool.sha256);
 if(r.tests.some(t=>t.action==='fail'))assert.notEqual(r.exit,0);
 for(const [kind,predicate] of [['top_level_counts',t=>!t.test.includes('/')],['subtest_counts',t=>t.test.includes('/')]])for(const status of ['pass','fail','skip'])assert.equal(r[kind][status],r.tests.filter(t=>predicate(t)&&t.action===status).length);
 assert(new Date(r.ended)>=new Date(r.started));
}
for(const r of receipts)receiptCheck(r);
const byLabel=Object.fromEntries(receipts.map(r=>[r.label,r]));
for(const label of ['sealed-final-validate','sealed-final-scoped-vet','sealed-final-scoped-errortype','sealed-final-focused-format','sealed-final-checker-controls','sealed-causes-normal','final-world-restored-normal','restored-empty-cache','restored-alias-valid-input'])assert.equal(byLabel[label].exit,0,label);
for(const [label,exit] of [['sealed-root',1],['sealed-final-deterministicio',1],['sealed-final-cli',1],['sealed-final-runner-focused',1],['sealed-final-configured-fast-lint',2],['sealed-final-unfiltered-scoped-lint',2],['sealed-final-nested-format',1],['sealed-causes-mutant',1],['final-world-restored-mutant',1],['restored-empty-mutant-control',1],['restored-alias-rejects-input',1]])assert.equal(byLabel[label].exit,exit,label);
assert.equal(byLabel['sealed-causes-mutant'].subtest_counts.fail,2);assert.equal(byLabel['final-world-restored-mutant'].subtest_counts.fail,1);
const controls=[];for(const [name,change] of [['forged_zero_exit',r=>r.exit=0],['forged_output_hash',r=>r.stdout_sha256='0'.repeat(64)],['native_claim',r=>r.native=true],['lost_named_terminal',r=>r.tests=[]]]){const r=structuredClone(byLabel['sealed-causes-mutant']);change(r);assert.throws(()=>receiptCheck(r));controls.push({name,rejected:true});}
const format=JSON.parse(read(out+'/sealed-final-nested-format.stdout'));assert.equal(format.raw_exit,0);assert.equal(format.formatted,false);assert.equal(format.stdout,'tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go\n');
const before=JSON.parse(read(out+'/source-proof.json')).current_sources;assert.deepEqual(Object.keys(proof.current_sources).filter(p=>before[p]!==proof.current_sources[p]),['tools/gomad3/architecture_test.go','tools/gomad3/deterministicio/adapter_rewrite_test.go','tools/gomad3/runner/completion_test.go']);
const userFiles={'.turbo/plans/gomad3-glossary-update.md':'97868a86c0a263fbea61c336bd9557e4d71cf43ae4390e9a2449e6f7cd815188','.turbo/technical-debt.md':'c219247c01fb305592f0314ec46971cee30f00e5985dbd280c1c9e3aafe60287'};for(const [path,hash] of Object.entries(userFiles)){assert.equal(sha(read(path)),hash);assert.equal(git(['ls-files','--',path]).length,0);}
const staticProof=JSON.parse(read(out+'/final-static-bindings.json'));assert.equal(staticProof.supported_source_sets.length,2);assert.equal(staticProof.host_source_set.length,1);assert.equal(staticProof.unchanged_runtime_inputs,87);
const accounting=JSON.parse(read(out+'/accounting-analysis.json'));assert.equal(accounting.current.delta_top_level,1);
const golden=JSON.parse(read(proof.golden.current.path));assert.equal(golden.length,126);assert.equal(sha(read(proof.golden.current.path)),proof.golden.current.sha256);
const task=JSON.parse(read(out+'/sealed-task-state.stdout'));assert.equal(task.status,'in_progress');
let manifestResult=null;
if(existsSync(resolve(out,'manifest.json'))){const manifest=JSON.parse(read(out+'/manifest.json'));for(const f of manifest.files){assert.equal(sha(read(out+'/'+f.path)),f.sha256,f.path);assert.equal(statSync(resolve(out,f.path)).size,f.bytes);}manifestResult={files:manifest.files.length,sha256:sha(read(out+'/manifest.json'))};}
console.log(JSON.stringify({valid:true,terminal_receipts:receipts.length,current_source_paths:Object.keys(proof.current_sources).length,mapping_rows:282,blocked_child_rows:104,restored_same_input_conditions:4,negative_receipt_controls:controls,user_documents_unchanged_and_untracked:true,manifest:manifestResult,review_verdict:null,native:false,full_host_pass:false}));
