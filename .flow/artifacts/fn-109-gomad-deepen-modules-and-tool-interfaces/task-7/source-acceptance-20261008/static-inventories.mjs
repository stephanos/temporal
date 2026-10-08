import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';
const repo='/Users/stephan/Workspace/skunkworks/gomad/temporal',out=path.join(repo,'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-7/source-acceptance-20261008');
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const receipt=JSON.parse(fs.readFileSync(path.join(out,'final-architecture-host-vet-receipt.json')));
if(receipt.exit_code!==0)throw Error('inventory gate not green');
const events=fs.readFileSync(path.join(out,receipt.log),'utf8').split('\n').flatMap(line=>{try{return [JSON.parse(line)];}catch{return [];}}),inventories=[];
for(const [os,arch] of [['darwin','arm64'],['linux','amd64'],['linux','arm64']]){
 const test='TestHostPackageVet/'+os+'/'+arch,output=events.filter(e=>e.Test===test&&e.Output).map(e=>e.Output).join(''),match=output.match(/validated source inventory: (.*)\n/);
 if(!match||!events.some(e=>e.Test===test&&e.Action==='pass'))throw Error('missing nonempty passing inventory '+test);
 const inventory=JSON.parse(match[1]),packages=inventory.Packages.map(p=>p.ImportPath);
 if(!packages.length||packages.some(p=>!/^[a-zA-Z0-9./_-]+$/.test(p)))throw Error('invalid package inventory');
 inventories.push({platform:{os,arch},package_count:packages.length,inventory_sha256:hash(match[1]),inventory,source_manifest:receipt.source_manifest,source_tree_sha256:receipt.source_tree_sha256,vet:{command:'GOOS='+os+' GOARCH='+arch+' go vet -tags test_dep '+packages.join(' '),execution:'new execution inside unchanged TestHostPackageVet; not repeated',parent_receipt:'final-architecture-host-vet-receipt.json',test,exit_code:0,elapsed_seconds:events.find(e=>e.Test===test&&e.Action==='pass').Elapsed},list_command:'cd tools/gomad3 && GOOS='+os+' GOARCH='+arch+' go list -mod=readonly -deps -test -tags test_dep '+packages.join(' ')});
}
fs.writeFileSync(path.join(out,'static-inventories.json'),JSON.stringify(inventories,null,2)+'\n');
for(const inventory of inventories.filter(i=>i.platform.os==='darwin'||i.platform.arch==='amd64')){
 const result=spawnSync('node',[path.join(out,'run.mjs'),'source-list-inventory-'+inventory.platform.os,inventory.list_command],{cwd:repo,stdio:'inherit'});
 if(result.status!==0)process.exit(result.status??1);
}
