import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';
const out=path.dirname(new URL(import.meta.url).pathname),roots=['/tmp/fn109-task9-final-process.Dirh0T','/tmp/fn109-task9-final-selection.fYRXEO'];
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
function walk(root,relative=''){
 const file=path.join(root,relative),stat=fs.lstatSync(file),value={path:relative||'.',uid:stat.uid,gid:stat.gid,mode:stat.mode&0o7777,size:stat.size};
 if(stat.isSymbolicLink())return [{...value,type:'symlink',target:fs.readlinkSync(file)}];
 if(stat.isFile())return [{...value,type:'file',sha256:hash(fs.readFileSync(file))}];
 if(stat.isDirectory())return [{...value,type:'directory'},...fs.readdirSync(file).sort().flatMap(name=>walk(root,path.join(relative,name)))];
 throw Error('unexpected special file '+file);
}
const process=spawnSync('ps',['-eo','pid,ppid,args'],{encoding:'utf8'});if(process.status!==0)throw Error(process.stderr);
const result={roots:roots.map(root=>({root,entries:walk(root),live_command_mentions:process.stdout.split('\n').filter(line=>line.includes(root))})),historical_receipts:['../adapter-integration-20261005/implementation/process-probe-final/receipt.json','../adapter-integration-20261005/implementation/source-selection-final/receipt.json'],process_snapshot_sha256:hash(Buffer.from(process.stdout)),scope:'Read-only ownership/input inventory; no relocation or deletion performed. No current terminal handle uses these exact task9-owned historical roots.'};
fs.writeFileSync(path.join(out,'tmp-ownership.json'),JSON.stringify(result,null,2)+'\n',{flag:'wx'});
console.log(JSON.stringify(result.roots.map(x=>({root:x.root,entries:x.entries.length,bytes:x.entries.filter(y=>y.type==='file').reduce((n,y)=>n+y.size,0),live_mentions:x.live_command_mentions.length}))));
