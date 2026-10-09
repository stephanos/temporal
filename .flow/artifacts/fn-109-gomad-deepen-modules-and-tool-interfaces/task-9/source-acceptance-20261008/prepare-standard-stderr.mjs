import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
const out=path.dirname(new URL(import.meta.url).pathname),prior=JSON.parse(fs.readFileSync(path.join(out,'listing-control-v2-inputs.json'))),scratch=path.join(prior.scratch,'standard-stderr'),hash=bytes=>crypto.createHash('sha256').update(bytes).digest('hex');
fs.mkdirSync(scratch);fs.mkdirSync(path.join(scratch,'module'));
const files=[...prior.files], fixture=fs.readFileSync(path.join(out,'standard-stderr-control_test.go.txt')),dispatcher=fs.readFileSync(path.join(out,'standard-stderr-dispatcher.cjs'));
for(const graph of [prior.original,prior.final,prior.current]){const file=path.join(graph,'tools/gomad3/target/task9_standard_stderr_control_test.go');fs.writeFileSync(file,fixture,{flag:'wx'});files.push({path:file,sha256:hash(fixture)});}
for(const [relative,bytes,mode]of [['go',dispatcher,0o700],['module/go.mod',fs.readFileSync(path.join(prior.scratch,'module/go.mod')),0o600]]){const file=path.join(scratch,relative);fs.writeFileSync(file,bytes,{mode,flag:'wx'});files.push({path:file,sha256:hash(bytes)});}
for(const size of [65535,65536,65537,4*1024*1024]){const bytes=Buffer.alloc(size);for(let i=0;i<size;i++)bytes[i]=97+i%26;const file=path.join(scratch,'payload-'+size);fs.writeFileSync(file,bytes,{flag:'wx'});files.push({path:file,sha256:hash(bytes)});}
const metadata={scratch,original:prior.original,final:prior.final,current:prior.current,files,fixture_sha256:hash(fixture),dispatcher_sha256:hash(dispatcher),literal:'Each payload byte i is ASCII a+(i%26); real checked full stderr write followed by exit7. No stdout, compiler or target launch.',scope:'Matched additive private standard-inventory entry, actual full cause/Stderr/ProcessState observations; no source/test replacement.'};
fs.writeFileSync(path.join(out,'standard-stderr-inputs.json'),JSON.stringify(metadata,null,2)+'\n',{flag:'wx'});
console.log(JSON.stringify({scratch,fixture_sha256:metadata.fixture_sha256,dispatcher_sha256:metadata.dispatcher_sha256,files:files.length}));
