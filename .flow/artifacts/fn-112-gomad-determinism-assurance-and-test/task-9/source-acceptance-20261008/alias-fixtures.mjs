import {mkdirSync,writeFileSync} from 'node:fs';
import {resolve} from 'node:path';
import {out,write,sha} from './capture.mjs';
const directories=['artifact','choice','deterministicio','qualification','record','runner','target','toolchain','upgrade','world'],manifest=[];
for(const kind of ['valid','exported-alias'])for(const directory of directories){
 const dir=resolve(out,'alias-fixtures',kind,directory);mkdirSync(dir,{recursive:true});
 const body=`package ${directory}\ntype PublicValue string\ntype privateAlias = string\n`+(kind==='exported-alias'&&directory==='runner'?'type PublicAlias = string\n':'');
 const path=resolve(dir,'fixture.go');writeFileSync(path,body,{flag:'wx'});manifest.push({path,sha256:sha(body)});
}
write('alias-fixture-inputs.json',{directories,files:manifest});
