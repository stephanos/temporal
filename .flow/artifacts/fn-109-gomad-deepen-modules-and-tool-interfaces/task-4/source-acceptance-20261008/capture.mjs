import {createHash} from 'node:crypto';
import {readFileSync,writeFileSync,existsSync,statSync} from 'node:fs';
import {spawnSync} from 'node:child_process';
import {dirname,resolve} from 'node:path';
import {fileURLToPath} from 'node:url';
import assert from 'node:assert/strict';
export const root='/Users/stephan/Workspace/skunkworks/gomad/temporal';
export const out=dirname(fileURLToPath(import.meta.url));
export const stock='/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin';
export const sha=b=>createHash('sha256').update(b).digest('hex');
export const read=p=>readFileSync(resolve(root,p));
export function git(args){const r=spawnSync('git',args,{cwd:root,maxBuffer:64<<20});assert.equal(r.status,0,r.stderr.toString());return r.stdout;}
export function sources(){return Object.fromEntries(git(['ls-files','-z']).toString().split('\0').filter(p=>p&&!p.startsWith('.flow/')&&existsSync(resolve(root,p))&&statSync(resolve(root,p)).isFile()).sort().map(p=>[p,sha(read(p))]));}
export function write(name,value){writeFileSync(resolve(out,name),typeof value==='string'||Buffer.isBuffer(value)?value:JSON.stringify(value,null,2)+'\n',{flag:'wx'});}
if(process.argv[1]===fileURLToPath(import.meta.url)){
 const [label,...argv]=process.argv.slice(2);assert(label&&argv.length&&/^[a-z0-9-]+$/.test(label));
 for(const suffix of ['.json','.stdout','.stderr'])assert(!existsSync(resolve(out,label+suffix)),'receipt exists '+label);
 const env={...process.env},removed=['GOROOT','GOBIN','GOMADSEED','GOMAD3_CHILD_SEED','GOMAD3_SEED'];
 for(const k of Object.keys(env))if(k.startsWith('GOMAD')&&/CAPTURE|SEED/.test(k)&&!removed.includes(k))removed.push(k);
 for(const k of removed)delete env[k];
 Object.assign(env,{GOENV:'off',GOWORK:'off',GOTOOLCHAIN:'local',GOFLAGS:'',GOEXPERIMENT:'nogreenteagc',GOMAXPROCS:'2',GOMAD3_STOCK_GO:stock+'/go',PATH:stock+':'+env.PATH,TMPDIR:'/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX',GOTMPDIR:'/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX'});
 const before=sources(),started=new Date(),result=spawnSync(argv[0],argv.slice(1),{cwd:root,env,timeout:600000,maxBuffer:64<<20}),ended=new Date(),after=sources();
 write(label+'.stdout',result.stdout??'');write(label+'.stderr',result.stderr??'');
 const terminal=(result.stdout?.toString()??'').split('\n').flatMap(l=>{try{return [JSON.parse(l)];}catch{return [];}}).filter(e=>e.Test&&['pass','fail','skip'].includes(e.Action)),top=terminal.filter(e=>!e.Test.includes('/'));
 const receipt={argv,cwd:root,environment:Object.fromEntries(['GOENV','GOWORK','GOTOOLCHAIN','GOFLAGS','GOEXPERIMENT','GOMAXPROCS','GOMAD3_STOCK_GO','PATH','TMPDIR','GOTMPDIR','GOCACHE'].map(k=>[k,env[k]??null])),removed_environment:removed,exit:result.status,signal:result.signal,error:result.error?.message??null,started:started.toISOString(),ended:ended.toISOString(),elapsed_seconds:(ended-started)/1000,sources_before_sha256:sha(JSON.stringify(before)),sources_after_sha256:sha(JSON.stringify(after)),source_changes:[...new Set([...Object.keys(before),...Object.keys(after)])].filter(p=>before[p]!==after[p]),stdout_sha256:sha(result.stdout??''),stderr_sha256:sha(result.stderr??''),tool_sha256:{go:sha(read(stock+'/go')),gofmt:sha(read(stock+'/gofmt')),golangci_lint:sha(read('/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0')),errortype:sha(read('/tmp/fn109-lint-tools.ZdNe1t50/errortype'))},tests:terminal.map(e=>({package:e.Package,test:e.Test,action:e.Action})),top_level_counts:Object.fromEntries(['pass','fail','skip'].map(a=>[a,top.filter(e=>e.Action===a).length]))};
 write(label+'.json',receipt);console.log(JSON.stringify({label,exit:receipt.exit,elapsed_seconds:receipt.elapsed_seconds,source_changes:receipt.source_changes,counts:receipt.top_level_counts}));console.log(result.stdout?.toString().slice(-1600)??'');console.log(result.stderr?.toString().slice(-1600)??'');process.exitCode=result.status??1;
}
