import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';

const repo='/Users/stephan/Workspace/skunkworks/gomad/temporal', out=path.dirname(new URL(import.meta.url).pathname), owner=path.dirname(out);
const base='d635e23f00d926a43b942f25a9d05bd0ccb72025', hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const scratch=fs.mkdtempSync('/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX/task9-history-');
const commands=[];
function run(argv,cwd,input) {
 const start=performance.now(),r=spawnSync(argv[0],argv.slice(1),{cwd,input,maxBuffer:128*1024*1024});
 commands.push({argv,cwd,exit_code:r.status,signal:r.signal,elapsed_seconds:(performance.now()-start)/1000,stdout_sha256:hash(r.stdout??Buffer.alloc(0)),stderr_sha256:hash(r.stderr??Buffer.alloc(0))});
 if(r.status!==0)throw Error(r.stderr?.toString());return r.stdout;
}
const archive=run(['git','archive',base,'tools/gomad3','tools/gomad3sim','go.mod','go.sum'],repo);
for(const name of ['original','historical-final']) {
 const directory=path.join(scratch,name);fs.mkdirSync(directory);run(['tar','-x','-C',directory],repo,archive);
}
const original=path.join(scratch,'original'),final=path.join(scratch,'historical-final');
const preimage=JSON.parse(fs.readFileSync(path.join(owner,'preimage.json'))), frozen=JSON.parse(fs.readFileSync(path.join(owner,'source-freeze.json')));
const checks=[];
for(const [file,expected]of Object.entries(preimage)) {
 const bytes=fs.readFileSync(path.join(original,file)), retained=fs.readFileSync(path.join(owner,'preimage',file));
 if(hash(bytes)!==expected.sha256||hash(bytes)!==hash(retained)||bytes.length!==expected.bytes)throw Error('preimage mismatch '+file);
 checks.push({stage:'original',path:file,sha256:hash(bytes),bytes:bytes.length});
}
const patch=path.join(owner,'task-only.patch');run(['git','apply','--check',patch],final);run(['git','apply',patch],final);
for(const [file,expected]of Object.entries(frozen)) {
 const bytes=fs.readFileSync(path.join(final,file));if(hash(bytes)!==expected.sha256||bytes.length!==expected.bytes)throw Error('historical final mismatch '+file);
 checks.push({stage:'historical-final',path:file,sha256:hash(bytes),bytes:bytes.length});
}
const walk=dir=>fs.readdirSync(dir,{withFileTypes:true}).flatMap(e=>e.isDirectory()?walk(path.join(dir,e.name)):e.isFile()?[path.join(dir,e.name)]:[]);
const sources=Object.fromEntries([['original',original],['historical-final',final]].map(([name,dir])=>[name,walk(dir).sort().map(file=>({path:file,sha256:hash(fs.readFileSync(file))}))]));
const metadata={base,scratch,original,final,archive_sha256:hash(archive),patch:{path:patch,sha256:hash(fs.readFileSync(patch))},checks,sources,commands,root_module_bindings:checks.filter(x=>x.path==='go.mod'),limitations:'Complete coherent historical nested modules and simulation source; exact retained patch only. Root go.mod/go.sum retained in each source manifest. No source/fixture/guard/profile overlay.'};
fs.writeFileSync(path.join(out,'historical-inputs.json'),JSON.stringify(metadata,null,2)+'\n',{flag:'wx'});
console.log(JSON.stringify({scratch,original,final,verified_preimages:5,verified_final_files:8}));
