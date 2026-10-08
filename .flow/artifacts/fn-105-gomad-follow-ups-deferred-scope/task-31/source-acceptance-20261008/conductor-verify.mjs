import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {readFileSync,existsSync,statSync} from 'node:fs';
import {spawnSync} from 'node:child_process';
import {dirname,resolve} from 'node:path';
import {fileURLToPath} from 'node:url';

const root='/Users/stephan/Workspace/skunkworks/gomad/temporal';
const out=dirname(fileURLToPath(import.meta.url));
const sha=v=>createHash('sha256').update(v).digest('hex');
const read=p=>readFileSync(resolve(root,p));
const local=p=>readFileSync(resolve(out,p));
const json=p=>JSON.parse(local(p));
function git(args){const r=spawnSync('git',args,{cwd:root,maxBuffer:32<<20});assert.equal(r.status,0,r.stderr.toString());return r.stdout;}
const frozen=json('evidence.json');
assert.equal(sha(local('evidence.json')),'bc386c73af3dd6442ed809ea2c927f51fb0967d5f4becf309005b691435b708c');
git(['merge-base','--is-ancestor',frozen.base_commit,'HEAD']);
assert.equal(frozen.native_qualification,false);
assert.equal(frozen.review_verdict,null);
for(const e of frozen.artifacts){assert.equal(local(e.path).length,e.bytes,e.path);assert.equal(sha(local(e.path)),e.sha256,e.path);}
const tracked=git(['ls-files','-z']).toString().split('\0').filter(p=>p&&!p.startsWith('.flow/')&&existsSync(resolve(root,p))&&statSync(resolve(root,p)).isFile()).sort();
const current=Object.fromEntries(tracked.map(p=>[p,sha(read(p))]));
assert.equal(sha(JSON.stringify(current)),frozen.source_identity_sha256);
function receipt(path,expected){
 const r=json(path),stem=path.replace(/\.json$/,'');
 assert.equal(r.exit,expected,path);assert.equal(r.signal,null,path);assert.equal(r.error,null,path);
 assert.equal(sha(local(stem+'.stdout')),r.stdout_sha256,path);
 assert.equal(sha(local(stem+'.stderr')),r.stderr_sha256,path);
 assert.equal(r.sources_before_sha256,frozen.source_identity_sha256,path);
 assert.equal(r.sources_after_sha256,frozen.source_identity_sha256,path);
 assert.deepEqual(r.source_changes,[],path);
 assert.equal(r.environment.GOENV,'off');assert.equal(r.environment.GOWORK,'off');assert.equal(r.environment.GOTOOLCHAIN,'local');
 assert.equal(sha(read(r.environment.GOMAD3_STOCK_GO)),r.tool_sha256.go);
 assert.equal(sha(read(resolve(dirname(r.environment.GOMAD3_STOCK_GO),'gofmt'))),r.tool_sha256.gofmt);
 const events=local(stem+'.stdout').toString().split('\n').filter(l=>l.startsWith('{')).flatMap(l=>{try{return [JSON.parse(l)];}catch{return [];}});
 const top=events.filter(e=>e.Test&&!e.Test.includes('/')&&['pass','fail','skip'].includes(e.Action));
 assert.deepEqual(r.top_level_counts,Object.fromEntries(['pass','fail','skip'].map(a=>[a,top.filter(e=>e.Action===a).length])),path);
 return r;
}
for(const e of frozen.command_receipts){assert.equal(sha(local(e.path)),e.sha256);assert.deepEqual(receipt(e.path,e.exit).argv,e.argv);}
const labels=['portable-clock-source','portable-bridge-pins','portable-root-activation','portable-clock-manifest'];
const expected=[39,3,4,1];
for(let i=0;i<labels.length;i++)assert.deepEqual(receipt(labels[i]+'.json',0).top_level_counts,{pass:expected[i],fail:0,skip:0});
const source=json('source-proof.json');
assert.equal(source.source_slice.length,42);assert.equal(source.native_qualification,false);
for(const e of [...source.source_slice,...source.references,...source.clock_documents,...source.user_files])assert.equal(sha(read(e.path)),e.sha256,e.path);
assert.equal(git(['rev-parse',source.original_implementation+'^']).toString().trim(),source.original_base);
assert.equal(git(['rev-parse',source.followup+'^']).toString().trim(),source.original_implementation);
for(const e of source.stdlib_fixtures){assert.equal(sha(read(e.path)),e.sha256);assert(read(e.path).equals(git(['show',source.original_implementation+':'+e.path])));}
const prior=JSON.parse(read(source.references[0].path));
assert.equal(prior.exact_bindings.length,87);
for(const e of prior.exact_bindings)assert.equal(sha(read(e.path)),e.sha256,e.path);
assert.equal(sha(read(prior.archive.path)),prior.archive.sha256);
const changed=prior.input_closure.filter(e=>sha(read(e.path))!==e.current_sha256);
assert.equal(prior.input_closure.length,1223);assert.equal(changed.length,1);
assert.equal(changed[0].path,'tools/gomad3/runner/internal/campaign/retained_evidence_test.go');
assert.equal(source.reused.unchanged_closure,1222);
assert.equal(source.reused.changed_since_prior[0].current,sha(read(changed[0].path)));
assert.equal(source.reused.changed_since_prior[0].reused,false);
const rawPath=prior.retained_raw_manifest.path,raw=JSON.parse(read(rawPath));let bytes=0;
for(const e of raw){const stored=read(rawPath.replace('/raw-manifest.json','/raw/')+(e.retained_name??e.name));if(e.retained_name)assert.equal(sha(stored),e.retained_sha256);const original=e.encoding==='utf8-json-string'?Buffer.from(JSON.parse(stored).value):stored;assert.equal(original.length,e.bytes);assert.equal(sha(original),e.sha256);bytes+=original.length;}
assert.equal(raw.length,72);assert.equal(bytes,521317);
for(const e of prior.identity.bindings)assert.equal(sha(read(e.path)),e.sha256);
const transport=json('transport-proof.json');assert.equal(transport.native_qualification,false);
assert.deepEqual(transport.receipts.map(r=>r.exit),[1,0]);
for(const r of transport.receipts){assert.equal(sha(git(['show',r.ref+':'+r.source_path])),r.source_sha256);assert.equal(sha(local(r.label+'.stdout')),r.stdout_sha256);assert.equal(sha(local(r.label+'.stderr')),r.stderr_sha256);assert.deepEqual(r.tests,[{test:transport.fixture_source.function,action:r.exit?'fail':'pass'}]);}
const standards=json('standards-proof.json');assert.equal(standards.findings.length,24);
assert.equal(standards.task_path_findings,0);assert.equal(standards.task_hunk_findings,0);
for(const f of standards.findings){assert.equal(sha(read(f.path)),f.source_file_sha256);const line=read(f.path).toString().split('\n')[f.line-1];assert.equal(sha(line),f.source_line_sha256);assert.equal(git(['blame','-L',f.line+','+f.line,'--porcelain','--',f.path]).toString().split(' ')[0],f.blame_commit);assert.equal(f.in_original_task_path,false);assert.equal(f.introduced_by_pure_r26_commit,false);}
for(const label of ['baseline-validate','configured-original-r26-lint','configured-root-source-lint','configured-errortype'])receipt(label+'.json',0);
assert.deepEqual(standards.format.listed,['tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go']);
assert.equal(sha(local('format-list.stdout')),standards.format.stdout_sha256);
assert.equal(sha(local('format-list.stderr')),standards.format.stderr_sha256);
assert(read(standards.format.listed[0]).equals(git(['show','a936b597b4c62fa50f11a6c16c91111cd52b1ec3:'+standards.format.listed[0]])));
assert.equal(source.forward_workloads.length,8);
for(const w of source.forward_workloads)assert.deepEqual(w.seeds,[11,17]);
const runtime=read('tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go').toString();
assert(runtime.includes('faketime += gomadClockTickDraw()'));assert(!runtime.includes('gomadClockTickOffset'));assert(runtime.includes('gomadClockForward && requestDeadline < faketime'));
const activation=read('tools/gomad3sim/runtime_time_toolchain.go');
assert.equal(sha(activation),'211c01f57125ba62115b1ffce5d2479d3c22116d51a41aefcfb1a576e8b393a9');
assert.deepEqual([...activation.toString().matchAll(/^\/\/go:linkname (.+)$/gm)].map(m=>m[1]),source.bridge_directives);
const fresh=[];
for(let i=0;i<labels.length;i++){const path='conductor-'+labels[i]+'.json';if(existsSync(resolve(out,path))){assert.deepEqual(receipt(path,0).top_level_counts,{pass:expected[i],fail:0,skip:0});fresh.push(path);}}
if(fresh.length){assert.equal(fresh.length,4);receipt('conductor-validate.json',0);}
const modelPath='conductor-model-activation.json';
if(existsSync(resolve(out,modelPath)))assert.deepEqual(receipt(modelPath,0).top_level_counts,{pass:7,fail:0,skip:0});
console.log(JSON.stringify({frozen_artifacts:frozen.artifacts.length,worker_commands:frozen.command_receipts.length,source_bodies:42,runtime_inputs:87,raw_files:72,raw_original_bytes:bytes,unchanged_closure:1222,changed_unreused:changed[0].path,portable:{pass:47,fail:0,skip:0},fresh_conductor_commands:fresh.length?5:0,task_owned_lint_findings:0,inherited_lint_findings:24,native_qualification:false}));
