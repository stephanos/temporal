import {createHash} from 'node:crypto';
import {readFileSync, writeFileSync, existsSync, statSync} from 'node:fs';
import {spawnSync} from 'node:child_process';
import {dirname, resolve} from 'node:path';
import {fileURLToPath} from 'node:url';
const root='/Users/stephan/Workspace/skunkworks/gomad/temporal';
const out=dirname(fileURLToPath(import.meta.url));
const stock='/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin';
const hash=b=>createHash('sha256').update(b).digest('hex');
function git(args) {
  const r=spawnSync('git',args,{cwd:root,maxBuffer:32<<20});
  if(r.status!==0) throw Error(r.stderr.toString());
  return r.stdout;
}
function sources(){return Object.fromEntries(git(['ls-files','-z']).toString().split('\0').filter(p=>p&&!p.startsWith('.flow/')&&existsSync(resolve(root,p))&&statSync(resolve(root,p)).isFile()).sort().map(p=>[p,hash(readFileSync(resolve(root,p)))]));}
const [label,...argv]=process.argv.slice(2);
if(!/^[a-z0-9-]+$/.test(label)||!argv.length||existsSync(resolve(out,label+'.json')))throw Error('invalid or duplicate receipt');
const env={...process.env,GOENV:'off',GOWORK:'off',GOTOOLCHAIN:'local',GOPROXY:'off',GOSUMDB:'off',GOFLAGS:'',GOEXPERIMENT:'nogreenteagc',GOMAXPROCS:'2',GOMAD3_STOCK_GO:stock+'/go',PATH:stock+':'+process.env.PATH};
for(const k of ['GOROOT','GOMADSEED','GOMAD3_CHILD_SEED','GOMAD3_SEED'])delete env[k];
const before=sources(), start=new Date();
if(!existsSync(resolve(out,'source-hashes.json')))writeFileSync(resolve(out,'source-hashes.json'),JSON.stringify(before,null,2)+'\n');
const r=spawnSync(argv[0],argv.slice(1),{cwd:root,env,timeout:610000,maxBuffer:64<<20}),end=new Date(),after=sources();
const stdout=r.stdout??Buffer.alloc(0),stderr=r.stderr??Buffer.alloc(0);
writeFileSync(resolve(out,label+'.output.json'),JSON.stringify({encoding:'base64',stdout:stdout.toString('base64'),stderr:stderr.toString('base64')})+'\n');
const receipt={argv,cwd:root,environment:Object.fromEntries(['GOENV','GOWORK','GOTOOLCHAIN','GOPROXY','GOSUMDB','GOFLAGS','GOEXPERIMENT','GOMAXPROCS','GOMAD3_STOCK_GO','PATH'].map(k=>[k,env[k]])),unset:['GOROOT','GOMADSEED','GOMAD3_CHILD_SEED','GOMAD3_SEED'],base_commit:git(['rev-parse','HEAD']).toString().trim(),started:start.toISOString(),ended:end.toISOString(),elapsed_seconds:(end-start)/1000,exit:r.status,signal:r.signal,error:r.error?.message??null,source_count:Object.keys(before).length,sources_before_sha256:hash(JSON.stringify(before)),sources_after_sha256:hash(JSON.stringify(after)),source_changes:Object.keys(before).filter(p=>before[p]!==after[p]),stdout_sha256:hash(stdout),stderr_sha256:hash(stderr),tool_sha256:{go:hash(readFileSync(stock+'/go')),gofmt:hash(readFileSync(stock+'/gofmt'))},output_file:label+'.output.json'};
writeFileSync(resolve(out,label+'.json'),JSON.stringify(receipt,null,2)+'\n');
console.log(JSON.stringify({label,exit:r.status,signal:r.signal,error:r.error?.message,seconds:receipt.elapsed_seconds,source_changes:receipt.source_changes}));
console.log(stdout.toString().slice(-1800));console.log(stderr.toString().slice(-1000));
process.exitCode=r.status??1;
