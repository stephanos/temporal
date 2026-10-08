import {readdirSync,statSync} from 'node:fs';
import {out,read,sha,write} from './capture.mjs';
function walk(path,prefix=''){return readdirSync(path).flatMap(n=>statSync(path+'/'+n).isDirectory()?walk(path+'/'+n,prefix+n+'/'):[prefix+n]);}
const names=walk(out).filter(n=>!['freeze.json','verification-result.json','verification.json','verification.stdout','verification.stderr'].includes(n)).sort();
write('freeze.json',{algorithm:'sha256',created:new Date().toISOString(),scope:'All task6 source-acceptance files existing at seal; verification-result and its capturing receipt deliberately outside self-referential manifest. No old frozen evidence rewritten.',files:names.map(name=>({name,bytes:read(out+'/'+name).length,sha256:sha(read(out+'/'+name))}))});console.log('Sealed '+names.length+' proof files.');
