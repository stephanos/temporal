import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {readFileSync,writeFileSync} from 'node:fs';
import {spawnSync} from 'node:child_process';
import {dirname,resolve} from 'node:path';
import {fileURLToPath} from 'node:url';
const root='/Users/stephan/Workspace/skunkworks/gomad/temporal';
const out=dirname(fileURLToPath(import.meta.url));
const previous='.flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task-2/source-acceptance-20261007';
const compact='.flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task-2/gfield-compact-20261005';
const historic='.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/task-13';
const hash=b=>createHash('sha256').update(b).digest('hex');
const read=p=>readFileSync(resolve(root,p));
const json=p=>JSON.parse(read(p));
const git=args=>{const r=spawnSync('git',args,{cwd:root,maxBuffer:32<<20});assert.equal(r.status,0,r.stderr.toString());return r.stdout;};
const reference=p=>({path:p,sha256:hash(read(p))});
const prior=json(previous+'/evidence.json'),bindings=json(previous+'/raw/bindings.json');
const snapshot=json(resolve(out,'source-hashes.json'));
const tracked=git(['ls-files','-z']).toString().split('\0').filter(p=>p&&snapshot[p]);
assert.deepEqual(tracked.filter(p=>hash(read(p))!==snapshot[p]),[]);
const exactBindings=Object.entries(bindings.scoped_bindings);
for(const [p,digest] of exactBindings)assert.equal(hash(read(p)),digest,p);
assert.equal(exactBindings.length,87);
const priorModuleDelta=git(['diff','--name-only',prior.base_commit,'--','tools/gomad3']).toString().trim();
assert.equal(priorModuleDelta,'');
let rawBytes=0;
const rawManifest=json(previous+'/raw-manifest.json');
for(const f of rawManifest){
  const stored=read(previous+'/raw/'+(f.retained_name??f.name));
  const original=f.encoding==='utf8-json-string'?Buffer.from(JSON.parse(stored).value,'utf8'):stored;
  assert.equal(hash(original),f.sha256,f.name);assert.equal(original.length,f.bytes,f.name);
  if(f.retained_sha256)assert.equal(hash(stored),f.retained_sha256,f.name);
  rawBytes+=original.length;
}
const reused=[];
for(const label of ['source-inventories-pinned','supplemental-source-final','scoped-receipt-binding-final','independent-identity']){
  const o=prior.observations.find(o=>o.label===label);assert(o,label);
  assert.equal(hash(read(o.receipt)),o.receipt_sha256);
  const r=json(o.receipt);assert.equal(r.exit,0);assert.deepEqual(r.source_changes,[]);
  for(const stream of ['stdout','stderr'])assert.equal(hash(read(previous+'/raw/'+label+'.'+stream)),r[stream+'_sha256']);
  reused.push({label,receipt:reference(o.receipt),argv:r.argv,exit:0,scope:'87 current exact inputs; materialized runtime/static/identity preservation only'});
}
const runtime='tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go';
const first='a3b9f80efab9356c0be2080779133337e2471ac0';
const pre='d635e23f00d926a43b942f25a9d05bd0ccb72025';
const compactBase='1b0bc277589d141aca8b534b03135ab3e57fc050';
const old=git(['show',first+':'+runtime]).toString(),now=read(runtime).toString();
const body=(s,name)=>{const text=s.match(new RegExp('func '+name+'\\([\\s\\S]*?\\n\\}'))?.[0];assert(text,name);return text;};
const originalBody=body(old,'gomadChoiceRunqIndex'),currentBody=body(now,'gomadChoiceRunqIndex');
assert.equal(originalBody.replaceAll('gomadIdentity','gomadID'),currentBody);
assert.equal(originalBody.split('gomadIdentity').length-1,1);
assert(!body(git(['show',pre+':'+runtime]).toString(),'gomadChoiceRunqIndex').includes('isSystemGoroutine'));
assert.equal(body(git(['show',compactBase+':'+runtime]).toString(),'gomadChoiceRunqIndex').replaceAll('gomadIdentity','gomadID'),currentBody);
const unchanged=[];
const originalHashes=json(historic+'/integrated-source-hashes.json');
for(const p of ['tools/gomad3/internal/gomadtool/conformance/runtime_owned.go','tools/gomad3/internal/gomadtool/conformance/runtime_owned_test.go','tools/gomad3/internal/gomadtool/conformance/testdata/runtime_owned/main.go','tools/gomad3/internal/gomadtool/conformance/testdata/runtime_owned/previous-controller.json']){
  assert.equal(hash(read(p)),originalHashes.sources[p],p);
  assert(read(p).equals(git(['show',first+':'+p])),p);
  unchanged.push(reference(p));
}
const compactContext=[];
for(const p of ['tools/gomad3/internal/gomadtool/conformance/runtime_owned.go','tools/gomad3/internal/gomadtool/conformance/runtime_owned_test.go','tools/gomad3/internal/gomadtool/conformance/runtime_scheduling.go','tools/gomad3/internal/gomadtool/conformance/testdata/runtime_owned/main.go','tools/gomad3/internal/gomadtool/conformance/testdata/runq_user_choice/main.go','tools/gomad3/SPEC.md']){
  assert(read(p).equals(git(['show',compactBase+':'+p])),p);
  compactContext.push(reference(p));
}
const preservation=json(compact+'/preservation.json');
assert.equal(preservation.total_renames,30);assert.equal(preservation.files.length,21);assert(preservation.files.every(f=>f.equal));
assert.equal(hash(read(compact+'/preservation.json')),bindings.preservation_receipt_sha256);
const archive='tools/gomad3/.toolchain/downloads/go1.27.1.src.tar.gz';
assert.equal(hash(read(archive)),bindings.archive_sha256);
assert.equal(bindings.archive_sha256,'4e408abae126d916b6164627193f2c54f0e3ca1312d693b86db45f862ab238b1');
const measurement=json(historic+'/same-source-counts.json');
assert.equal(measurement.length,64);
assert.equal(hash(read(historic+'/measurement-main.go')),measurement[0].source_sha256);
const measured={};
for(const version of ['before','after']){
  const rows=measurement.filter(r=>r.version===version);assert.equal(rows.length,32);assert.equal(new Set(rows.map(r=>r.seed)).size,32);assert.equal(new Set(rows.map(r=>r.source_sha256)).size,1);assert.equal(new Set(rows.map(r=>r.build_key)).size,1);
  measured[version]={build_key:rows[0].build_key,source_sha256:rows[0].source_sha256,seeds:32,decisions:rows.reduce((n,r)=>n+r.decisions,0),runtime_selected:rows.reduce((n,r)=>n+r.runtime_selected,0)};
}
assert.equal(measured.before.decisions,2261);assert.equal(measured.after.decisions,2080);assert.equal(measured.before.runtime_selected,31);assert.equal(measured.after.runtime_selected,0);
const observations=[];
for(const label of ['validate','portable-r9','architecture','task-owned-lint','task-owned-format']){
  const r=json(resolve(out,label+'.json')),encoded=json(resolve(out,r.output_file));assert.equal(r.exit,0);assert.equal(r.signal,null);assert.equal(r.error,null);assert.deepEqual(r.source_changes,[]);
  const stdout=Buffer.from(encoded.stdout,'base64'),stderr=Buffer.from(encoded.stderr,'base64');
  assert.equal(hash(stdout),r.stdout_sha256);assert.equal(hash(stderr),r.stderr_sha256);
  const text=stdout.toString();
  const count={pass:(text.match(/^--- PASS:/gm)??[]).length,fail:(text.match(/^--- FAIL:/gm)??[]).length,skip:(text.match(/^--- SKIP:/gm)??[]).length};
  if(['portable-r9','architecture'].includes(label)){assert(count.pass>0);assert.equal(count.fail,0);assert.equal(count.skip,0);}
  if(label==='portable-r9')assert(text.includes('--- PASS: TestRuntimeOwnedRejectsPreviousController'));
  if(label==='architecture')for(const p of ['darwin/arm64','linux/amd64','linux/arm64'])assert(text.includes('--- PASS: TestHostPackageVet/'+p));
  if(label==='task-owned-format')assert.equal(stdout.length,0,'unformatted owned source');
  observations.push({label,receipt:reference(resolve(out,label+'.json')),output:reference(resolve(out,r.output_file)),argv:r.argv,exit:r.exit,elapsed_seconds:r.elapsed_seconds,test_counts:count});
}
const sourceProof={base_commit:git(['rev-parse','HEAD']).toString().trim(),first_committed_r9:first,committed_pre_r9:pre,historical_snapshot:reference(historic+'/integrated-source-hashes.json'),historical_head_limit:'The head identifies pre-R9 committed code; dirty-snapshot file hashes bind historical executions. The committed HEAD alone does not reproduce the snapshot.',current_runq_body_sha256:hash(currentBody),first_runq_body_sha256:hash(originalBody),alpha_substitution:{from:'gomadIdentity',to:'gomadID',occurrences:1},compact_base:compactBase,current_exact_inputs:87,bindings:reference(previous+'/raw/bindings.json'),preservation:reference(compact+'/preservation.json'),retained_raw:{manifest:reference(previous+'/raw-manifest.json'),files:rawManifest.length,bytes:rawBytes},unchanged_primary_fixtures:unchanged,archive_sha256:bindings.archive_sha256,reused,measured,historical_measurement:reference(historic+'/same-source-counts.json'),native_limit:'Historical counts retain original identities. No current native cause, progress, decision-count, patched-runtime, overlay, replay or full-host execution.',source_edits:[]};
sourceProof.compact_context=compactContext;
sourceProof.prior_module_closure={base_commit:prior.base_commit,command:['git','diff','--name-only',prior.base_commit,'--','tools/gomad3'],changed_paths:[]};
const previousController=json('tools/gomad3/internal/gomadtool/conformance/testdata/runtime_owned/previous-controller.json').Identity;
const identityReport=json(previous+'/raw/independent-identity.stdout');
const currentController=createHash('sha256').update('gomad3-choice-implementation-v2').update(Buffer.from([0])).update(Buffer.from(identityReport.selected_source_fingerprint,'hex')).update(Buffer.from(previousController.ToolchainBuildKey,'hex')).digest('hex');
const oldController=Buffer.from(previousController.ImplementationSHA256).toString('hex');
assert.notEqual(oldController,currentController);
sourceProof.previous_controller={platform:previousController.GOOS+'/'+previousController.GOARCH,fixed_build_key:previousController.ToolchainBuildKey,recorded_implementation_sha256:oldController,current_implementation_at_same_build_key:currentController,current_source_fingerprint:identityReport.selected_source_fingerprint,proof:'fresh TestRuntimeOwnedRejectsPreviousController validates original then rejects current identity while target/platform/build key stay fixed'};
sourceProof.tools=Object.fromEntries(['/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0','/tmp/fn109-lint-tools.ZdNe1t50/errortype','/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go','/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/gofmt'].map(p=>[p,hash(readFileSync(p))]));
writeFileSync(resolve(out,'source-proof.json'),JSON.stringify(sourceProof,null,2)+'\n');
const evidence={task_id:'fn-114-gomad-correct-search-path-defects-and.13',status:'in_progress',base_commit:sourceProof.base_commit,commits:[],tests:observations.map(o=>o.argv.map(a=>JSON.stringify(a)).join(' ')),prs:[],baseline:'green for retained applicable source commands; native Quick gates transferred and not run',host:'linux/arm64 stock Go1.27.1 development',observations,source_proof:reference(resolve(out,'source-proof.json')),source_snapshot:reference(resolve(out,'source-hashes.json')),native_owners:['fn-149.1','fn-149.2','fn-128.1','fn-128.4','fn-128.7'],review_context:{required:['current complete gomadChoiceRunqIndex and full runqget patch path','primary runtime_owned fixture zero/one/two/busy modes, all-alternative matcher and mutual-progress assertions','secondary runq_user_choice finalizer-as-user assertion and registration through requireSearchReproduction','canonical SPEC [RUNTIME.SCHEDULING], controller source identity, preceding-controller refusal','first committed R9 body and compact whole-source preservation; historical dirty-snapshot limits'],pre_existing_annotation:{path:'tools/gomad3/internal/gomadtool/conformance/testdata/runq_user_choice/main.go:4',quote:'run first by a fixed rule',limit:'Secondary fixture header predates this acceptance and conflicts with canonical head-class wording. AGENTS prohibits changing separate comments during evidence-only reconciliation. Assertions use finalizer-as-user classification; primary busy worker has an actual runtime symbol.'}},lint_scope:{packages:['./internal/gomadtool/conformance','./choice/internal/wire'],base:pre,filter:'configured --new-from-rev; excludes inherited untouched findings',errortype:'configured go vet ran after golangci success',aggregate:'Whole-project lint remains fn-109.21-owned; scoped success supplies no unfiltered/whole-project pass'},product_edits:[],terminal_handles:'all owned command handles terminal; conductor owns review, Git and Flow completion'};
writeFileSync(resolve(out,'evidence.json'),JSON.stringify(evidence,null,2)+'\n');
JSON.parse(readFileSync(resolve(out,'evidence.json')));
console.log(JSON.stringify({verified_raw_files:rawManifest.length,verified_raw_bytes:rawBytes,current_exact_inputs:87,unchanged_primary_fixtures:unchanged.length,runq_alpha_equal:true,observations:observations.map(o=>({label:o.label,exit:o.exit,tests:o.test_counts})),source_edits:0}));
