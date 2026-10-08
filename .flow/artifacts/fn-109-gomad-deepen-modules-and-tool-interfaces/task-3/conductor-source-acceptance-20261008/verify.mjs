import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {root,out,read,sha,git,sources} from './capture.mjs';
const worker=out.replace('/conductor-source-acceptance-20261008','/source-acceptance-20261008');
assert.equal(sha(read(worker+'/evidence.json')),'fc76d940b79b81927a0c1b8dac3b9ea7a32841eb6a0845be15f5afa3c339b412');
const evidence=JSON.parse(read(worker+'/evidence.json')),frozen=JSON.parse(read(worker+'/freeze-manifest.json'));
const check=spawnSync('node',[worker+'/verify.mjs'],{cwd:root,maxBuffer:64<<20});
assert.equal(check.status,0,check.stderr.toString());
assert.deepEqual(sources(),frozen.current_sources);
for(const commit of git(['rev-list','--reverse',evidence.base_commit+'..HEAD']).toString().trim().split('\n').filter(Boolean)){
 for(const path of git(['diff-tree','-m','--no-commit-id','--name-only','-r','--no-renames',commit]).toString().trim().split('\n').filter(Boolean)){
  if(path.startsWith('.flow/'))continue;
  const bytes=git(['show',commit+':'+path]);
  if(path==='MILESTONES.md'){assert.equal(sha(bytes),frozen.current_sources[path]);continue;}
  const edit=evidence.admitted_test_edits.find(edit=>edit.path===path);
  assert(edit,'unadmitted committed path '+path);
  assert([edit.before_sha256,edit.after_sha256].includes(sha(bytes)),'unadmitted intermediate source '+path);
 }
}
const names=[['campaign-state-machines',0],['final-portable-runner-boundaries',0],['portable-package-architecture',0],['controller-missing-attempted',1],['controller-wrong-success',1],['canonical-controller-after-mutations',0],['final-errortype',0],['final-generated-validation',0],['final-gofmt',0],['final-diff-check',0],['final-configured-fast-lint',2],['final-relevant-unfiltered-lint',2]];
const tests=new Set();
for(const [name,exit] of names){
 const r=JSON.parse(read(out+'/conductor-'+name+'.json')),original=JSON.parse(read(worker+'/'+name+'.json'));
 assert.deepEqual(r.argv,original.argv);assert.equal(r.exit,exit);assert.equal(r.signal,null);assert.equal(r.error,null);
 assert.deepEqual(r.source_changes,[]);assert.equal(r.sources_before_sha256,evidence.source_identity_sha256);assert.equal(r.sources_after_sha256,evidence.source_identity_sha256);
 for(const key of ['GOENV','GOWORK','GOTOOLCHAIN','GOFLAGS','GOEXPERIMENT','GOMAXPROCS','GOMAD3_STOCK_GO'])assert.equal(r.environment[key],original.environment[key]);
 assert.deepEqual(r.tool_sha256,original.tool_sha256);
 const raw=read(out+'/conductor-'+name+'.stdout'),stderr=read(out+'/conductor-'+name+'.stderr');
 assert.equal(sha(raw),r.stdout_sha256);assert.equal(sha(stderr),r.stderr_sha256);
 const terminalBag=tests=>tests.map(test=>JSON.stringify(test)).sort();
 assert.deepEqual(terminalBag(r.tests),terminalBag(original.tests),'identical package-qualified terminal identities, multiplicities and results '+name);
 if(exit===0){assert(!r.tests.some(test=>test.action!=='pass'));for(const test of r.tests)if(!test.test.includes('/'))tests.add(test.package+'::'+test.test);}
 if(name.startsWith('controller-')){assert(raw.toString().includes('controller_completion_test.go:132: statistics ='));assert(r.tests.some(test=>test.test.endsWith('/unclassified_attempt')&&test.action==='fail'));assert.equal(stderr.length,0);}
 if(name==='final-gofmt')assert.equal(raw.length+stderr.length,0);
 if(name.includes('lint')){
  const parse=bytes=>[...bytes.toString().matchAll(/^(tools\/gomad3\/[^:\n]+):(\d+):(\d+): ([^\n]+) \(([^)]+)\)$/gm)].map(match=>match[0]);
  assert.deepEqual(parse(raw),parse(read(worker+'/'+name+'.stdout')),'same actual unfiltered lint locations');
 }
}
assert.equal(tests.size,95);
assert.deepEqual([...tests].sort(),JSON.parse(read(worker+'/accepted-coverage-proof.json')).tests.map(test=>test.package+'::'+test.test).sort());
console.log(JSON.stringify({worker_evidence_sha256:sha(read(worker+'/evidence.json')),source_identity_sha256:evidence.source_identity_sha256,independent_portable_unique:tests.size,root_receipts:names.length,controller_mutations_rejected:2,global_lint_green:false,native:false,review_verdict:null,write_operations:0}));
