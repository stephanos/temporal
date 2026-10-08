import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';
const repo='/Users/stephan/Workspace/skunkworks/gomad/temporal';
const out=path.join(repo,'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-7/source-acceptance-20261008');
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const go='/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go';
const anchor='a3b9f80efab9356c0be2080779133337e2471ac0';
let receipt;
if(process.argv[2]==='initialize'){
 const scratch=fs.mkdtempSync('/Users/stephan/Workspace/skunkworks/.gomad-fn1097-callers-');
 const original=path.join(scratch,'original');fs.mkdirSync(original);
 const archive=spawnSync('git',['archive',anchor,'tools/gomad3'],{cwd:repo,maxBuffer:128<<20});if(archive.status!==0)throw Error(archive.stderr);
 const extracted=spawnSync('tar',['-x','--strip-components=2','-C',original],{input:archive.stdout,maxBuffer:1<<20});if(extracted.status!==0)throw Error(extracted.stderr);
 const preimages=JSON.parse(fs.readFileSync(path.join(repo,'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-7/preimages.json')));
 const preserved=[];
 for(const [name,entry] of Object.entries(preimages)){
  const bytes=fs.readFileSync(path.join(repo,entry.preimage));if(hash(bytes)!==entry.sha256)throw Error('preimage hash mismatch '+name);
  if(name.includes('internal/preparation'))continue;
  fs.writeFileSync(path.join(original,path.relative('tools/gomad3',name)),bytes);
  preserved.push({path:name,sha256:hash(bytes)});
 }
 const harness=fs.readFileSync(path.join(repo,'tools/gomad3/runner/preparation_source_test.go'));
 fs.writeFileSync(path.join(original,'runner/preparation_source_test.go'),harness);
 const replacements={},overlayInputs=[];
 for(const name of ['deterministicio/profile.go','deterministicio/adapter_registry.go','target/target.go']){
  const source=path.join(original,name),bytes=fs.readFileSync(source);
  let text=bytes.toString().replaceAll('runtime.GOOS','"linux"').replaceAll('runtime.GOARCH','"amd64"');
  if(!text.includes('runtime.'))text=text.replace('\n\t"runtime"','');
  const target=path.join(scratch,'original-'+name.replaceAll('/','-'));fs.writeFileSync(target,text);
  replacements[source]=target;overlayInputs.push({source,sha256:hash(bytes),overlay:target,overlay_sha256:hash(text)});
 }
 const overlay=path.join(scratch,'original-overlay.json');fs.writeFileSync(overlay,JSON.stringify({Replace:replacements},null,2)+'\n');
 const walk=dir=>fs.readdirSync(dir,{withFileTypes:true}).flatMap(entry=>entry.isDirectory()?walk(path.join(dir,entry.name)):[path.join(dir,entry.name)]);
 const sourceManifest=walk(original).sort().map(p=>({path:path.relative(original,p),sha256:hash(fs.readFileSync(p))}));
 fs.writeFileSync(path.join(out,'original-caller-source.json'),JSON.stringify(sourceManifest,null,2)+'\n');
 receipt={anchor,scratch,original,inputs:path.join(scratch,'inputs'),overlay,overlayInputs,preserved,harness_sha256:hash(harness),original_source_manifest:'original-caller-source.json',original_source_sha256:hash(JSON.stringify(sourceManifest)),scope:'SOURCE-only supported-profile stock cross builds; original actual caller bodies byte-exact; targets never executed'};
 fs.writeFileSync(path.join(out,'caller-controls.json'),JSON.stringify(receipt,null,2)+'\n');
}else{
 receipt=JSON.parse(fs.readFileSync(path.join(out,'caller-controls.json')));
}
if(process.argv[2]==='refresh-harness'){
 const harness=fs.readFileSync(path.join(repo,'tools/gomad3/runner/preparation_source_test.go'));
 fs.writeFileSync(path.join(receipt.original,'runner/preparation_source_test.go'),harness);
 receipt.harness_sha256=hash(harness);
 fs.writeFileSync(path.join(out,'caller-controls.json'),JSON.stringify(receipt,null,2)+'\n');
 const walk=dir=>fs.readdirSync(dir,{withFileTypes:true}).flatMap(entry=>entry.isDirectory()?walk(path.join(dir,entry.name)):[path.join(dir,entry.name)]);
 const sourceManifest=walk(receipt.original).sort().map(p=>({path:path.relative(receipt.original,p),sha256:hash(fs.readFileSync(p))}));
 fs.writeFileSync(path.join(out,'original-caller-source-final.json'),JSON.stringify(sourceManifest,null,2)+'\n');
 receipt.final_source_manifest='original-caller-source-final.json';receipt.final_source_sha256=hash(JSON.stringify(sourceManifest));
 fs.writeFileSync(path.join(out,'caller-controls.json'),JSON.stringify(receipt,null,2)+'\n');
}
if(process.argv[2]==='add-execution-controls'){
 const replacements=JSON.parse(fs.readFileSync(receipt.overlay)).Replace;
 for(const name of ['runner/internal/execution/bootstrap_unix.go','runner/internal/execution/launch_plan_unix.go']){
  const source=path.join(receipt.original,name),bytes=fs.readFileSync(source);
  if((bytes.toString().match(/syscall\.Dup2/g)??[]).length!==1)throw Error('unexpected Dup2 sites '+name);
  let text=bytes.toString().replace('syscall.Dup2','sourceControlUnreachableDup2');
  if(name.endsWith('bootstrap_unix.go'))text+='\nfunc sourceControlUnreachableDup2(int, int) error { panic("SOURCE control reached forbidden target execution") }\n';
  const overlay=path.join(receipt.scratch,'original-'+name.replaceAll('/','-'));fs.writeFileSync(overlay,text);replacements[source]=overlay;
  receipt.overlayInputs.push({source,sha256:hash(bytes),overlay,overlay_sha256:hash(text),scope:'test-only compile compatibility; fail closed if original target execution reached'});
 }
 fs.writeFileSync(receipt.overlay,JSON.stringify({Replace:replacements},null,2)+'\n');
 fs.writeFileSync(path.join(out,'caller-controls.json'),JSON.stringify(receipt,null,2)+'\n');
}
if(process.argv[2]==='reset-inputs'||process.argv[2]==='initialize'){
 if(!receipt.inputs.startsWith(receipt.scratch+'/')||!receipt.scratch.startsWith('/Users/stephan/Workspace/skunkworks/.gomad-fn1097-callers-'))throw Error('invalid owned scratch');
 fs.rmSync(receipt.inputs,{recursive:true,force:true});fs.mkdirSync(receipt.inputs);
 const manifest=[];
 for(const fixture of ['simple','adapter']){
  const module=path.join(receipt.inputs,fixture,'module');fs.mkdirSync(module,{recursive:true});
  const mod=fixture==='simple'?'module example.com/fn1097-preservation\n\ngo 1.27.1\n':fs.readFileSync(path.join(repo,'tools/gomad3/deterministicio/testdata/sprig/go.mod'));
  const main='package main\n\nfunc main() {}\n';fs.writeFileSync(path.join(module,'go.mod'),mod);fs.writeFileSync(path.join(module,'main.go'),main);
  if(fixture==='adapter')fs.copyFileSync(path.join(repo,'tools/gomad3/deterministicio/testdata/sprig/go.sum'),path.join(module,'go.sum'));
  for(const caller of ['explore','plan']){
   const root=path.join(receipt.inputs,fixture,caller,'toolchain'),key='7'.repeat(64),count=path.join(root,'build-count');
   const script=`#!/bin/sh\nset -eu\ncase "$1" in build) printf 'build\\n' >> '${count}';; esac\nGOCACHE='${root}/source-cache' GOOS=linux GOARCH=amd64 exec '${go}' "$@"\n`;
   for(const binary of [path.join(root,'bin/go'),path.join(root,'builds',key,'bin/go')]){fs.mkdirSync(path.dirname(binary),{recursive:true});fs.writeFileSync(binary,script,{mode:0o700});manifest.push({path:binary,sha256:hash(script)});}
   fs.writeFileSync(path.join(root,'build-key'),key+'\n');
  }
  for(const file of fs.readdirSync(module)){const p=path.join(module,file);manifest.push({path:p,sha256:hash(fs.readFileSync(p))});}
 }
 fs.writeFileSync(path.join(out,'caller-fixed-inputs.json'),JSON.stringify({source_profile:'linux/amd64',inputs:manifest,prepared_cache_initial:'absent',source_cache_initial:'absent',build_counts_initial:0,targets_executed:0},null,2)+'\n');
}
if(process.argv[2]==='capture-state'){
 const name=process.argv[3];if(!/^(original|current)$/.test(name))throw Error('invalid caller state name');
 const walk=dir=>fs.existsSync(dir)?fs.readdirSync(dir,{withFileTypes:true}).flatMap(e=>e.isDirectory()?walk(path.join(dir,e.name)):[path.join(dir,e.name)]):[];
 const state=[];
 for(const fixture of ['simple','adapter'])for(const caller of ['explore','plan']){
  const root=path.join(receipt.inputs,fixture,caller,'toolchain'),build=path.join(root,'builds','7'.repeat(64));
  state.push({fixture,caller,source_cache_initial:'absent (owned reset)',source_cache_final:fs.existsSync(path.join(root,'source-cache'))?'present/warm':'absent',source_cache_files:walk(path.join(root,'source-cache')).length,prepared_cache_files:walk(path.join(build,'prepared-targets')).length,build_count:fs.readFileSync(path.join(root,'build-count'),'utf8').split('build\n').length-1});
 }
 fs.writeFileSync(path.join(out,name+'-caller-cache-state.json'),JSON.stringify(state,null,2)+'\n');
}
console.log(JSON.stringify({original:receipt.original,inputs:receipt.inputs,overlay:receipt.overlay}));
