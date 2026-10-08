import assert from 'node:assert/strict';
import {root,out,read,git,sha,reconstruct} from './reconstruct.mjs';
import {build,sourceMap} from './proof.mjs';
import {buildDelta} from './delta-proof.mjs';
import {buildLocations} from './lint-location-proof.mjs';
const frozen=JSON.parse(read(out+'/evidence.json'));
git(['merge-base','--is-ancestor',frozen.base_commit,'HEAD']);
const commits=git(['rev-list','--reverse',frozen.base_commit+'..HEAD']).toString().trim().split('\n').filter(Boolean);
for(const commit of commits){const paths=git(['diff-tree','-m','--no-commit-id','--name-only','-r',commit]).toString().trim().split('\n').filter(Boolean);for(const path of paths){assert(path.startsWith('.flow/')||path==='MILESTONES.md','unexpected committed path '+path);if(path==='MILESTONES.md')assert.equal(sha(git(['show',commit+':MILESTONES.md'])),frozen.milestones_ip_sha256,'intermediate board differs from captured IP');}}
assert.equal(sha(read('MILESTONES.md')),frozen.milestones_ip_sha256);
assert.equal(frozen.status,'in_progress');assert.equal(frozen.review_verdict,null);assert.equal(frozen.native_qualification,false);
assert.equal(sha(JSON.stringify(sourceMap())),frozen.source_identity_sha256);
for(const e of frozen.artifacts){const b=read(out+'/'+e.path);assert.equal(b.length,e.bytes,e.path);assert.equal(sha(b),e.sha256,e.path);}
assert.deepEqual(build(),JSON.parse(read(out+'/source-proof.json')));
assert.deepEqual(buildDelta(),JSON.parse(read(out+'/delta-proof.json')));
assert.deepEqual(buildLocations(),JSON.parse(read(out+'/lint-location-proof.json')));
const portable=new Set();
for(const e of frozen.command_receipts){
 const r=JSON.parse(read(out+'/'+e.label+'.json'));assert.equal(sha(read(out+'/'+e.label+'.json')),e.sha256);assert.equal(r.exit,e.exit);assert.equal(r.signal,null);assert.equal(r.error,null);assert.deepEqual(r.argv,e.argv);
 assert.equal(r.sources_before_sha256,frozen.source_identity_sha256);assert.equal(r.sources_after_sha256,frozen.source_identity_sha256);assert.deepEqual(r.source_changes,[]);
 assert.equal(sha(read(out+'/'+e.label+'.stdout')),r.stdout_sha256);assert.equal(sha(read(out+'/'+e.label+'.stderr')),r.stderr_sha256);
 assert.equal(r.environment.GOENV,'off');assert.equal(r.environment.GOWORK,'off');assert.equal(r.environment.GOTOOLCHAIN,'local');assert.equal(r.environment.GOEXPERIMENT,'nogreenteagc');assert.equal(r.environment.GOFLAGS,'');assert.equal(r.environment.GOMAXPROCS,'2');
 assert(r.removed_environment.includes('GOMAD3_CAPTURE_OPTIONS_BASELINE'));assert(r.removed_environment.includes('GOMAD3_CAPTURE_OPTIONS_AFTER'));
 assert.equal(sha(read(r.environment.GOMAD3_STOCK_GO)),r.tool_sha256.go);
 const events=read(out+'/'+e.label+'.stdout').toString().split('\n').flatMap(l=>{try{return [JSON.parse(l)];}catch{return [];}}).filter(v=>v.Test&&['pass','fail','skip'].includes(v.Action));
 assert.deepEqual(events.map(v=>({package:v.Package,test:v.Test,action:v.Action})),r.tests,e.label);
 const top=events.filter(v=>!v.Test.includes('/'));assert.deepEqual(r.top_level_counts,Object.fromEntries(['pass','fail','skip'].map(a=>[a,top.filter(v=>v.Action===a).length])));
 if(frozen.portable_pass_receipts.includes(e.label)){assert.equal(r.exit,0);assert.equal(r.top_level_counts.fail,0);assert.equal(r.top_level_counts.skip,0);assert(r.top_level_counts.pass>0);for(const t of top)portable.add(t.Package+'::'+t.Test);}
}
assert.equal(portable.size,71);assert.deepEqual([...portable].sort(),frozen.unique_portable_top_level_tests);
const guarded=JSON.parse(read(out+'/portable-transport.json'));assert.equal(guarded.exit,1);assert.deepEqual(guarded.top_level_counts,{pass:12,fail:1,skip:0});assert(read(out+'/portable-transport.stdout').toString().includes('host is linux/arm64'));
assert.deepEqual(guarded.tests.filter(t=>t.action==='fail').map(t=>t.test),['TestExecutionEvidenceIgnoresAggregateCoordinatorDeadlineAdjustment']);
const r=reconstruct();assert.equal(r.rows.length,20);assert.equal(r.rows.filter(e=>e.current_equals_original).length,5);
const proof=JSON.parse(read(out+'/source-proof.json'));assert.equal(proof.lint.actual_base_findings.filter(e=>e.task2_introduced_statement).length,0);assert.equal(proof.lint.unfiltered_runner_findings.filter(e=>e.task2_introduced_statement).length,0);
console.log(JSON.stringify({frozen_artifacts:frozen.artifacts.length,commands:frozen.command_receipts.length,unique_portable_top_level_passes:portable.size,guarded_fixture_unexecuted_failures:1,actual_original_postimages:20,unchanged_current:5,changed_attributed:15,private_options_declarations_exact:8,original_baseline_inputs:670,runtime_inputs:87,raw_files:72,task_owned_reported_findings:0,inherited_runner_findings:17,current_original_base_diff_findings:68,adjacent_base_diagnostic_findings:80,native_qualification:false,review_verdict:null}));
