import assert from 'node:assert/strict';
import {readdirSync,writeFileSync} from 'node:fs';
import {spawnSync} from 'node:child_process';
import {root,out} from './capture.mjs';
const results=[];
for(const name of readdirSync(out).sort()){
 const argv=['git','diff','--no-index','--check','/dev/null',out+'/'+name],r=spawnSync(argv[0],argv.slice(1),{cwd:root});
 results.push({path:name,argv,cwd:root,exit:r.status,stdout:r.stdout.toString(),stderr:r.stderr.toString()});
 assert.equal(r.stdout.length+r.stderr.length,0,'whitespace finding in '+name);
 assert([0,1].includes(r.status),'unexpected diff check failure '+name);
}
const r=spawnSync('git',['diff','--cached','--check'],{cwd:root});results.push({argv:['git','diff','--cached','--check'],cwd:root,exit:r.status,stdout:r.stdout.toString(),stderr:r.stderr.toString()});assert.equal(r.status,0);
writeFileSync(out+'/artifact-whitespace-results.json',JSON.stringify(results,null,2)+'\n',{flag:'wx'});console.log(JSON.stringify({files:results.length-1,whitespace_findings:0,no_staging:true}));
