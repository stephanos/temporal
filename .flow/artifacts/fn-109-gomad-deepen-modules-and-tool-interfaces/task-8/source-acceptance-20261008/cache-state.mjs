import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
const out=path.dirname(new URL(import.meta.url).pathname),hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const controls=JSON.parse(fs.readFileSync(path.join(out,'source-controls.json')));
const stat=p=>{const s=fs.statSync(p);return {path:p,realpath:fs.realpathSync(p),device:s.dev,inode:s.ino,mode:s.mode,size:s.size}};
const walk=dir=>fs.readdirSync(dir,{withFileTypes:true}).flatMap(e=>e.isDirectory()?walk(path.join(dir,e.name)):e.isFile()?[path.join(dir,e.name)]:[]);
const rows={};for(const side of ['original','current']){
 const root=path.join(controls[side],'.toolchain');
 rows[side]={root:stat(root),files:walk(root).sort().map(p=>({...stat(p),sha256:hash(fs.readFileSync(p))}))};
}
const cache='/Users/stephan/Workspace/skunkworks/.gomad-fn1129-cache-EseD1r';
fs.writeFileSync(path.join(out,'final-cache-state.json'),JSON.stringify({existing_gocache:stat(cache),controls:rows,scope:'Post-gate state only. Adapter/build caches are generated outputs, not immutable source/tool inputs. No cache deletion or native toolchain creation.',native:false},null,2)+'\n',{flag:'wx'});
