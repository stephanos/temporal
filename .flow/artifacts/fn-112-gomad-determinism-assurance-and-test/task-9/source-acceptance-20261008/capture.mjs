import {spawnSync} from 'node:child_process';
import {readFileSync,writeFileSync,existsSync,statSync} from 'node:fs';
import {dirname,resolve} from 'node:path';
import {fileURLToPath} from 'node:url';
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
export const root='/Users/stephan/Workspace/skunkworks/gomad/temporal';
export const out=dirname(fileURLToPath(import.meta.url));
export const stock='/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin';
export const sha=b=>createHash('sha256').update(b).digest('hex');
export const read=p=>readFileSync(resolve(root,p));
export const write=(n,v)=>writeFileSync(resolve(out,n),typeof v==='string'||Buffer.isBuffer(v)?v:JSON.stringify(v,null,2)+'\n',{flag:'wx'});
export function git(args){const r=spawnSync('git',args,{cwd:root,maxBuffer:128<<20});assert.equal(r.status,0,r.stderr?.toString());return r.stdout;}
export function sources(){return Object.fromEntries(git(['ls-files','tools/gomad3','tools/gomad3sim','tools/gomad3integration']).toString().trim().split('\n').filter(p=>existsSync(resolve(root,p))).map(p=>[p,sha(read(p))]));}
export function environment(){
 const env={...process.env},removed=['GOROOT','GOBIN','GOMADSEED','GOMAD3_CHILD_SEED','GOMAD3_SEED'];
 for(const k of Object.keys(env))if(k.startsWith('GOMAD')&&/CAPTURE|SEED/.test(k)&&!removed.includes(k))removed.push(k);
 for(const k of removed)delete env[k];
 const temp='/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX';assert(statSync(temp).isDirectory());
 Object.assign(env,{GOENV:'off',GOWORK:'off',GOTOOLCHAIN:'local',GOFLAGS:'',GOEXPERIMENT:'nogreenteagc',GOMAXPROCS:'2',GOMAD3_STOCK_GO:stock+'/go',PATH:stock+':'+env.PATH,TMPDIR:temp,GOTMPDIR:temp});return {env,removed};
}
if(process.argv[1]===fileURLToPath(import.meta.url)){
 const [label,...argv]=process.argv.slice(2);assert(label&&argv.length&&/^[a-z0-9-]+$/.test(label));
 for(const suffix of ['.json','.stdout','.stderr'])assert(!existsSync(resolve(out,label+suffix)),'existing receipt '+label);
 const {env,removed}=environment(),before=sources(),started=new Date(),mono=process.hrtime.bigint();
 const result=spawnSync(argv[0],argv.slice(1),{cwd:root,env,timeout:600000,maxBuffer:128<<20});
 const after=sources(),elapsed=process.hrtime.bigint()-mono;write(label+'.stdout',result.stdout??'');write(label+'.stderr',result.stderr??'');
 const events=(result.stdout?.toString()??'').split('\n').flatMap(l=>{try{return [JSON.parse(l)];}catch{return [];}}),terminal=events.filter(e=>e.Test&&['pass','fail','skip'].includes(e.Action));
 const counts=t=>Object.fromEntries(['pass','fail','skip'].map(a=>[a,t.filter(e=>e.Action===a).length]));
 const tools={go:stock+'/go',gofmt:stock+'/gofmt',golangci_lint:'/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0',errortype:'/tmp/fn109-lint-tools.ZdNe1t50/errortype'};
 const receipt={label,argv,cwd:root,head:git(['rev-parse','HEAD']).toString().trim(),executable_identity:existsSync(resolve(root,argv[0]))?{path:argv[0],sha256:sha(read(argv[0]))}:null,environment:Object.fromEntries(Object.entries(env).filter(([k])=>/^(GO|GOMAD|TMPDIR|PATH)/.test(k))),removed_environment:removed,exit:result.status,signal:result.signal,error:result.error?.message??null,started:started.toISOString(),ended:new Date().toISOString(),elapsed_seconds:Number(elapsed)/1e9,sources_before_sha256:sha(JSON.stringify(before)),sources_after_sha256:sha(JSON.stringify(after)),source_changes:[...new Set([...Object.keys(before),...Object.keys(after)])].filter(p=>before[p]!==after[p]),stdout_bytes:result.stdout?.length??0,stderr_bytes:result.stderr?.length??0,stdout_sha256:sha(result.stdout??''),stderr_sha256:sha(result.stderr??''),tool_identities:Object.fromEntries(Object.entries(tools).map(([k,p])=>[k,{path:p,sha256:sha(read(p))}])),tests:terminal.map(e=>({package:e.Package,test:e.Test,action:e.Action})),top_level_counts:counts(terminal.filter(e=>!e.Test.includes('/'))),subtest_counts:counts(terminal.filter(e=>e.Test.includes('/'))),package_results:events.filter(e=>!e.Test&&['pass','fail','skip'].includes(e.Action)).map(e=>({package:e.Package,action:e.Action})),native:false};
 write(label+'.json',receipt);console.log(JSON.stringify({label,exit:receipt.exit,error:receipt.error,elapsed_seconds:receipt.elapsed_seconds,source_changes:receipt.source_changes,top:receipt.top_level_counts,subtests:receipt.subtest_counts}));process.exitCode=result.status??1;
}
