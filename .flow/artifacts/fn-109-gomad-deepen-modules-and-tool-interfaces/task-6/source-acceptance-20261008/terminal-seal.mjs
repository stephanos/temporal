import {readdirSync,statSync} from 'node:fs';
import {out,read,sha,write} from './capture.mjs';
function walk(path,prefix=''){return readdirSync(path).flatMap(n=>statSync(path+'/'+n).isDirectory()?walk(path+'/'+n,prefix+n+'/'):[prefix+n]);}
const names=walk(out).filter(n=>n!=='terminal-seal.json').sort();
write('terminal-seal.json',{algorithm:'sha256',created:new Date().toISOString(),scope:'Every terminal worker artifact, including first freeze, verification result/receipt, supplemental source/command controls and fixtures; self excluded to avoid circularity. Final seal checked read-only after generation.',files:names.map(name=>({name,bytes:read(out+'/'+name).length,sha256:sha(read(out+'/'+name))}))});console.log('Terminal seal covers '+names.length+' files.');
