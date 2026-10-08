import assert from 'node:assert/strict';
import {readdirSync,statSync} from 'node:fs';
import {resolve} from 'node:path';
import {read,write,out,sha} from './capture.mjs';
const excluded=['manifest.json','terminal-seal.json','manifest-freeze.json','manifest-freeze.stdout','manifest-freeze.stderr','terminal-verification.json','terminal-verification.stdout','terminal-verification.stderr'];
const walk=prefix=>readdirSync(resolve(out,prefix),{withFileTypes:true}).flatMap(e=>{const p=prefix?prefix+'/'+e.name:e.name;assert(!e.isSymbolicLink(),p);return e.isDirectory()?walk(p):[p];});
if(process.argv.includes('--terminal')){
 const manifest=JSON.parse(read(out+'/manifest.json'));for(const f of manifest.files)assert.equal(sha(read(out+'/'+f.path)),f.sha256,f.path);
 const terminal=['manifest-freeze','terminal-verification'].flatMap(l=>['.json','.stdout','.stderr'].map(s=>l+s)).map(path=>({path,sha256:sha(read(out+'/'+path)),bytes:read(out+'/'+path).length}));
 const receipt=JSON.parse(read(out+'/terminal-verification.json'));assert.equal(receipt.exit,0);
 write('terminal-seal.json',{manifest_sha256:sha(read(out+'/manifest.json')),manifest_files:manifest.files.length,terminal_files:terminal,lane:'RELEASED',commands_running:0,delegates_running:0,review_verdict:null,native:false});
 console.log(JSON.stringify({terminal_seal_sha256:sha(read(out+'/terminal-seal.json')),manifest_sha256:sha(read(out+'/manifest.json')),lane:'RELEASED',commands_running:0}));
}else{
 const files=walk('').filter(p=>!excluded.includes(p)).sort().map(path=>({path,sha256:sha(read(out+'/'+path)),bytes:statSync(resolve(out,path)).size}));
 write('manifest.json',{files,exclusions:excluded,exclusion_reason:'Manifest/self and two final capture triplets are bound by terminal-seal.json; seal own hash is returned to conductor.',complete_worker_directory_only:true});
 console.log(JSON.stringify({files:files.length,bytes:files.reduce((n,f)=>n+f.bytes,0),sha256:sha(read(out+'/manifest.json'))}));
}
