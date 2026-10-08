import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';
const repo='/Users/stephan/Workspace/skunkworks/gomad/temporal';
const out=path.join(repo,'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-8/source-acceptance-20261008');
const go='/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go';
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const scratch=fs.mkdtempSync('/Users/stephan/Workspace/skunkworks/.gomad-fn1098-source-');
const sourceFiles=['deterministicio/profile.go','deterministicio/adapter_registry.go','target/target.go'];
const original=path.join(scratch,'original/temporal/tools/gomad3');
const current=path.join(scratch,'current/temporal/tools/gomad3');
for(const [name,module] of [['original',original],['current',current]]){
 fs.mkdirSync(module,{recursive:true});
 if(name==='original'){
  const archive=spawnSync('git',['archive','a3b9f80efab9356c0be2080779133337e2471ac0','tools/gomad3'],{cwd:repo,maxBuffer:128<<20});
  if(archive.status!==0)throw Error(archive.stderr);
  const extracted=spawnSync('tar',['-x','--strip-components=2','-C',module],{input:archive.stdout});
  if(extracted.status!==0)throw Error(extracted.stderr);
  for(const [file,sha256] of Object.entries(JSON.parse(fs.readFileSync(path.join(out,'../preimages.json'))))){
   const bytes=fs.readFileSync(path.join(out,'../preimages',file));if(hash(bytes)!==sha256)throw Error('preimage mismatch '+file);
   fs.writeFileSync(path.join(module,path.relative('tools/gomad3',file)),bytes);
  }
  for(const file of ['internal/preparation/inspection.go','internal/preparation/inspection_test.go'])fs.unlinkSync(path.join(module,file));
 }else{
  const paths=spawnSync('git',['ls-files','--cached','--others','--exclude-standard','tools/gomad3'],{cwd:repo,encoding:'utf8'});
  if(paths.status!==0)throw Error(paths.stderr);
  for(const file of paths.stdout.trim().split('\n')){const destination=path.join(module,path.relative('tools/gomad3',file));fs.mkdirSync(path.dirname(destination),{recursive:true});fs.copyFileSync(path.join(repo,file),destination);}
 }
 const repository=path.resolve(module,'../..');
 for(const entry of fs.readdirSync(repo)){if(entry==='tools'||entry==='.git'||entry==='.flow')continue;fs.symlinkSync(path.join(repo,entry),path.join(repository,entry));}
 for(const entry of fs.readdirSync(path.join(repo,'tools'))){if(entry==='gomad3')continue;fs.symlinkSync(path.join(repo,'tools',entry),path.join(repository,'tools',entry));}
 const root=path.join(module,'.toolchain');
 const key='7'.repeat(64);
 const script=`#!/bin/sh\nset -eu\ncase "$1" in build|test|run) echo SOURCE-forbidden-command >&2; exit 97;; esac\nGOCACHE=/Users/stephan/Workspace/skunkworks/.gomad-fn1129-cache-EseD1r GOOS=linux GOARCH=amd64 exec '${go}' "$@"\n`;
 for(const command of [path.join(root,'bin/go'),path.join(root,'builds',key,'bin/go')]){fs.mkdirSync(path.dirname(command),{recursive:true});fs.writeFileSync(command,script,{mode:0o700});}
 fs.writeFileSync(path.join(root,'build-key'),key+'\n');
}
const inputs=[],overlays=[];
for(const [name,module] of [['original',original],['current',current]]){
 const Replace={};
 for(const file of sourceFiles){
  const source=path.join(module,file),bytes=fs.readFileSync(source);
  const occurrences={GOOS:(bytes.toString().match(/runtime\.GOOS/g)??[]).length,GOARCH:(bytes.toString().match(/runtime\.GOARCH/g)??[]).length};
  let text=bytes.toString().replaceAll('runtime.GOOS','"linux"').replaceAll('runtime.GOARCH','"amd64"');
  if(!text.includes('runtime.'))text=text.replace('\n\t"runtime"','');
  const overlay=path.join(scratch,name+'-'+file.replaceAll('/','-'));fs.writeFileSync(overlay,text);Replace[source]=overlay;
  inputs.push({path:source,sha256:hash(bytes),overlay,overlay_sha256:hash(text),occurrences,change:'declared supported linux/amd64 SOURCE inputs; public production guard untouched'});
 }
 const overlay=path.join(scratch,name+'-overlay.json');fs.writeFileSync(overlay,JSON.stringify({Replace},null,2)+'\n');overlays.push(overlay);
}
const walk=dir=>fs.readdirSync(dir,{withFileTypes:true}).flatMap(e=>e.isDirectory()&&e.name!=='.toolchain'?walk(path.join(dir,e.name)):e.isFile()?[path.join(dir,e.name)]:[]);
for(const [name,module] of [['original',original],['current',current]]){
 const manifest=walk(module).sort().map(file=>({path:path.relative(module,file),sha256:hash(fs.readFileSync(file))}));
 if(name==='current'){
  const listing=spawnSync('git',['ls-files','--cached','--others','--exclude-standard','tools/gomad3'],{cwd:repo,encoding:'utf8'});if(listing.status!==0)throw Error(listing.stderr);
  const candidate=listing.stdout.trim().split('\n').map(file=>({path:path.relative('tools/gomad3',file),sha256:hash(fs.readFileSync(path.join(repo,file)))})).sort((a,b)=>a.path.localeCompare(b.path));
  const copied=[...manifest].sort((a,b)=>a.path.localeCompare(b.path));if(JSON.stringify(candidate)!==JSON.stringify(copied))throw Error('current copy inventory differs');
  fs.writeFileSync(path.join(out,'current-copy-equality-'+hash(JSON.stringify(candidate))+'.json'),JSON.stringify({source_files:candidate,copied_files:copied,equal:true,scratch},null,2)+'\n');
 }
 const bytes=JSON.stringify(manifest,null,2)+'\n';fs.writeFileSync(path.join(out,name+'-control-source-'+hash(bytes)+'.json'),bytes);fs.writeFileSync(path.join(out,name+'-control-source.json'),bytes);
}
const controlBytes=JSON.stringify({scratch,original,current,overlays,inputs,profile:'linux/amd64',host:'linux/arm64',native:false,scope:'actual SOURCE computation/listing/adapters/lifetimes; no compiled target launch'},null,2)+'\n';fs.writeFileSync(path.join(out,'source-controls-'+hash(controlBytes)+'.json'),controlBytes);fs.writeFileSync(path.join(out,'source-controls.json'),controlBytes);
console.log(JSON.stringify({scratch,original,current,overlays}));
