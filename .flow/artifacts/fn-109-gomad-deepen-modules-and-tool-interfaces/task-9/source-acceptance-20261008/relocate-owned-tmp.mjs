import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';
const out=path.dirname(new URL(import.meta.url).pathname),owner=JSON.parse(fs.readFileSync(path.join(out,'tmp-ownership.json'))),destination='/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX/task9-recovered-tmp-20261009';
const allowed=['/tmp/fn109-task9-final-process.Dirh0T','/tmp/fn109-task9-final-selection.fYRXEO'];
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
function inventory(root,relative=''){
 const file=path.join(root,relative),stat=fs.lstatSync(file),value={path:relative||'.',uid:stat.uid,gid:stat.gid,mode:stat.mode&0o7777,size:stat.size};
 if(stat.isSymbolicLink())return [{...value,type:'symlink',target:fs.readlinkSync(file)}];
 if(stat.isFile())return [{...value,type:'file',sha256:hash(fs.readFileSync(file))}];
 if(stat.isDirectory())return [{...value,type:'directory'},...fs.readdirSync(file).sort().flatMap(name=>inventory(root,path.join(relative,name)))];
 throw Error('unexpected special file');
}
const comparable=entries=>entries.map(entry=>{const copy={...entry};if(copy.type==='directory')delete copy.size;return copy;});
if(JSON.stringify(owner.roots.map(x=>x.root))!==JSON.stringify(allowed))throw Error('scope mismatch');
const resume=process.argv[2]==='resume-own-copy';
if(fs.existsSync(destination)!==resume)throw Error('destination state mismatch');
const commands=spawnSync('ps',['-eo','pid,ppid,args'],{encoding:'utf8'});if(commands.status!==0)throw Error(commands.stderr);
for(const entry of owner.roots){if(fs.realpathSync(entry.root)!==entry.root||JSON.stringify(inventory(entry.root))!==JSON.stringify(entry.entries)||commands.stdout.split('\n').some(line=>line.includes(entry.root)))throw Error('ownership or terminal state changed '+entry.root);}
if(!resume)fs.mkdirSync(destination,{mode:0o700});
if(resume&&JSON.stringify(fs.readdirSync(destination).sort())!==JSON.stringify([path.basename(allowed[0])]))throw Error('unexpected partial destination entry');
const moves=[];
for(const entry of owner.roots){
 const target=path.join(destination,path.basename(entry.root));
 if(fs.existsSync(target)){
  if(!resume||entry.root!==allowed[0])throw Error('collision');
  const candidate=inventory(target).map(value=>{const copy={...value};if(copy.type==='directory')delete copy.mode;return copy;}),expected=entry.entries.map(value=>{const copy={...value};if(copy.type==='directory')delete copy.mode;return copy;});
  if(JSON.stringify(comparable(candidate))!==JSON.stringify(comparable(expected)))throw Error('partial copy identity mismatch');
 }else fs.cpSync(entry.root,target,{recursive:true,dereference:false,verbatimSymlinks:true,preserveTimestamps:true,errorOnExist:true,force:false});
 for(const value of entry.entries.filter(value=>value.type==='directory'))fs.chmodSync(path.join(target,value.path),value.mode);
 const copied=inventory(target);
 if(JSON.stringify(comparable(copied))!==JSON.stringify(comparable(entry.entries)))throw Error('copied identity mismatch');
 moves.push({source:entry.root,destination:target,entries:copied,verified_before_retirement:true,recoverable:true});
}
fs.writeFileSync(path.join(out,'tmp-relocation-copied.json'),JSON.stringify({moves},null,2)+'\n',{flag:'wx'});
for(const move of moves){
 if(!allowed.includes(move.source)||JSON.stringify(comparable(inventory(move.source)))!==JSON.stringify(comparable(inventory(move.destination))))throw Error('pre-retirement identity changed');
 fs.rmSync(move.source,{recursive:true});
 if(fs.existsSync(move.source)||JSON.stringify(comparable(inventory(move.destination)))!==JSON.stringify(comparable(move.entries)))throw Error('post-relocation mismatch');
}
const space=spawnSync('df',['-B1','/tmp'],{encoding:'utf8'});
const result={moves,post_move_space:space.stdout,process_snapshot_sha256:hash(Buffer.from(commands.stdout)),scope:'Only two explicitly admitted task9-owned historical scratch roots retired after full copy/inventory verification. Historical receipts untouched; recovery remains at listed destinations.'};
fs.writeFileSync(path.join(out,'tmp-relocation.json'),JSON.stringify(result,null,2)+'\n',{flag:'wx'});console.log(JSON.stringify({moves:moves.map(x=>({source:x.source,destination:x.destination,recoverable:x.recoverable})),space:space.stdout}));
