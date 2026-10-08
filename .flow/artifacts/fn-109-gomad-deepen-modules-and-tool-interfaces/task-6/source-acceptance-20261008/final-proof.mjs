import assert from 'node:assert/strict';
import {fileURLToPath} from 'node:url';
import {statSync} from 'node:fs';
import {resolve} from 'node:path';
import {build,ref} from './proof.mjs';
import {read,write,git,sha,sources,board} from './capture.mjs';
import {spans} from '../../task-4/source-acceptance-20261008/proof.mjs';
const dir='.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-6/source-acceptance-20261008';
const name=f=>/^func (?:\([^)]*\) )?(\w+)/.exec(f.signature)[1];
export function finalProof(){
 const proof=build(),path='.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/go-interface-changes.md',before=read(dir+'/interface-inventory-before.md').toString(),after=read(path).toString(),heading='## fn-109.6 private executor dependencies (fulfils fn-105.3 D3)';
 const start=before.indexOf('\n\n',before.indexOf(heading))+2,end=before.indexOf('\n\n### Removed exported declarations',start),newStart=after.indexOf('\n\n',after.indexOf(heading))+2,newEnd=after.indexOf('\n\n### Removed exported declarations',newStart);
 assert.equal(before.slice(0,start),after.slice(0,newStart));assert.equal(before.slice(end),after.slice(newEnd));assert(before.slice(start,end).startsWith('**Status: implemented.**'));assert(after.slice(newStart,newEnd).startsWith('**Historical inventory.**'));
 assert(after.slice(newStart,newEnd).includes('9b5ae6b39 on 8364bd6a0'));assert(after.slice(newStart,newEnd).includes('TestRunnerRequestsCompileInExternalModule'));
 proof.inventory.current_duplicate_section_gap={resolved:true,worker_edit:true,conductor_admission:ref('.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.6.md'),before:ref(dir+'/interface-inventory-before.md'),after:ref(path),exact_before_paragraph:before.slice(start,end),exact_after_paragraph:after.slice(newStart,newEnd),all_other_bytes_exact:true,diff:git(['diff','--no-ext-diff','--',path]).toString(),historical_duplicate_entire_block:true};
 const task3='.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-3/source-acceptance-20261008/accepted-coverage-proof.json',accepted=JSON.parse(read(task3));
 for(const f of accepted.current_functions)assert.equal(spans(read(f.path).toString()).find(g=>g.signature===f.signature)?.body,f.body);
 proof.accepted_task3_statistics_restoration={receipt:ref(task3),current_functions:accepted.current_functions,early_migration_rows:accepted.early_migration_rows,all_four_current_function_bytes_exact:true,scope:'Three exact whole-stat assertions restored in two files; subsequent task3 accepted additive characterization is bound separately, not assumed equal to earlier historical proof.'};
 const fixtures=['campaign-options-before.json','campaign-options-request-after.json'];
 proof.canonical_options.original_rows_byte_equivalent=undefined;proof.canonical_options.original_75_rows_structurally_exact=true;
 proof.canonical_options.current_fixture_bytes=fixtures.map((n,i)=>{const current='tools/gomad3/runner/testdata/'+n,old='.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-2/'+(i?'campaign-options-request-after.json':'campaign-options-before-complete.json');assert.equal(sha(read(current)),sha(read(old)));return {current:ref(current),retained_original:ref(old),exact_bytes:true};});
 proof.documentation_edits=[path];proof.source_identity_sha256=sha(JSON.stringify(sources()));proof.board=board();
 const housekeeping=proof.test_function_mapping.find(r=>r.test==='TestCleanupRemovesSupersededFiles'),listed=[...housekeeping.original.body.matchAll(/^\s*"([^"]+)",$/gm)].map(m=>m[1]);assert.equal(listed.length,13);
 proof.housekeeping_current_stat=listed.map(p=>{const path='tools/gomad3/'+p;try{statSync(resolve(path));return {path,exists:true,error:null};}catch(e){return {path,exists:false,error:e.code,syscall:e.syscall};}});for(const row of proof.housekeeping_current_stat){assert.equal(row.exists,false,row.path);assert.equal(row.error,'ENOENT',row.path+' exact stat outcome');}
 proof.imported_helpers=['capture.mjs','proof.mjs'].map(n=>ref(dir+'/'+n)).concat(['task-2/source-acceptance-20261008/owner-chain.mjs','task-4/source-acceptance-20261008/proof.mjs','task-4/source-acceptance-20261008/attribution.mjs','task-5/source-acceptance-20261008/proof.mjs','task-5/source-acceptance-20261008/capture.mjs'].map(n=>ref('.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/'+n)));
 proof.setup_observations={dispatch_binding_path_absent:'.flow/artifacts/fn-112-gomad-v041-runtime-and-deterministic-io/task-5/source-acceptance-20261008/source-binding.json',actual_binding:ref('.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-5/source-acceptance-20261008/source-binding.json'),setup_failure_not_test_failure:true};assert.equal(proof.setup_observations.actual_binding.sha256,'78ca24a19fbb1df331dc2c63221ac17c625686ce59d62b3e63c104b3f015a152');
 return proof;
}
if(process.argv[1]===fileURLToPath(import.meta.url)){write('final-source-proof.json',finalProof());console.log('Exact admitted paragraph, accepted assertion restorations, 13 exact ENOENT facts, owner chain and both source sets rebound; no review verdict.');}
