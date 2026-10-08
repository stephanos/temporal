import {createHash} from 'node:crypto';
import {readFileSync,writeFileSync,mkdtempSync,existsSync} from 'node:fs';
import {spawnSync} from 'node:child_process';
import {dirname} from 'node:path';
import {fileURLToPath} from 'node:url';
import assert from 'node:assert/strict';
const root='/Users/stephan/Workspace/skunkworks/gomad/temporal',out=dirname(fileURLToPath(import.meta.url));
const sha=v=>createHash('sha256').update(v).digest('hex');
const go='/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go';
const sourcePath='tools/gomad3/runner/internal/execution/simulation_time.go';
const testPath='tools/gomad3/runner/internal/execution/simulation_time_test.go';
const green='32cc7e6d2ba635fb87b9f53f8b71ede0f9010e28',red='70bb38e5ddec2d271c12f0e8c2489855f08eb0ef';
function git(ref,path){const r=spawnSync('git',['show',ref+':'+path],{cwd:root,maxBuffer:32<<20});assert.equal(r.status,0);return r.stdout;}
const name='TestSimulationTimeArbiterAcceptsForwardTickEpoch';
const original=git(green,testPath).toString().match(new RegExp('func '+name+'\\(t \\*testing.T\\) \\{[^]*?\\n\\}'))?.[0];
assert(original);assert(readFileSync(root+'/'+testPath,'utf8').includes(original));
const adapted=original.replace('newSimulationTimeArbiter(true)','newSimulationTimeArbiter()');
assert.notEqual(original,adapted);
assert(!existsSync(out+'/transport-proof.json'));
const receipts=[];
for(const [label,ref,body,want] of [['transport-original-host-red',red,adapted,1],['transport-original-host-green',green,original,0]]){
 for(const suffix of ['.stdout','.stderr'])assert(!existsSync(out+'/'+label+suffix));
 const tmp=mkdtempSync('/tmp/r26-transport-'),source=git(ref,sourcePath),fixture='package execution\nimport("context";"testing")\n'+body+'\n';
 writeFileSync(tmp+'/simulation_time.go',source,{flag:'wx'});writeFileSync(tmp+'/causal_test.go',fixture,{flag:'wx'});
 const argv=[go,'test','-count=1','-json','-tags','test_dep',tmp+'/simulation_time.go',tmp+'/causal_test.go'];
 const start=new Date(),r=spawnSync(argv[0],argv.slice(1),{cwd:root,env:process.env,timeout:60000,maxBuffer:16<<20}),end=new Date();
 writeFileSync(out+'/'+label+'.stdout',r.stdout??'',{flag:'wx'});writeFileSync(out+'/'+label+'.stderr',r.stderr??'',{flag:'wx'});
 const events=r.stdout.toString().trim().split('\n').map(l=>JSON.parse(l));
 const tests=events.filter(e=>e.Test&&['pass','fail','skip'].includes(e.Action));
 receipts.push({label,ref,source_path:sourcePath,source_sha256:sha(source),fixture_sha256:sha(fixture),argv,cwd:root,exit:r.status,signal:r.signal,error:r.error?.message??null,started:start.toISOString(),ended:end.toISOString(),elapsed_seconds:(end-start)/1000,stdout_sha256:sha(r.stdout),stderr_sha256:sha(r.stderr),tests:tests.map(e=>({test:e.Test,action:e.Action})),observation_scope:'stock host execution of complete historical arbiter source; no patched runtime or native process transport'});
 console.log(r.stdout.toString());assert.equal(r.status,want,r.stderr.toString());assert.equal(tests.length,1);
 assert.equal(tests[0].Action,want===0?'pass':'fail');
 if(want)assert(r.stdout.toString().includes('simulation time request does not match the current epoch'));
}
writeFileSync(out+'/transport-proof.json',JSON.stringify({receipts,fixture_source:{ref:green,path:testPath,function:name,sha256:sha(original),current_body_equal:true},adaptation:'Only the predecessor factory call omits the new forward bool; complete historical implementation sources are unmodified.',historical_native_red_green_provenance:'No original D26 native RED/GREEN execution receipt found in retained Git/artifact paths. These are fresh portable source observations.',native_qualification:false},null,2)+'\n',{flag:'wx'});
