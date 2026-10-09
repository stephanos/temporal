import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
const out=path.dirname(new URL(import.meta.url).pathname),v1=JSON.parse(fs.readFileSync(path.join(out,'listing-control-inputs.json'))),scratch=path.join(v1.scratch,'v2');
fs.mkdirSync(scratch);const hash=b=>crypto.createHash('sha256').update(b).digest('hex'),files=[];
for(const relative of ['module/go.mod','module/main.go','installation/build-key']){const file=path.join(scratch,relative),bytes=fs.readFileSync(path.join(v1.scratch,relative));fs.mkdirSync(path.dirname(file),{recursive:true});fs.writeFileSync(file,bytes,{flag:'wx'});files.push({path:file,sha256:hash(bytes)});}
const dispatcher=fs.readFileSync(path.join(out,'listing-dispatcher-v2.cjs'));
for(const relative of ['installation/bin/go','installation/builds/'+'9'.repeat(64)+'/bin/go']){const file=path.join(scratch,relative);fs.mkdirSync(path.dirname(file),{recursive:true});fs.writeFileSync(file,dispatcher,{mode:0o700,flag:'wx'});files.push({path:file,sha256:hash(dispatcher)});}
const metadata={...v1,scratch,version:2,dispatcher_sha256:hash(dispatcher),files:[...v1.files,...files],scope:'Same immutable package-local fixture and three coherent source graphs; versioned checked write-all producer and separate installation paths, preserving every v1 input.'};
fs.writeFileSync(path.join(out,'listing-control-v2-inputs.json'),JSON.stringify(metadata,null,2)+'\n',{flag:'wx'});console.log(JSON.stringify({scratch,dispatcher_sha256:hash(dispatcher)}));
