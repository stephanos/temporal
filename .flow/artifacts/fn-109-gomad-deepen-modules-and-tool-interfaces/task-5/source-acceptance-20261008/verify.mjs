import assert from 'node:assert/strict';
import {fileURLToPath} from 'node:url';
import {read,sha,out,sources,board,git} from './capture.mjs';
import {walk,build,ref,named,reconstruct} from './proof.mjs';
import {lint} from './lint.mjs';
import {reconcile} from './reconcile.mjs';
export const json=n=>JSON.parse(read(out+'/'+n+'.json'));
export const finalLabels=['restored-portable-cli','restored-portable-runner','restored-architecture-installation','restored-qualification-control','restored-generators','restored-errortype','restored-format-check','restored-diff-check','restored-fast-lint','restored-unfiltered-lint'];
export function receipts(){
 return walk(out).filter(p=>p.endsWith('.json')).flatMap(p=>{const r=JSON.parse(read(p));if(!Array.isArray(r.argv))return [];const label=p.slice(out.length+1,-5),stdout=read(out+'/'+label+'.stdout'),stderr=read(out+'/'+label+'.stderr');
  assert.equal(sha(stdout),r.stdout_sha256,label);assert.equal(sha(stderr),r.stderr_sha256,label);assert.equal(stdout.length,r.stdout_bytes);assert.equal(stderr.length,r.stderr_bytes);assert.deepEqual(r.source_changes,[]);assert.equal(r.sources_before_sha256,r.sources_after_sha256);assert.equal(r.signal,null);assert.equal(r.error,null);assert(Date.parse(r.ended)>=Date.parse(r.started));assert(BigInt(r.monotonic_elapsed_nanos)>=0n);
  const events=stdout.toString().split('\n').flatMap(l=>{try{return [JSON.parse(l)];}catch{return [];}}),terminal=events.filter(e=>e.Test&&['pass','fail','skip'].includes(e.Action)).map(e=>({package:e.Package,test:e.Test,action:e.Action}));assert.deepEqual(terminal,r.tests);const top=terminal.filter(e=>!e.test.includes('/'));assert.deepEqual(Object.fromEntries(['pass','fail','skip'].map(a=>[a,top.filter(e=>e.action===a).length])),r.top_level_counts);assert.deepEqual(terminal.map(e=>e.package+' '+e.test).filter((v,i,a)=>a.indexOf(v)!==i),r.duplicate_terminal_identities);
  assert.deepEqual(events.filter(e=>!e.Test&&['pass','fail','skip'].includes(e.Action)).map(e=>({package:e.Package,action:e.Action})),r.package_results);
  const paths={go:'/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go',gofmt:'/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/gofmt',golangci_lint:'/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0',errortype:'/tmp/fn109-lint-tools.ZdNe1t50/errortype'};for(const [k,path] of Object.entries(paths))assert.equal(sha(read(path)),r.tool_sha256[k]);
  return [{label,receipt:ref(p),...r}];
 });
}
export function coverage(){
 const selected=finalLabels.flatMap(l=>json(l).tests.filter(t=>!t.test.includes('/'))),identities=selected.map(t=>t.package+' '+t.test);assert.equal(selected.length,139);assert.equal(new Set(identities).size,139);assert(selected.every(t=>t.action==='pass'));
 const all=receipts(),observations=all.flatMap(r=>r.tests.map(t=>({...t,label:r.label}))),duplicates=observations.filter((v,i,a)=>a.findIndex(x=>x.package===v.package&&x.test===v.test)!==i);
 return {final_selected:{unique_top_level_pass:139,fail:0,skip:0,labels:finalLabels.slice(0,4),package_qualified_identities:selected},all_observations:observations,duplicate_observations:duplicates,historical_pre_restoration_full_cli:{receipt:ref(out+'/portable-cli-full.json'),pass:89,fail:3,skip:0,bootstrap_package_failure:json('portable-cli-full').package_results.filter(e=>e.action==='fail'&&!json('portable-cli-full').tests.some(t=>t.package===e.package)),current_full_cli_pass_claimed:false},historical_runner_focused:{receipt:ref(out+'/focused-runner.json'),pass:9,fail:4,skip:0,current_full_runner_pass_claimed:false},old_passes_reused:false,native:false};
}
export function redBinding(){
 const original=JSON.parse(read(out+'/sources-current.json')),path='tools/gomad3/cmd/gomad/internal/cli/explore_failure_writer_test.go',body=read(path).toString();original[path]=sha(body);const ordered=Object.fromEntries(Object.keys(original).sort().map(p=>[p,original[p]])),red=json('writer-regression-red-import-corrected');assert.equal(sha(JSON.stringify(ordered)),red.sources_before_sha256);assert.equal(red.exit,1);assert.equal(red.top_level_counts.fail,1);assert.equal(red.tests.filter(e=>e.test.includes('/')&&e.action==='fail').length,2);assert.equal(json('writer-regression-green').exit,0);assert.equal(json('writer-regression-green').top_level_counts.pass,2);
 const first={...ordered,[path]:sha(body.replace('go.temporal.io/server/tools/gomad3/runner','gomad3/runner'))};assert.equal(sha(JSON.stringify(first)),json('writer-regression-red').sources_before_sha256);assert.equal(json('writer-regression-red').tests.length,0);
 return {path,executed_corrected_red_source:body,sha256:sha(body),source_identity_sha256:red.sources_before_sha256,actual_execution_manifest_rederived_from_pre_restoration_manifest_plus_explicit_untracked_input:true,initial_import_setup_source_sha256:first[path],setup_error_is_not_semantic_red:true,literal_three_attempt_byte_expectations:true,semantic_red:ref(out+'/writer-regression-red-import-corrected.json'),green:ref(out+'/writer-regression-green.json')};
}
export function verify(){
 const freeze=json('freeze'),files=walk(out).map(p=>p.slice(out.length+1)).sort();assert.deepEqual(files,[...freeze.files.map(e=>e.name),'freeze.json'].sort());for(const e of freeze.files){const b=read(out+'/'+e.name);assert.equal(b.length,e.bytes,e.name);assert.equal(sha(b),e.sha256,e.name);}
 assert.deepEqual(sources(),json('sources-current-final'));assert.deepEqual(board(),freeze.board);assert.equal(sha(JSON.stringify(sources())),freeze.source_identity_sha256);
 assert.deepEqual(build(),json('source-proof-final'));assert.deepEqual(lint(['restored-fast-lint','restored-unfiltered-lint']),json('lint-attribution-final'));assert.deepEqual(reconcile(),json('lint-reconciliation'));assert.deepEqual(coverage(),json('coverage'));assert.deepEqual(redBinding(),json('regression-red-source'));
 for(const e of json('import-bindings').files)assert.equal(sha(read(e.path)),e.sha256,e.path);
 const status=JSON.parse(read(out+'/terminal-task-status.stdout'));assert.equal(status.status,'in_progress');assert.equal(json('terminal-task-status').exit,0);
 const evidence=json('evidence');assert.deepEqual(evidence.commits,[]);assert.equal(evidence.review_verdict,null);assert.equal(evidence.native,false);assert.equal(evidence.status,'in_progress');assert.equal(evidence.base_commit,'adf32d48330e6768309244b4fc61a5a81ce3e204');
 console.log(JSON.stringify({readonly:true,no_go:true,no_writes:true,files:freeze.files.length,source_identity_sha256:freeze.source_identity_sha256,board:freeze.board,portable_unique_pass:139,skip:0,lint:json('lint-attribution-final').counts,task_status:'in_progress',review_verdict:null,native:false}));
}
if(process.argv[1]===fileURLToPath(import.meta.url))verify();
