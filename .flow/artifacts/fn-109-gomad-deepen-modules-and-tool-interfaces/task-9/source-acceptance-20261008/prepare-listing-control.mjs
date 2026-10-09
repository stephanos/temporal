import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';
const repo='/Users/stephan/Workspace/skunkworks/gomad/temporal',out=path.dirname(new URL(import.meta.url).pathname),scratch='/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX/task9-listing.YToMJw';
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const history=JSON.parse(fs.readFileSync(path.join(out,'historical-inputs.json')));
const list=spawnSync('git',['ls-files','tools/gomad3','tools/gomad3sim','go.mod','go.sum'],{cwd:repo,encoding:'utf8'});
const extra=spawnSync('git',['ls-files','--others','--exclude-standard','tools/gomad3'],{cwd:repo,encoding:'utf8'});
if(list.status!==0||extra.status!==0)throw Error('source enumeration failed');
const current=path.join(scratch,'current');fs.mkdirSync(current);
const copied=[...new Set((list.stdout+extra.stdout).trim().split('\n'))].sort().filter(p=>fs.existsSync(path.join(repo,p))).map(p=>{
 const source=path.join(repo,p),destination=path.join(current,p);fs.mkdirSync(path.dirname(destination),{recursive:true});fs.copyFileSync(source,destination);return {path:destination,source,sha256:hash(fs.readFileSync(destination))};
});
const fixture=fs.readFileSync(path.join(out,'listing-control_test.go.txt'));
const files=[...copied];
for(const graph of [history.original,history.final,current]){
 const dest=path.join(graph,'tools/gomad3/target/task9_listing_control_test.go');if(fs.existsSync(dest))throw Error('control target already exists');fs.writeFileSync(dest,fixture,{flag:'wx'});files.push({path:dest,sha256:hash(fixture)});
}
const literal={'module/go.mod':'module example.com/task9\n\ngo 1.26.4\n','module/main.go':'package main\nfunc main() {}\n','installation/build-key':'9'.repeat(64)+'\n'};
for(const [relative,content]of Object.entries(literal)){const file=path.join(scratch,relative);fs.mkdirSync(path.dirname(file),{recursive:true});fs.writeFileSync(file,content,{flag:'wx'});files.push({path:file,sha256:hash(Buffer.from(content))});}
const dispatcher=fs.readFileSync(path.join(out,'listing-dispatcher.cjs'));
for(const relative of ['installation/bin/go','installation/builds/'+'9'.repeat(64)+'/bin/go']){const file=path.join(scratch,relative);fs.mkdirSync(path.dirname(file),{recursive:true});fs.writeFileSync(file,dispatcher,{mode:0o700,flag:'wx'});files.push({path:file,sha256:hash(dispatcher)});}
const metadata={scratch,current,original:history.original,final:history.final,fixture_sha256:hash(fixture),dispatcher_sha256:hash(dispatcher),literal,files,scope:'Three coherent graphs with the same additive package-local control only; public ReviewCapabilities and private standard inventory, no compiler/native/provenance substitution.'};
fs.writeFileSync(path.join(out,'listing-control-inputs.json'),JSON.stringify(metadata,null,2)+'\n',{flag:'wx'});
console.log(JSON.stringify({scratch,current,files:files.length}));
